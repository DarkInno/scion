package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// sleepCheck builds a CustomCheck that sleeps for d, respecting ctx.
func sleepCheck(name string, d time.Duration) *CustomCheck {
	c, _ := NewCustomCheck(name, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	})
	return c
}

// freeAddr returns a "host:port" that is guaranteed to refuse connections.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

// localHTTPServer starts an httptest.Server and returns it together with the
// allow-list entry needed to reach it (its 127.0.0.1 address).
func localHTTPServer(t *testing.T, h http.Handler) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	// srv.URL looks like http://127.0.0.1:port
	u := srv.URL
	host, _, err := net.SplitHostPort(strings.TrimPrefix(u, "http://"))
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv, host
}

func TestNewDefaults(t *testing.T) {
	hc := New()
	if hc.timeout != DefaultTimeout {
		t.Fatalf("timeout = %v, want %v", hc.timeout, DefaultTimeout)
	}
	if hc.cacheTTL != DefaultCacheTTL {
		t.Fatalf("cacheTTL = %v, want %v", hc.cacheTTL, DefaultCacheTTL)
	}
	if len(hc.Checks()) != 0 {
		t.Fatalf("expected zero checks")
	}
}

func TestWithOptions(t *testing.T) {
	hc := New(WithTimeout(time.Second), WithCacheTTL(0))
	if hc.timeout != time.Second {
		t.Fatalf("timeout = %v", hc.timeout)
	}
	if hc.cacheTTL != 0 {
		t.Fatalf("cacheTTL = %v", hc.cacheTTL)
	}
	// WithTimeout(0) is a no-op: keeps the default.
	hc2 := New(WithTimeout(0))
	if hc2.timeout != DefaultTimeout {
		t.Fatalf("WithTimeout(0) should be ignored, got %v", hc2.timeout)
	}
}

