package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateRoutePrefix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"already valid", "/api/v1/auth", "/api/v1/auth", false},
		{"trailing slash", "/api/v1/auth/", "/api/v1/auth", false},
		{"no leading slash", "api/v1/auth", "/api/v1/auth", false},
		{"empty string", "", "", true},
		{"with spaces", "  /api/v1/auth  ", "/api/v1/auth", false},
		{"root", "/", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateRoutePrefix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoutePrefix(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateRoutePrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: 3600000000000, // 1h in nanoseconds for duration
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Test register route exists
	body := []byte(`{"email":"route@test.com","password":"securepassword123","name":"Route"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("register route failed: expected %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	// Test login route exists
	body = []byte(`{"email":"route@test.com","password":"securepassword123"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("login route failed: expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

// Need bytes import for routes_test
type _ struct{}
