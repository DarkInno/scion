package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newCORSHandler(next http.Handler, opts CORSOptions) http.Handler {
	return CORS(opts)(next)
}

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func TestCORSNoOrigin(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Origin header set.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORSExactMatch(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://example.com, got %q", got)
	}
}

func TestCORSWildcardMatch(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://*.example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://sub.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://sub.example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://sub.example.com, got %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         3600,
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("expected Access-Control-Max-Age=3600, got %q", got)
	}
}

func TestCORSNotAllowedOrigin(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORSCredentialsWithWildcard(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for AllowCredentials=true with wildcard origin, but did not panic")
		} else {
			t.Logf("caught expected panic: %v", r)
		}
	}()

	// This should panic at construction time.
	_ = CORS(CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})
}

func TestCORSSubdomainAttack(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://*.example.com"},
	})

	// "attacker-example.com" is NOT a subdomain of "example.com".
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://attacker-example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("subdomain attack: expected no Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORSSuffixAttack(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://*.example.com"},
	})

	// "example.com.attacker.com" should NOT match "*.example.com".
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com.attacker.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("suffix attack: expected no Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORSVaryHeader(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The Vary header should contain "Origin".
	vary := rec.Header().Get("Vary")
	if vary == "" {
		t.Fatal("expected Vary header to be set")
	}
	found := false
	for _, v := range strings.Split(vary, ",") {
		if strings.TrimSpace(v) == "Origin" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Vary to contain Origin, got %q", vary)
	}
}

func TestCORSExposeHeaders(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Another"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Expose-Headers")
	if got != "X-Custom-Header, X-Another" {
		t.Errorf("expected Access-Control-Expose-Headers=X-Custom-Header, X-Another, got %q", got)
	}
}

func TestValidateCORSConfig(t *testing.T) {
	// Valid config should return nil.
	if err := ValidateCORSConfig(CORSOptions{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	}); err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}

	// Invalid config: credentials + wildcard.
	if err := ValidateCORSConfig(CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}); err == nil {
		t.Error("expected error for credentials+wildcard, got nil")
	}
}

func TestCORSMaxAgeClamp(t *testing.T) {
	handler := newCORSHandler(okHandler(), CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         999999,
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Max-Age")
	if got != "86400" {
		t.Errorf("expected Access-Control-Max-Age to be clamped to 86400, got %q", got)
	}
}

func BenchmarkCORSExactMatch(b *testing.B) {
	handler := CORS(CORSOptions{
		AllowedOrigins: []string{"https://example.com", "https://api.example.com"},
	})(okHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkCORSNoOrigin(b *testing.B) {
	handler := CORS(CORSOptions{
		AllowedOrigins: []string{"https://example.com"},
	})(okHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
