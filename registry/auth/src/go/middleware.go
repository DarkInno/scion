package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsKey contextKey = "authClaims"

// Middleware validates the JWT from the Authorization header.
// Attach this to routes that require authentication.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		// Trim whitespace around the token — clients may send extra spaces.
		token := strings.TrimSpace(parts[1])
		if token == "" {
			respondError(w, http.StatusUnauthorized, "empty token")
			return
		}

		claims, err := ParseToken(token, h.config.JWTSecret, h.config.Issuer)
		if err != nil {
			h.logger.Debug("token validation failed", "error", err)
			respondError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext extracts auth claims from the request context.
// Returns nil if the request is not authenticated.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}
