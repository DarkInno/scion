package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// ClientIP falls back to RemoteAddr when no proxy middleware is used.
	ip := ClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %q", ip)
	}
}

func TestClientIPProxyCount(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", " 1.1.1.1, 2.2.2.2, 3.3.3.3 ")

	opts := ProxyOptions{ProxyCount: 2}
	ip := ClientIPWithOptions(req, opts)

	if ip != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1 (leftmost after skipping 2 proxies), got %q", ip)
	}
}

func TestClientIPCIDRWhitelist(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 10.0.0.1")

	opts := ProxyOptions{
		TrustedProxies: []string{"10.0.0.0/8"},
	}
	ip := ClientIPWithOptions(req, opts)

	// 10.0.0.1 is trusted (in 10.0.0.0/8), so the first non-trusted is 1.1.1.1.
	if ip != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1, got %q", ip)
	}
}

func TestClientIPNoXFF(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.16.0.5:8080"
	// No X-Forwarded-For or X-Real-IP headers.

	opts := ProxyOptions{TrustedProxies: []string{"10.0.0.0/8"}}
	ip := ClientIPWithOptions(req, opts)

	if ip != "172.16.0.5" {
		t.Errorf("expected RemoteAddr IP 172.16.0.5, got %q", ip)
	}
}

func TestClientIPRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "203.0.113.50")

	// No X-Forwarded-For, so X-Real-IP is used as fallback.
	opts := ProxyOptions{ProxyCount: 1}
	ip := ClientIPWithOptions(req, opts)

	if ip != "203.0.113.50" {
		t.Errorf("expected X-Real-IP value 203.0.113.50, got %q", ip)
	}
}

func TestTrustedProxyMiddleware(t *testing.T) {
	var capturedIP string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIP, _ = r.Context().Value(clientIPKey).(string)
	})

	mw := TrustedProxy(ProxyOptions{
		TrustedProxies: []string{"10.0.0.0/8"},
	})

	handler := mw(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "5.5.5.5, 10.0.0.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedIP != "5.5.5.5" {
		t.Errorf("expected client IP in context to be 5.5.5.5, got %q", capturedIP)
	}
}

func TestProxyCountClamp(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 3.3.3.3, 4.4.4.4, 5.5.5.5")

	// ProxyCount=100 should be clamped to 10 (maxProxyCount).
	opts := ProxyOptions{ProxyCount: 100}
	ip := ClientIPWithOptions(req, opts)

	// With 5 IPs and ProxyCount=10 (clamped): idx = 5 - 10 - 1 = -6 < 0,
	// so it falls through to return ips[0].
	if ip != "1.1.1.1" {
		t.Errorf("expected leftmost IP 1.1.1.1 when ProxyCount exceeds chain length, got %q", ip)
	}
}

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		ip   string
		cidr []string
		want bool
	}{
		{"10.0.0.1", []string{"10.0.0.0/8"}, true},
		{"10.255.255.255", []string{"10.0.0.0/8"}, true},
		{"172.16.0.1", []string{"10.0.0.0/8"}, false},
		{"192.168.1.1", []string{"192.168.0.0/16"}, true},
		{"192.168.1.1", []string{"10.0.0.0/8", "192.168.0.0/16"}, true},
		{"8.8.8.8", []string{"10.0.0.0/8"}, false},
		{"invalid-ip", []string{"10.0.0.0/8"}, false},
		{"10.0.0.1", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			// Pre-parse CIDRs like the middleware does.
			nets := make([]*net.IPNet, 0, len(tt.cidr))
			for _, cidr := range tt.cidr {
				_, ipNet, err := net.ParseCIDR(cidr)
				if err != nil {
					t.Fatalf("invalid CIDR %q: %v", cidr, err)
				}
				nets = append(nets, ipNet)
			}
			got := isTrustedNets(tt.ip, nets)
			if got != tt.want {
				t.Errorf("isTrustedNets(%q, %v) = %v, want %v", tt.ip, tt.cidr, got, tt.want)
			}
		})
	}
}
