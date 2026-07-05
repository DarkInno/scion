package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const maxCORSMaxAge = 86400 // 24 hours

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// Security defaults:
//   - Empty AllowedOrigins = reject all origins (safe default).
//   - AllowCredentials=true with AllowedOrigins=["*"] panics (security vulnerability).
//   - Vary: Origin is always set on allowed responses (prevents CDN cache poisoning).
func CORS(opts ...CORSOptions) func(http.Handler) http.Handler {
	var opt CORSOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	// Apply defaults.
	if len(opt.AllowedMethods) == 0 {
		opt.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	}
	if len(opt.AllowedHeaders) == 0 {
		opt.AllowedHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}
	if opt.MaxAge <= 0 {
		opt.MaxAge = 86400
	}
	if opt.MaxAge > maxCORSMaxAge {
		opt.MaxAge = maxCORSMaxAge
	}

	// Security constraint: AllowCredentials=true cannot coexist with wildcard origin.
	if opt.AllowCredentials {
		for _, o := range opt.AllowedOrigins {
			if o == "*" {
				panic("cors: AllowCredentials=true cannot be used with AllowedOrigins=[\"*\"]")
			}
		}
	}

	// Pre-process origins into exact match map and wildcard patterns.
	exactOrigins := make(map[string]struct{}, len(opt.AllowedOrigins))
	var wildcardPatterns []string
	for _, o := range opt.AllowedOrigins {
		if strings.Contains(o, "*") {
			wildcardPatterns = append(wildcardPatterns, o)
		} else {
			exactOrigins[strings.ToLower(o)] = struct{}{}
		}
	}

	// Pre-compute comma-joined header strings.
	methodsStr := strings.Join(opt.AllowedMethods, ", ")
	headersStr := strings.Join(opt.AllowedHeaders, ", ")
	exposedStr := strings.Join(opt.ExposedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// No Origin header = not a CORS request, pass through.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Reject origins containing CRLF to prevent HTTP header injection.
			if !isSafeOrigin(origin) {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Check if origin is allowed.
			allowed := isOriginAllowed(origin, exactOrigins, wildcardPatterns)

			if !allowed {
				// For preflight requests, return 204 without CORS headers.
				// The browser will reject the request.
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Set CORS response headers.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if opt.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if exposedStr != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedStr)
			}
			// Vary: Origin prevents CDN cache poisoning.
			w.Header().Add("Vary", "Origin")

			// Handle preflight (OPTIONS) request.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methodsStr)
				w.Header().Set("Access-Control-Allow-Headers", headersStr)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(opt.MaxAge))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the given origin matches any allowed pattern.
func isOriginAllowed(origin string, exact map[string]struct{}, wildcards []string) bool {
	// Empty list = reject all (safe default).
	if len(exact) == 0 && len(wildcards) == 0 {
		return false
	}

	// Exact match (case-insensitive).
	if _, ok := exact[strings.ToLower(origin)]; ok {
		return true
	}

	// Wildcard match (prefix only, e.g., https://*.example.com).
	for _, pattern := range wildcards {
		if matchWildcard(origin, pattern) {
			return true
		}
	}

	return false
}

// matchWildcard supports ONLY subdomain wildcard patterns like "https://*.example.com".
// Security constraints:
//   - The "*" MUST be preceded by a non-empty prefix (e.g., "https://") AND
//     followed by a non-empty suffix containing at least one "." (e.g., ".example.com").
//   - This prevents patterns like "*example.com" (matches "attackerexample.com")
//     or "https://*" (matches everything) or "*.com" (matches "attacker.com").
func matchWildcard(origin, pattern string) bool {
	parts := strings.SplitN(pattern, "*", 2)
	if len(parts) != 2 {
		return false
	}
	prefix := parts[0]
	suffix := parts[1]

	// Both prefix and suffix must be non-empty.
	if prefix == "" || suffix == "" {
		return false
	}

	// Suffix must contain at least one dot to prevent overly broad patterns.
	// e.g., "*.example.com" is valid, but "*ample.com" is rejected.
	if !strings.Contains(suffix, ".") {
		return false
	}

	return strings.HasPrefix(strings.ToLower(origin), strings.ToLower(prefix)) &&
		strings.HasSuffix(strings.ToLower(origin), strings.ToLower(suffix))
}

// isSafeOrigin validates that an origin string does not contain CRLF characters
// that could lead to HTTP header injection.
func isSafeOrigin(origin string) bool {
	return !strings.ContainsAny(origin, "\r\n")
}

// ValidateCORSConfig validates CORS options and returns an error if invalid.
// Useful for catching configuration errors at startup rather than runtime.
func ValidateCORSConfig(opt CORSOptions) error {
	if opt.AllowCredentials {
		for _, o := range opt.AllowedOrigins {
			if o == "*" {
				return fmt.Errorf("cors: AllowCredentials=true cannot be used with AllowedOrigins=[\"*\"]")
			}
		}
	}
	return nil
}