func TestAddCheckValidation(t *testing.T) {
	hc := New()
	if err := hc.AddCheck(nil); err == nil {
		t.Fatal("nil check must error")
	}
	// typed-nil (a constructor that returned nil due to an invalid name) must
	// be rejected without panicking.
	typedNil, _ := NewCustomCheck("", func(ctx context.Context) error { return nil })
	if err := hc.AddCheck(typedNil); err == nil {
		t.Fatal("typed-nil check must error")
	}
	// Exercise AddCheck's own validation via a direct Check implementation,
	// bypassing constructor validation.
	cases := []struct {
		name string
		ok   bool
	}{
		{"", false}, // empty
		{strings.Repeat("a", MaxNameLen+1), false}, // too long
		{"bad\r\nname", false},                     // CRLF
		{"ok", true},                               // valid
		{"ok", false},                              // duplicate
	}
	for _, c := range cases {
		err := hc.AddCheck(rawCheck{name: c.name})
		if (err == nil) != c.ok {
			t.Errorf("AddCheck(rawCheck{name:%q}) err=%v, want ok=%v", c.name, err, c.ok)
		}
	}
	if len(hc.Checks()) != 1 {
		t.Fatalf("checks = %d, want 1", len(hc.Checks()))
	}
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"db", true},
		{"", false},
		{strings.Repeat("a", MaxNameLen), true},
		{strings.Repeat("a", MaxNameLen+1), false},
		{"a\rb", false},
		{"a\nb", false},
		{"a\r\nb", false},
	}
	for _, c := range cases {
		err := ValidateName(c.name)
		if (err == nil) != c.ok {
			t.Errorf("ValidateName(%q) err=%v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestValidateURL(t *testing.T) {
	good := []string{
		"http://example.com/",
		"https://example.com:443/health",
		"http://8.8.8.8/",
		"https://1.1.1.1/ping",
	}
	for _, u := range good {
		if err := validateURL(u, nil); err != nil {
			t.Errorf("validateURL(%q) = %v, want nil", u, err)
		}
	}
	bad := []string{
		"",
		"http://localhost/",
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://172.16.0.1/",
		"http://172.31.255.255/",
		"http://169.254.1.1/",
		"http://0.0.0.0/",
		"http://[::1]/",
		"http://[fc00::1]/",
		"http://[fe80::1]/",
		"ftp://example.com/",
		"http://",
		"http://example.com/" + strings.Repeat("a", MaxURLLen), // too long
		"http://example.com/\r\nX-Inject: 1",
	}
	for _, u := range bad {
		if err := validateURL(u, nil); err == nil {
			t.Errorf("validateURL(%q) = nil, want error", u)
		}
	}
	// allowed private IP bypasses the block.
	if err := validateURL("http://127.0.0.1/", []string{"127.0.0.1"}); err != nil {
		t.Errorf("allowed 127.0.0.1: %v", err)
	}
	if err := validateURL("http://10.0.0.5/", []string{"10.0.0.0/8"}); err != nil {
		t.Errorf("allowed CIDR: %v", err)
	}
}

func TestIsPrivateIP(t *testing.T) {
	privates := []string{"127.0.0.1", "10.1.2.3", "192.168.0.1", "172.16.5.5", "169.254.0.1", "0.0.0.0", "::1", "fc00::1", "fe80::1"}
	for _, s := range privates {
		if !isPrivateIP(net.ParseIP(s), nil) {
			t.Errorf("isPrivateIP(%s) = false, want true", s)
		}
	}
	public := []string{"8.8.8.8", "1.1.1.1", "203.0.113.1"}
	for _, s := range public {
		if isPrivateIP(net.ParseIP(s), nil) {
			t.Errorf("isPrivateIP(%s) = true, want false", s)
		}
	}
	// allow-list
	if isPrivateIP(net.ParseIP("127.0.0.1"), []string{"127.0.0.1"}) {
		t.Error("allowed 127.0.0.1 flagged private")
	}
	if isPrivateIP(net.ParseIP("10.0.0.1"), []string{"10.0.0.0/8"}) {
		t.Error("allowed CIDR flagged private")
	}
}

func TestRunChecksEmpty(t *testing.T) {
	hc := New(WithCacheTTL(0))
	status, results := hc.RunChecks(context.Background())
	if status != StatusHealthy {
		t.Fatalf("status = %s, want healthy", status)
	}
	if len(results) != 0 {
		t.Fatalf("results = %v, want empty", results)
	}
}

func TestRunChecksAllPass(t *testing.T) {
	srv, host := localHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	tcpAddr := ln.Addr().String()

	httpChk, err := NewHTTPCheck("api", srv.URL, WithAllowedPrivateIPs(host))
	if err != nil {
		t.Fatalf("NewHTTPCheck: %v", err)
	}
	tcpChk, err := NewTCPCheck("redis", tcpAddr)
	if err != nil {
		t.Fatalf("NewTCPCheck: %v", err)
	}
	customChk, _ := NewCustomCheck("cache", func(ctx context.Context) error { return nil })

	hc := New(WithCacheTTL(0), WithTimeout(2*time.Second))
	if err := hc.AddCheck(httpChk); err != nil {
		t.Fatal(err)
	}
	if err := hc.AddCheck(tcpChk); err != nil {
		t.Fatal(err)
	}
	if err := hc.AddCheck(customChk); err != nil {
		t.Fatal(err)
	}

	status, results := hc.RunChecks(context.Background())
	if status != StatusHealthy {
		t.Fatalf("status = %s, want healthy", status)
	}
	for _, name := range []string{"api", "redis", "cache"} {
		if results[name].Status != StatusPass {
			t.Errorf("%s = %+v, want pass", name, results[name])
		}
	}
}

func TestRunChecksMixedFail(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(time.Second))
	good, _ := NewCustomCheck("good", func(ctx context.Context) error { return nil })
	bad, _ := NewCustomCheck("bad", func(ctx context.Context) error { return errors.New("boom") })
	hc.AddCheck(good)
	hc.AddCheck(bad)

	status, results := hc.RunChecks(context.Background())
	if status != StatusUnhealthy {
		t.Fatalf("status = %s, want unhealthy", status)
	}
	if results["good"].Status != StatusPass {
		t.Errorf("good = %+v", results["good"])
	}
	if results["bad"].Status != StatusFail {
		t.Errorf("bad = %+v", results["bad"])
	}
	if !strings.Contains(results["bad"].Error, "boom") {
		t.Errorf("bad error = %q", results["bad"].Error)
	}
}

func TestRunChecksConcurrentTiming(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(2*time.Second))
	for i := 0; i < 4; i++ {
		hc.AddCheck(sleepCheck(fmt.Sprintf("c-%d", i), 80*time.Millisecond))
	}
	start := time.Now()
	status, _ := hc.RunChecks(context.Background())
	elapsed := time.Since(start)
	if status != StatusHealthy {
		t.Fatalf("status = %s", status)
	}
	// 4 checks * 80ms serially = 320ms; concurrently should be well under that.
	if elapsed > 250*time.Millisecond {
		t.Fatalf("checks not concurrent: took %v", elapsed)
	}
}

func TestCacheHitAndExpiry(t *testing.T) {
	var calls int32
	hc := New(WithCacheTTL(80*time.Millisecond), WithTimeout(time.Second))
	hc.AddCheck(mustCustom(t, "c", func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}))

	// first call runs the check
	hc.RunChecks(context.Background())
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
	// second call within TTL returns cached -> no extra call
	hc.RunChecks(context.Background())
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls = %d, want 1 (cached)", got)
	}
	// after TTL expires, runs again
	time.Sleep(120 * time.Millisecond)
	hc.RunChecks(context.Background())
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2 (after expiry)", got)
	}
}

