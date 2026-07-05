package ratelimit

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Interface
// Limiter is the interface for rate limiters.
// All three implementations (FixedWindowLimiter, SlidingWindowLimiter,
// TokenBucketLimiter) satisfy this interface.
type Limiter interface {
	Allow(key string) Result
}

// KeyFunc extracts a rate limit key from an HTTP request.
// The key is used to identify the client for rate limiting purposes.
// Common implementations include extracting the client IP, user ID, or
// a custom composite key.
type KeyFunc func(r *http.Request) string

// Rate Limit Response Headers
const (
	HeaderLimit      = "X-RateLimit-Limit"
	HeaderRemaining  = "X-RateLimit-Remaining"
	HeaderReset      = "X-RateLimit-Reset"
	HeaderRetryAfter = "Retry-After"
)

// rateLimitResponse is the JSON body sent with a 429 response.
type rateLimitResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int    `json:"retry_after"`
}

// Middleware
// Middleware creates an HTTP middleware that rate limits requests using the
// provided limiter and key function.
//
// The returned middleware has the standard signature func(http.Handler) http.Handler.
//
// On every request:
//  1. The key is extracted using keyFunc (defaults to KeyByIP if nil).
//  2. Empty keys and keys containing CRLF/null bytes are replaced with "anonymous".
//  3. Keys exceeding MaxKeyLength are truncated.
//  4. The limiter checks if the request is allowed.
//  5. Rate limit headers are set on all responses.
//  6. If denied, a 429 Too Many Requests response is returned with a JSON body.
//  7. If allowed, the next handler is called.
//
// The middleware does not include the rate limit key in any response header,
// preventing information leakage.
func Middleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	if keyFunc == nil {
		keyFunc = KeyByIP
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract and sanitize the key
			key := normalizeKey(keyFunc(r))

			// Check the rate limit
			result := limiter.Allow(key)

			// Set rate limit headers on all responses (allowed and denied).
			// These headers contain only numeric values; no key information
			// is leaked.
			w.Header().Set(HeaderLimit, strconv.Itoa(result.Limit))
			w.Header().Set(HeaderRemaining, strconv.Itoa(result.Remaining))
			w.Header().Set(HeaderReset, strconv.FormatInt(result.ResetAt, 10))

			if !result.Allowed {
				w.Header().Set(HeaderRetryAfter, strconv.Itoa(result.RetryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(rateLimitResponse{
					Error:      "Too Many Requests",
					Message:    "Rate limit exceeded. Please try again later.",
					RetryAfter: result.RetryAfter,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Key Functions
// normalizeKey bounds and validates a limiter key before it reaches the store.
func normalizeKey(key string) string {
	if key == "" || strings.ContainsAny(key, "\r\n\x00") {
		return "anonymous"
	}
	if len(key) > MaxKeyLength {
		return key[:MaxKeyLength]
	}
	return key
}

// KeyByIP extracts the client IP address from r.RemoteAddr only.
//
// It deliberately does not trust X-Forwarded-For or X-Real-IP. Those headers
// are client-controlled unless a deployment has a verified trusted-proxy layer,
// and using them here would let attackers spoof rate-limit buckets.
func KeyByIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// KeyByUserID extracts the user ID from the specified request header.
// This is commonly used with an Authorization or X-User-ID header.
// If the header is absent, an empty string is returned (the middleware
// will substitute "anonymous").
func KeyByUserID(headerName string) KeyFunc {
	return func(r *http.Request) string {
		return r.Header.Get(headerName)
	}
}

// KeyByCustom wraps a custom KeyFunc, allowing arbitrary key extraction logic.
// This is useful for composite keys (e.g., IP + route path).
func KeyByCustom(fn KeyFunc) KeyFunc {
	return fn
}

// KeyGlobal returns a constant key, applying a single rate limit to all
// requests regardless of client. This is useful for protecting downstream
// services from overall traffic spikes.
func KeyGlobal(r *http.Request) string {
	return "global"
}
