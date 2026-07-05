package middleware

import "net/http"

const maxBodyLimit = 100 << 20 // 100MB

// BodyLimit returns a middleware that limits the request body size.
// It wraps r.Body with http.MaxBytesReader which returns 413 when exceeded.
// This protects against memory exhaustion from oversized requests.
//
// Note: The handler should check for errors when reading the body.
// MaxBytesReader sets r.Body to a limited reader that returns error on overflow.
func BodyLimit(opts ...BodyLimitOptions) func(http.Handler) http.Handler {
	var opt BodyLimitOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.MaxSize <= 0 {
		opt.MaxSize = 1 << 20 // 1MB default
	}
	if opt.MaxSize > maxBodyLimit {
		opt.MaxSize = maxBodyLimit
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// http.MaxBytesReader returns 413 Request Entity Too Large
			// when the body exceeds the limit.
			r.Body = http.MaxBytesReader(w, r.Body, opt.MaxSize)
			next.ServeHTTP(w, r)
		})
	}
}
