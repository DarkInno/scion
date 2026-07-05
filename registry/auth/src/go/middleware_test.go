package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_ValidToken(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
	}
	h := NewHandler(store, cfg)

	user := &User{ID: 1, Email: "auth@example.com"}
	token, err := GenerateToken(user, cfg.JWTSecret, cfg.TokenExpiry, cfg.Issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("expected claims in context")
			return
		}
		if claims.UserID != user.ID {
			t.Errorf("expected UserID %d, got %d", user.ID, claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestMiddleware_MissingHeader(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{JWTSecret: "this_is_a_very_long_secret_key_that_is_at_least_32"}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rr := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth header")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestMiddleware_InvalidFormat(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{JWTSecret: "this_is_a_very_long_secret_key_that_is_at_least_32"}
	h := NewHandler(store, cfg)

	tests := []struct {
		name  string
		value string
	}{
		{"no bearer prefix", "token123"},
		{"wrong prefix", "Basic token123"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			req.Header.Set("Authorization", tt.value)
			rr := httptest.NewRecorder()

			h.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("handler should not be called with invalid format")
			})).ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
			}
		})
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret: "this_is_a_very_long_secret_key_that_is_at_least_32",
		Issuer:    "Scion-auth",
	}
	h := NewHandler(store, cfg)

	user := &User{ID: 1, Email: "expired@example.com"}
	token, err := GenerateToken(user, cfg.JWTSecret, -time.Hour, cfg.Issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with expired token")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestMiddleware_WrongSecret(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret: "this_is_a_very_long_secret_key_that_is_at_least_32",
		Issuer:    "Scion-auth",
	}
	h := NewHandler(store, cfg)

	user := &User{ID: 1, Email: "wrong@example.com"}
	token, err := GenerateToken(user, "different_secret_key_that_is_also_long_", time.Hour, cfg.Issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with wrong secret")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestClaimsFromContext(t *testing.T) {
	claims := &Claims{UserID: 42, Email: "ctx@example.com"}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	got := ClaimsFromContext(ctx)
	if got == nil {
		t.Fatal("expected claims from context")
	}
	if got.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", got.UserID)
	}
}

func TestClaimsFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	got := ClaimsFromContext(ctx)
	if got != nil {
		t.Error("expected nil claims from empty context")
	}
}
