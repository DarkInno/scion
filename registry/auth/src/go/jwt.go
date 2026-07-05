package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends jwt.RegisteredClaims with custom user fields.
//
// The JWT ID (jti) claim is stored in RegisteredClaims.ID.
// Use it for token revocation/blocklist implementations.
//
// IMPORTANT: Do NOT use the raw JWT string or its SHA-256 hash as a
// blocklist key — ECDSA signatures are malleable, and different base64url
// encodings can produce different byte sequences for the same payload.
// Always use the RegisteredClaims.ID (jti) claim as the blocklist key.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// generateJTI creates a cryptographically secure random token ID.
func generateJTI() (string, error) {
	b := make([]byte, 16) // 128 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateToken creates a new JWT for the given user.
func GenerateToken(user *User, secret string, expiry time.Duration, issuer string) (string, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", fmt.Errorf("failed to generate JTI: %w", err)
	}

	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			Audience:  jwt.ClaimStrings{issuer}, // token is only valid for this service
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti, // standard jti claim
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates a JWT string and returns its claims.
func ParseToken(tokenString string, secret string, issuer string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Reject non-HMAC signing methods (prevents "none" algorithm attack)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Verify issuer to prevent token reuse across services
	if issuer != "" && claims.Issuer != issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	// Verify audience
	if issuer != "" {
		matched := false
		for _, aud := range claims.Audience {
			if aud == issuer {
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("invalid token audience")
		}
	}

	return claims, nil
}
