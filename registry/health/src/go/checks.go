package health

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// HTTPCheck
// ---------------------------------------------------------------------------

// HTTPCheck performs an HTTP request and treats a 2xx/3xx response as healthy.
// It enforces an SSRF policy both at construction time and at dial time.
type HTTPCheck struct {
	name    string
	url     string
	method  string
	timeout time.Duration
	allowed []string
	client  *http.Client
}

// HTTPOption configures an HTTPCheck.
type HTTPOption func(*HTTPCheck)

// WithHTTPMethod sets the HTTP method (default GET).
func WithHTTPMethod(m string) HTTPOption {
	return func(h *HTTPCheck) { h.method = m }
}

// WithHTTPTimeout sets the per-request timeout.
func WithHTTPTimeout(t time.Duration) HTTPOption {
	return func(h *HTTPCheck) { h.timeout = t }
}

// WithAllowedPrivateIPs permits specific private IP literals or CIDRs to bypass
// the SSRF protection (e.g. for deliberately probing an internal service).
func WithAllowedPrivateIPs(ips ...string) HTTPOption {
	return func(h *HTTPCheck) { h.allowed = append(h.allowed, ips...) }
}

// NewHTTPCheck creates an HTTPCheck. The URL is validated for length, CRLF and
// SSRF (localhost / private / loopback IPs) at construction time. A private IP
// is only accepted when present in the allow-list.
func NewHTTPCheck(name, rawURL string, opts ...HTTPOption) (*HTTPCheck, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("http check: %w", err)
	}

	h := &HTTPCheck{
		name:    name,
		url:     rawURL,
		method:  http.MethodGet,
		timeout: DefaultTimeout,
	}
	for _, o := range opts {
		o(h)
	}
	// Validate once with the final allow-list so an explicitly allowed
	// private IP is accepted.
	if err := validateURL(rawURL, h.allowed); err != nil {
		return nil, fmt.Errorf("http check: %w", err)
	}

	h.client = newHTTPClient(h.timeout, h.allowed)
	return h, nil
}

// Name returns the check name.
func (h *HTTPCheck) Name() string { return h.name }

// Execute issues the HTTP request bound by ctx and reports the result.
func (h *HTTPCheck) Execute(ctx context.Context) Result {
	start := time.Now()
	method := h.method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, h.url, nil)
	if err != nil {
		return FailResult(err)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return FailResult(err)
	}
	defer resp.Body.Close()
	// Drain the body so the underlying connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	latency := time.Since(start)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return PassResult(latency)
	}
	return Result{Status: StatusFail, Error: fmt.Sprintf("unexpected status code: %d", resp.StatusCode)}
}

// ---------------------------------------------------------------------------
// TCPCheck
// ---------------------------------------------------------------------------

// TCPCheck attempts a TCP dial against an address (host:port). Private IPs are
// allowed because TCP checks are typically aimed at internal services such as
// databases and caches.
type TCPCheck struct {
	name    string
	addr    string
	timeout time.Duration
}

// TCPOption configures a TCPCheck.
type TCPOption func(*TCPCheck)

// WithTCPTimeout sets the dial timeout.
func WithTCPTimeout(t time.Duration) TCPOption {
	return func(c *TCPCheck) { c.timeout = t }
}

// NewTCPCheck creates a TCPCheck.
func NewTCPCheck(name, addr string, opts ...TCPOption) (*TCPCheck, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("tcp check: %w", err)
	}
	if addr == "" {
		return nil, fmt.Errorf("tcp check: address is empty")
	}
	if containsCRLF(addr) {
		return nil, fmt.Errorf("tcp check: address contains CRLF characters")
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return nil, fmt.Errorf("tcp check: invalid address %q: %w", addr, err)
	}
	c := &TCPCheck{name: name, addr: addr, timeout: DefaultTimeout}
	for _, o := range opts {
		o(c)
	}
	if c.timeout <= 0 {
		c.timeout = DefaultTimeout
	}
	return c, nil
}

// Name returns the check name.
func (c *TCPCheck) Name() string { return c.name }