func TestCacheDisabled(t *testing.T) {
	var calls int32
	hc := New(WithCacheTTL(0), WithTimeout(time.Second))
	hc.AddCheck(mustCustom(t, "c", func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}))
	hc.RunChecks(context.Background())
	hc.RunChecks(context.Background())
	hc.RunChecks(context.Background())
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("calls = %d, want 3 (no cache)", got)
	}
}

func TestTimeoutContextCancel(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(50*time.Millisecond))
	hc.AddCheck(sleepCheck("slow", 5*time.Second))

	start := time.Now()
	status, results := hc.RunChecks(context.Background())
	elapsed := time.Since(start)

	if status != StatusUnhealthy {
		t.Fatalf("status = %s, want unhealthy", status)
	}
	if results["slow"].Status != StatusFail {
		t.Fatalf("slow = %+v, want fail", results["slow"])
	}
	// The check respects ctx, so it should return promptly after the timeout.
	if elapsed > 500*time.Millisecond {
		t.Fatalf("timeout not respected: took %v", elapsed)
	}
}

func TestTimeoutViaParentContext(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(10*time.Second))
	hc.AddCheck(sleepCheck("slow", 5*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	status, results := hc.RunChecks(ctx)
	elapsed := time.Since(start)
	if status != StatusUnhealthy {
		t.Fatalf("status = %s", status)
	}
	if results["slow"].Status != StatusFail {
		t.Fatalf("slow = %+v", results["slow"])
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("parent ctx timeout not respected: %v", elapsed)
	}
}

func TestHTTPCheckPassAndFail(t *testing.T) {
	srv, host := localHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	pass, err := NewHTTPCheck("ok", srv.URL+"/good", WithAllowedPrivateIPs(host))
	if err != nil {
		t.Fatalf("NewHTTPCheck: %v", err)
	}
	if r := pass.Execute(context.Background()); r.Status != StatusPass {
		t.Fatalf("pass = %+v", r)
	}

	fail, err := NewHTTPCheck("bad", srv.URL+"/bad", WithAllowedPrivateIPs(host))
	if err != nil {
		t.Fatalf("NewHTTPCheck: %v", err)
	}
	r := fail.Execute(context.Background())
	if r.Status != StatusFail {
		t.Fatalf("fail status = %+v", r)
	}
	if !strings.Contains(r.Error, "500") {
		t.Errorf("fail error = %q", r.Error)
	}

	// connection refused (port closed) -> fail
	closed, err := NewHTTPCheck("closed", "http://"+host+":1/", WithAllowedPrivateIPs(host))
	if err != nil {
		t.Fatalf("NewHTTPCheck closed: %v", err)
	}
	if r := closed.Execute(context.Background()); r.Status != StatusFail {
		t.Fatalf("closed = %+v, want fail", r)
	}
}

func TestHTTPCheckMethod(t *testing.T) {
	var seen string
	srv, host := localHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	c, err := NewHTTPCheck("head", srv.URL, WithAllowedPrivateIPs(host), WithHTTPMethod(http.MethodHead))
	if err != nil {
		t.Fatal(err)
	}
	if r := c.Execute(context.Background()); r.Status != StatusPass {
		t.Fatalf("status = %+v", r)
	}
	if seen != http.MethodHead {
		t.Fatalf("method = %q, want HEAD", seen)
	}
}

func TestTCPCheckPassAndFail(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	pass, err := NewTCPCheck("up", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if r := pass.Execute(context.Background()); r.Status != StatusPass {
		t.Fatalf("pass = %+v", r)
	}

	fail, err := NewTCPCheck("down", freeAddr(t))
	if err != nil {
		t.Fatal(err)
	}
	if r := fail.Execute(context.Background()); r.Status != StatusFail {
		t.Fatalf("fail = %+v, want fail", r)
	}
}

func TestTCPCheckValidation(t *testing.T) {
	if _, err := NewTCPCheck("", "127.0.0.1:80"); err == nil {
		t.Fatal("empty name must error")
	}
	if _, err := NewTCPCheck("ok", ""); err == nil {
		t.Fatal("empty addr must error")
	}
	if _, err := NewTCPCheck("ok", "no-port"); err == nil {
		t.Fatal("invalid addr must error")
	}
	if _, err := NewTCPCheck("ok", "127.0.0.1:80\r\n"); err == nil {
		t.Fatal("CRLF addr must error")
	}
}

func TestCustomCheck(t *testing.T) {
	pass, _ := NewCustomCheck("ok", func(ctx context.Context) error { return nil })
	if r := pass.Execute(context.Background()); r.Status != StatusPass {
		t.Fatalf("pass = %+v", r)
	}
	fail, _ := NewCustomCheck("bad", func(ctx context.Context) error { return errors.New("nope") })
	if r := fail.Execute(context.Background()); r.Status != StatusFail {
		t.Fatalf("fail = %+v", r)
	}
	if !strings.Contains(fail.Execute(context.Background()).Error, "nope") {
		t.Error("missing error text")
	}
	if _, err := NewCustomCheck("ok", nil); err == nil {
		t.Fatal("nil fn must error")
	}
}

func TestLiveness(t *testing.T) {
	hc := New()
	if r := hc.Liveness(); r.Status != StatusPass {
		t.Fatalf("liveness = %+v", r)
	}
}

func TestHandlersHealthy(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(time.Second))
	hc.AddCheck(mustCustom(t, "db", func(ctx context.Context) error { return nil }))
	h := NewHealthHandler(hc)

	// liveness
	rec := httptest.NewRecorder()
	h.Liveness(rec, httptest.NewRequest(http.MethodGet, "/live", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("liveness code = %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != StatusHealthy {
		t.Errorf("liveness status = %v", body["status"])
	}
	if _, ok := body["checks"]; ok {
		t.Errorf("liveness should not include checks, got %v", body["checks"])
	}

	// readiness -> ready
	rec = httptest.NewRecorder()
	h.Readiness(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("readiness code = %d", rec.Code)
	}
	body = decode(t, rec.Body.Bytes())
	if body["status"] != StatusReady {
		t.Errorf("readiness status = %v", body["status"])
	}
	if _, ok := body["checks"].(map[string]interface{})["db"]; !ok {
		t.Errorf("readiness missing db check: %v", body["checks"])
	}

	// health -> healthy
	rec = httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health code = %d", rec.Code)
	}
	body = decode(t, rec.Body.Bytes())
	if body["status"] != StatusHealthy {
		t.Errorf("health status = %v", body["status"])
	}
}

func TestHandlersUnhealthy(t *testing.T) {
	hc := New(WithCacheTTL(0), WithTimeout(time.Second))
	hc.AddCheck(mustCustom(t, "db", func(ctx context.Context) error { return errors.New("down") }))
	h := NewHealthHandler(hc)

	rec := httptest.NewRecorder()
	h.Readiness(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness code = %d, want 503", rec.Code)
	}
	body := decode(t, rec.Body.Bytes())
	if body["status"] != StatusNotReady {
		t.Errorf("readiness status = %v", body["status"])
	}

	rec = httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("health code = %d, want 503", rec.Code)
	}
	body = decode(t, rec.Body.Bytes())
	if body["status"] != StatusUnhealthy {
		t.Errorf("health status = %v", body["status"])
	}
}

func TestHandlerContentType(t *testing.T) {
	hc := New(WithCacheTTL(0))
	h := NewHealthHandler(hc)
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	if v := rec.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", v)
	}
}

