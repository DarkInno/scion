package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const maxProxyCount = 10

// ClientIP extracts the client IP address from the request.
// It first checks the context (set by TrustedProxy middleware).
// If not found, it falls back to RemoteAddr.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey).(string); ok && ip != "" {
		return ip
	}
	return remoteAddrIP(r)
}

// proxyConfig is an internal parsed version of ProxyOptions.
// CIDR strings are pre-parsed into net.IPNet at init time to avoid
// repeated parsing on every request.
type proxyConfig struct {
	nets       []*net.IPNet
	proxyCount int
}

// parseProxyConfig parses CIDR strings once and caches the result.
func parseProxyConfig(opts ProxyOptions) proxyConfig {
	if len(opts.TrustedProxies) == 0 && opts.ProxyCount <= 0 {
		return proxyConfig{}
	}
	count := opts.ProxyCount
	if count > maxProxyCount {
		count = maxProxyCount
	}

	nets := make([]*net.IPNet, 0, len(opts.TrustedProxies))
	for _, cidr := range opts.TrustedProxies {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		nets = append(nets, ipNet)
	}

	return proxyConfig{nets: nets, proxyCount: count}
}

// ClientIPWithOptions extracts the client IP using custom proxy options.
// If the extracted IP is not a valid IP address, falls back to RemoteAddr.
func ClientIPWithOptions(r *http.Request, opts ProxyOptions) string {
	if len(opts.TrustedProxies) == 0 && opts.ProxyCount <= 0 {
		return remoteAddrIP(r)
	}

	cfg := parseProxyConfig(opts)
	ip := clientIPFromConfig(r, cfg)
	if net.ParseIP(ip) == nil {
		// XFF contained garbage; fall back to RemoteAddr.
		return remoteAddrIP(r)
	}
	return ip
}

// stripPort removes a port suffix from an IP address (e.g., "192.168.1.1:8080" -> "192.168.1.1").
// If no port is present, the original string is returned.
func stripPort(ip string) string {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		return ip
	}
	return host
}

// clientIPFromConfig does the actual IP extraction using a pre-parsed config.
// SECURITY: If no proxy config is provided (proxyCount==0 and no CIDR nets),
// X-Forwarded-For is NOT trusted and RemoteAddr is returned directly.
// This prevents IP spoofing when the middleware is used without configuration.
func clientIPFromConfig(r *http.Request, cfg proxyConfig) string {
	// No proxy configuration = do not trust X-Forwarded-For at all.
	if cfg.proxyCount == 0 && len(cfg.nets) == 0 {
		return remoteAddrIP(r)
	}

	// Try X-Forwarded-For first.
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		// Fallback to X-Real-IP.
		xff = r.Header.Get("X-Real-IP")
	}
	if xff == "" {
		return remoteAddrIP(r)
	}

	// X-Forwarded-For: client, proxy1, proxy2, ..., proxyN
	ips := strings.Split(xff, ",")

	// ProxyCount mode.
	if cfg.proxyCount > 0 {
		idx := len(ips) - cfg.proxyCount - 1
		if idx >= 0 && idx < len(ips) {
			return stripPort(strings.TrimSpace(ips[idx]))
		}
		if len(ips) > 0 {
			return stripPort(strings.TrimSpace(ips[0]))
		}
		return remoteAddrIP(r)
	}

	// CIDR whitelist mode: from right to left, find the first non-trusted IP.
	for i := len(ips) - 1; i >= 0; i-- {
		ip := stripPort(strings.TrimSpace(ips[i]))
		if !isTrustedNets(ip, cfg.nets) {
			return ip
		}
	}

	return remoteAddrIP(r)
}

// TrustedProxy returns a middleware that extracts the real client IP
// from X-Forwarded-For / X-Real-IP and stores it in the request context.
// The proxy config is pre-parsed once at middleware creation time.
func TrustedProxy(opts ...ProxyOptions) func(http.Handler) http.Handler {
	var opt ProxyOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	cfg := parseProxyConfig(opt)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIPFromConfig(r, cfg)
			if net.ParseIP(ip) == nil {
				ip = remoteAddrIP(r)
			}
			ctx := context.WithValue(r.Context(), clientIPKey, ip)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// remoteAddrIP extracts the IP from r.RemoteAddr, stripping the port.
func remoteAddrIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isTrustedNets checks if an IP falls within any of the given networks.
// This uses pre-parsed net.IPNet and is significantly faster than parsing CIDR strings.
func isTrustedNets(ip string, nets []*net.IPNet) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, ipNet := range nets {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}
	return false
}
