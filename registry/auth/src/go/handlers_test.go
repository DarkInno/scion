package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockUserStore is a test double for UserStore.
type mockUserStore struct {
	users      map[string]*User
	createErr  error
	getByIDErr error
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*User)}
}

func (m *mockUserStore) CreateUser(email, passwordHash, name string) (*User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if _, exists := m.users[email]; exists {
		return nil, errors.New("duplicate email")
	}
	u := &User{
		ID:        uint(len(m.users) + 1),
		Email:     email,
		Password:  passwordHash,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[email] = u
	return u, nil
}

func (m *mockUserStore) GetUserByEmail(email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserStore) GetUserByID(id uint) (*User, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func TestHandler_Register(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	body, _ := json.Marshal(RegisterRequest{
		Email:    "new@example.com",
		Password: "securepassword123",
		Name:     "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp AuthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token in response")
	}
	if resp.User.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", resp.User.Email)
	}
}

func TestHandler_Register_InvalidInput(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{JWTSecret: "this_is_a_very_long_secret_key_that_is_at_least_32", BcryptCost: DefaultBCryptCost}
	h := NewHandler(store, cfg)

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "invalid email",
			body:       RegisterRequest{Email: "bad", Password: "securepassword123", Name: "User"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "short password",
			body:       RegisterRequest{Email: "user@example.com", Password: "short", Name: "User"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "malformed JSON",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			rr := httptest.NewRecorder()
			h.Register(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	// First registration succeeds
	body, _ := json.Marshal(RegisterRequest{Email: "dup@example.com", Password: "securepassword123", Name: "User"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("first register failed: %d %s", rr.Code, rr.Body.String())
	}

	// Second registration with same email fails with generic message
	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	rr2 := httptest.NewRecorder()
	h.Register(rr2, req2)
	if rr2.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rr2.Code)
	}
	var errResp map[string]string
	json.Unmarshal(rr2.Body.Bytes(), &errResp)
	if !strings.Contains(errResp["error"], "unable to create account") {
		t.Errorf("expected generic error message, got %q", errResp["error"])
	}
}

func TestHandler_Login(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	// Register a user first
	pw := "securepassword123"
	hash, _ := HashPassword(pw, cfg.BcryptCost)
	store.CreateUser("login@example.com", hash, "Login User")

	body, _ := json.Marshal(LoginRequest{Email: "login@example.com", Password: pw})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp AuthResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected token in response")
	}
}

func TestHandler_Login_InvalidPassword(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	hash, _ := HashPassword("correctpassword", cfg.BcryptCost)
	store.CreateUser("user@example.com", hash, "User")

	body, _ := json.Marshal(LoginRequest{Email: "user@example.com", Password: "wrongpassword"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	var errResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errResp)
	if !strings.Contains(errResp["error"], "invalid email or password") {
		t.Errorf("expected generic error, got %q", errResp["error"])
	}
}

func TestHandler_Login_UserNotFound(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:  "this_is_a_very_long_secret_key_that_is_at_least_32",
		BcryptCost: DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	body, _ := json.Marshal(LoginRequest{Email: "nobody@example.com", Password: "anypassword"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	var errResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errResp)
	if !strings.Contains(errResp["error"], "invalid email or password") {
		t.Errorf("expected generic error, got %q", errResp["error"])
	}
}

func TestHandler_Login_RateLimit(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:  "this_is_a_very_long_secret_key_that_is_at_least_32",
		BcryptCost: DefaultBCryptCost,
	}
	h := NewHandler(store, cfg).WithRateLimiter(NewMemoryRateLimiter(2, time.Minute))

	body, _ := json.Marshal(LoginRequest{Email: "user@example.com", Password: "wrong"})
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		h.Login(rr, req)
		if i < 2 {
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 on attempt %d, got %d", i+1, rr.Code)
			}
		} else {
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("expected 429 on attempt %d, got %d", i+1, rr.Code)
			}
		}
	}
}

func TestHandler_Me(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{
		JWTSecret:   "this_is_a_very_long_secret_key_that_is_at_least_32",
		TokenExpiry: time.Hour,
		Issuer:      "Scion-auth",
		BcryptCost:  DefaultBCryptCost,
	}
	h := NewHandler(store, cfg)

	user, _ := store.CreateUser("me@example.com", "hash", "Me")
	claims := &Claims{UserID: user.ID, Email: user.Email}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	req := httptest.NewRequest(http.MethodGet, "/me", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp UserPublic
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Email != "me@example.com" {
		t.Errorf("expected email me@example.com, got %s", resp.Email)
	}
}

func TestHandler_Me_Unauthorized(t *testing.T) {
	store := newMockUserStore()
	cfg := &Config{BcryptCost: DefaultBCryptCost}
	h := NewHandler(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rr := httptest.NewRecorder()

	h.Me(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestDecodeBody_MaxSize(t *testing.T) {
	// 1MB + 1 should be truncated by LimitReader
	large := make([]byte, maxRequestBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(large))

	var dst map[string]interface{}
	err := decodeBody(req, &dst)
	// Should fail because truncated JSON is invalid
	if err == nil {
		t.Error("expected error for oversized body")
	}
}

func TestRespondJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	respondJSON(rr, http.StatusOK, map[string]string{"key": "value"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
	if !strings.Contains(rr.Body.String(), "key") {
		t.Error("expected body to contain key")
	}
}

func TestRespondError(t *testing.T) {
	rr := httptest.NewRecorder()
	respondError(rr, http.StatusBadRequest, "bad request")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "bad request" {
		t.Errorf("expected error 'bad request', got %q", resp["error"])
	}
}