func TestResultJSON(t *testing.T) {
	pass := PassResult(5 * time.Millisecond)
	b, _ := json.Marshal(pass)
	if !strings.Contains(string(b), `"status":"pass"`) {
		t.Errorf("pass json = %s", b)
	}
	if !strings.Contains(string(b), `"latency_ms":5`) {
		t.Errorf("pass json missing latency: %s", b)
	}
	if strings.Contains(string(b), `"error"`) {
		t.Errorf("pass json should not have error: %s", b)
	}

	fail := FailResult(errors.New("x"))
	b, _ = json.Marshal(fail)
	if !strings.Contains(string(b), `"status":"fail"`) {
		t.Errorf("fail json = %s", b)
	}
	if !strings.Contains(string(b), `"error":"x"`) {
		t.Errorf("fail json missing error: %s", b)
	}
	if strings.Contains(string(b), `"latency_ms"`) {
		t.Errorf("fail json should not have latency: %s", b)
	}
}

// rawCheck is a minimal Check used to exercise AddCheck validation directly,
// bypassing the constructors' own name validation.
type rawCheck struct {
	name string
}

func (r rawCheck) Name() string                       { return r.name }
func (r rawCheck) Execute(ctx context.Context) Result { return PassResult(0) }

func mustCustom(t *testing.T, name string, fn CustomFunc) *CustomCheck {
	t.Helper()
	c, err := NewCustomCheck(name, fn)
	if err != nil {
		t.Fatalf("NewCustomCheck(%q): %v", name, err)
	}
	return c
}

func decode(t *testing.T, b []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return m
}
