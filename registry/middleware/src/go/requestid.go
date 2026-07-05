package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxRequestIDLen = 128 // Maximum length for client-supplied request IDs

// RequestID returns a middleware that generates or propagates a unique request ID.
// If the request already has the ID header, it is reused (for distributed tracing).
// Otherwise, a new UUIDv7 is generated and set in both the response header and context.
//
// SECURITY: Client-supplied request IDs are validated:
//   - Must not contain CRLF characters (prevents header injection / DoS).
//   - Must not exceed maxRequestIDLen characters (prevents memory exhaustion).
//   - If validation fails, a new UUIDv7 is generated instead.
func RequestID(opts ...RequestIDOptions) func(http.Handler) http.Handler {
	var opt RequestIDOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.HeaderName == "" {
		opt.HeaderName = "X-Request-ID"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(opt.HeaderName)
			// Validate client-supplied ID: reject CRLF, null bytes, and excessive length.
			if id != "" && (len(id) > maxRequestIDLen || strings.ContainsAny(id, "\r\n\x00")) {
				id = ""
			}
			if id == "" {
				if opt.Generator != nil {
					id = opt.Generator()
				}
				if id == "" {
					id = generateUUIDv7()
				}
			}

			// Inject into response header.
			w.Header().Set(opt.HeaderName, id)

			// Store in context.
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID extracts the Request ID from context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// generateUUIDv7 creates a UUIDv7 (RFC 9562) without external dependencies.
// Format: xxxxxxxx-xxxx-7xxx-8xxx-xxxxxxxxxxxx (8-4-4-4-12)
//
// Layout:
//
//	bytes 0-3:  32-bit timestamp (high)
//	bytes 4-5:  16-bit timestamp (low)
//	byte  6:    4-bit version (0111) | 12-bit rand
//	byte  7:    12-bit rand (cont.)
//	byte  8:    2-bit variant (10) | 14-bit rand
//	byte  9:    14-bit rand (cont.)
//	bytes 10-15: 48-bit rand
func generateUUIDv7() string {
	var b [16]byte

	// 48-bit Unix millisecond timestamp, big-endian.
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	// Fill remaining 10 bytes (b[6]..b[15]) with random data.
	if _, err := io.ReadFull(rand.Reader, b[6:]); err != nil {
		// crypto/rand failure is extremely rare (e.g., entropy source exhausted).
		// Fall back to time-based pseudo-random to avoid crashing.
		return fallbackID()
	}

	// Set version to 7 (upper 4 bits of byte 6).
	b[6] = (b[6] & 0x0f) | 0x70

	// Set variant to 10 (upper 2 bits of byte 8).
	b[8] = (b[8] & 0x3f) | 0x80

	// Format as standard UUID string: 8-4-4-4-12 hex characters.
	// Use encoding/hex for correctness (Sprintf %x on []byte is unreliable for width control).
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}

// fallbackID generates a time-based ID when crypto/rand fails.
func fallbackID() string {
	return fmt.Sprintf("%d-%04x", time.Now().UnixNano(), time.Now().Nanosecond()%0xFFFF)
}
