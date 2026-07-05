package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDebugMode_Cached(t *testing.T) {
	// Ensure caching works — multiple calls return same value.
	os.Unsetenv("DEBUG")
	resetDebugCache()
	defer resetDebugCache()
	a := DebugMode()
	b := DebugMode()
	if a != b {
		t.Error("expected DebugMode to be cached")
	}
}

func TestDumpRequest_Disabled(t *testing.T) {
	os.Unsetenv("DEBUG")
	resetDebugCache()
	defer resetDebugCache()
	handler := DumpRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestDumpRequest_Enabled(t *testing.T) {
	os.Setenv("DEBUG", "true")
	defer os.Unsetenv("DEBUG")
	resetDebugCache()
	defer resetDebugCache()

	handler := DumpRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestDebugHandler_Disabled(t *testing.T) {
	os.Unsetenv("DEBUG")
	resetDebugCache()
	defer resetDebugCache()
	req := httptest.NewRequest(http.MethodGet, "/debug", nil)
	rec := httptest.NewRecorder()

	DebugHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when debug disabled, got %d", rec.Code)
	}
}

func TestDebugHandler_Localhost(t *testing.T) {
	os.Setenv("DEBUG", "true")
	defer os.Unsetenv("DEBUG")
	resetDebugCache()
	defer resetDebugCache()

	tests := []struct {
		name       string
		remoteAddr string
		status     int
	}{
		{"loopback_ipv4", "127.0.0.1:12345", http.StatusOK},
		{"loopback_ipv4_alt", "127.0.0.1:8080", http.StatusOK},
		{"loopback_ipv6", "[::1]:12345", http.StatusOK},
		{"external_ipv4", "8.8.8.8:12345", http.StatusForbidden},
		{"internal_ipv4", "192.168.1.1:12345", http.StatusForbidden},
		{"external_ipv6", "[2001:db8::1]:12345", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/debug", nil)
			req.RemoteAddr = tt.remoteAddr
			rec := httptest.NewRecorder()

			DebugHandler().ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("remoteAddr=%q: expected status %d, got %d", tt.remoteAddr, tt.status, rec.Code)
			}

			if tt.status == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if !strings.Contains(ct, "application/json") {
					t.Errorf("remoteAddr=%q: expected JSON content type, got %s", tt.remoteAddr, ct)
				}

				var info map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
					t.Errorf("remoteAddr=%q: invalid JSON response: %v", tt.remoteAddr, err)
				}
				if info["debug_mode"] != true {
					t.Errorf("remoteAddr=%q: expected debug_mode=true", tt.remoteAddr)
				}
			}
		})
	}
}