// Execute dials the address bound by ctx and reports the result.
func (c *TCPCheck) Execute(ctx context.Context) Result {
	start := time.Now()
	d := &net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return FailResult(err)
	}
	_ = conn.Close()
	return PassResult(time.Since(start))
}

// ---------------------------------------------------------------------------
// CustomCheck
// ---------------------------------------------------------------------------

// CustomFunc is a user-supplied check. It must respect the provided context so
// that timeouts and cancellations propagate correctly.
type CustomFunc func(ctx context.Context) error

// CustomCheck wraps a user function as a Check.
type CustomCheck struct {
	name string
	fn   CustomFunc
}

// NewCustomCheck creates a CustomCheck.
func NewCustomCheck(name string, fn CustomFunc) (*CustomCheck, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("custom check: %w", err)
	}
	if fn == nil {
		return nil, fmt.Errorf("custom check: function is nil")
	}
	return &CustomCheck{name: name, fn: fn}, nil
}

// Name returns the check name.
func (c *CustomCheck) Name() string { return c.name }

// Execute invokes the user function bound by ctx and reports the result.
func (c *CustomCheck) Execute(ctx context.Context) Result {
	start := time.Now()
	if err := c.fn(ctx); err != nil {
		return FailResult(err)
	}
	return PassResult(time.Since(start))
}

// ---------------------------------------------------------------------------
// SSRF / URL validation
// ---------------------------------------------------------------------------

// validateURL enforces the URL policy: non-empty, length <= MaxURLLen, no CRLF,
// http/https scheme, host present and not a private/loopback address (unless
// explicitly allowed).
func validateURL(rawURL string, allowed []string) error {
	if rawURL == "" {
		return fmt.Errorf("url is empty")
	}
	if len(rawURL) > MaxURLLen {
		return fmt.Errorf("url too long: %d > %d", len(rawURL), MaxURLLen)
	}
	if containsCRLF(rawURL) {
		return fmt.Errorf("url contains CRLF characters")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("ssrf blocked: localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip, allowed) {
			return fmt.Errorf("ssrf blocked: private/loopback ip %s", ip)
		}
	}
	return nil
}

// isPrivateIP reports whether ip is in a non-public range, unless it is on the
// allow-list (literal IP or CIDR).
func isPrivateIP(ip net.IP, allowed []string) bool {
	if ip == nil {
		return false
	}
	if isAllowedIP(ip, allowed) {
		return false
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

// isAllowedIP reports whether ip matches any entry of the allow-list. Each
// entry may be a literal IP or a CIDR range.
func isAllowedIP(ip net.IP, allowed []string) bool {
	for _, a := range allowed {
		if a == "" {
			continue
		}
		if ip.Equal(net.ParseIP(a)) {
			return true
		}
		if _, cidr, err := net.ParseCIDR(a); err == nil {
			if cidr.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// newHTTPClient builds a client whose transport blocks SSRF at dial time by
// resolving the host and refusing to connect to private IPs (DNS-rebinding
// safe). Redirects are not followed so a 3xx cannot redirect the probe to an
// internal address.
func newHTTPClient(timeout time.Duration, allowed []string) *http.Client {
	allowedCopy := append([]string(nil), allowed...)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return ssrfDial(ctx, network, addr, allowedCopy)
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	t := timeout
	if t <= 0 {
		t = DefaultTimeout
	}
	return &http.Client{
		Transport: transport,
		Timeout:   t,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ssrfDial resolves addr's host and refuses private/loopback IPs before
// connecting. For hostnames every resolved address is validated, which closes
// the DNS-rebinding window.
func ssrfDial(ctx context.Context, network, addr string, allowed []string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(host, "localhost") {
		return nil, fmt.Errorf("ssrf blocked: localhost")
	}
	// Literal IP: validate directly.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip, allowed) {
			return nil, fmt.Errorf("ssrf blocked: private/loopback ip %s", ip)
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	// Hostname: resolve and validate every returned address.
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip.IP, allowed) {
			return nil, fmt.Errorf("ssrf blocked: %s resolves to private ip %s", host, ip.IP)
		}
	}
	dialAddr := net.JoinHostPort(ips[0].IP.String(), port)
	return (&net.Dialer{}).DialContext(ctx, network, dialAddr)
}
