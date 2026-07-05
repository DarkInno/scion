package middleware

import (
	"net/http"
)

// wrappedWriter wraps http.ResponseWriter to track whether WriteHeader
// has been called and what status code was set.
// Used by recovery.go, logging.go, and timeout.go.
//
// Optional interfaces (Flusher, Hijacker, Pusher) are automatically
// discovered via Unwrap() by http.ResponseController (Go 1.20+).
type wrappedWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *wrappedWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap exposes the underlying ResponseWriter for http.ResponseController (Go 1.20+).
func (w *wrappedWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
