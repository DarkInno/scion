package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseToken(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"
	issuer := "Scion-auth"

	token, err := GenerateToken(user, secret, time.Hour, issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ParseToken(token, secret, issuer)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", claims.Email, user.Email)
	}
	if claims.Issuer != issuer {
		t.Errorf("Issuer mismatch: got %s, want %s", claims.Issuer, issuer)
	}
	if claims.ID == "" {
		t.Error("expected non-empty JTI (ID)")
	}
	if claims.Subject != "1" {
		t.Errorf("Subject mismatch: got %s, want '1'", claims.Subject)
	}
	matched := false
	for _, aud := range claims.Audience {
		if aud == issuer {
			matched = true
			break
		}
	}
	if !matched {
		t.Error("expected audience to contain issuer")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"
	issuer := "Scion-auth"

	token, err := GenerateToken(user, secret, time.Hour, issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token, "wrong_secret", issuer)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseToken_Expired(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"
	issuer := "Scion-auth"

	token, err := GenerateToken(user, secret, -time.Hour, issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token, secret, issuer)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseToken_InvalidIssuer(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"
	issuer := "Scion-auth"

	token, err := GenerateToken(user, secret, time.Hour, issuer)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ParseToken(token, secret, "other-issuer")
	if err == nil {
		t.Fatal("expected error for invalid issuer")
	}
}

func TestParseToken_InvalidAudience(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"
	issuer := "Scion-auth"

	// Manually create a token with wrong audience
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"other-audience"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("token creation failed: %v", err)
	}

	_, err = ParseToken(token, secret, issuer)
	if err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestParseToken_NonHMAC(t *testing.T) {
	// Create a token with "none" algorithm header
	token := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoxfQ."
	_, err := ParseToken(token, "secret", "issuer")
	if err == nil {
		t.Fatal("expected error for none algorithm")
	}
	if !strings.Contains(err.Error(), "none") && !strings.Contains(err.Error(), "signing method") {
		t.Errorf("expected signing method error, got: %v", err)
	}
}

func TestParseToken_EmptyIssuerAllowed(t *testing.T) {
	user := &User{ID: 1, Email: "test@example.com"}
	secret := "this_is_a_very_long_secret_key_that_is_at_least_32"

	// When issuer is empty, ParseToken should skip issuer/aud checks
	token, err := GenerateToken(user, secret, time.Hour, "")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(token, secret, "")
	if err != nil {
		t.Fatalf("ParseToken with empty issuer failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}
}

func TestGenerateJTI_Unique(t *testing.T) {
	jti1, err := generateJTI()
	if err != nil {
		t.Fatalf("generateJTI failed: %v", err)
	}
	jti2, err := generateJTI()
	if err != nil {
		t.Fatalf("generateJTI failed: %v", err)
	}
	if jti1 == jti2 {
		t.Error("expected two different JTIs")
	}
	if len(jti1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("expected JTI length 32, got %d", len(jti1))
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := ParseToken("not.a.token", "secret", "issuer")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}
