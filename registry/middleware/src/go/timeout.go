package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const maxTimeout = 5 * time.Minute

// Timeout returns a middleware that enforces a request deadline via context cancellation.
// The handler should check ctx.Done() to cooperate with the timeout.
// If the handler does not check ctx.Done(), it will continue running after the timeout
// but its response will be discarded.
//
// Note: This is cooperative timeout. The handler goroutine is not forcefully killed.
// For protection against goroutines that ignore ctx.Done(), ensure your handlers
// check the context at appropriate points (e.g., before slow DB queries).
func Timeout(opts ...TimeoutOptions) func(http.Handler) http.Handler {
	var opt TimeoutOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 30 * time.Second
	}
	if opt.Timeout > maxTimeout {
		opt.Timeout = maxTimeout
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), opt.Timeout)
			defer cancel()

			tw := &timeoutWriter{ResponseWriter: w}

			done := make(chan struct{})
			go func() {
				defer close(done)
				// Recover panics in the handler goroutine.
				// Recovery middleware (if present) runs in the parent goroutine
				// and cannot catch panics from this goroutine.
				defer func() {
					if p := recover(); p != nil {
						slog.Error("panic in timeout goroutine",
							"error", p,
							"method", r.Method,
							"path", r.URL.Path,
							"request_id", GetRequestID(r.Context()),
						)
						tw.mu.Lock()
						if !tw.wrote {
							tw.timedOut = true
							tw.wrote = true
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusInternalServerError)
							_ = json.NewEncoder(w).Encode(map[string]string{
								"error": "internal server error",
							})
						}
						tw.mu.Unlock()
					}
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			select {
			case <-done:
				// Request completed within timeout.
			case <-ctx.Done():
				// Timeout exceeded.
				tw.mu.Lock()
				if !tw.wrote {
					tw.timedOut = true
					tw.wrote = true
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGatewayTimeout)
					if opt.Message != "" {
						w.Write([]byte(opt.Message))
					} else {
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error": "request timeout",
						})
					}
				}
				tw.mu.Unlock()
			}
		})
	}
}

// timeoutWriter is a thread-safe ResponseWriter wrapper for the Timeout middleware.
// It protects against concurrent writes from both the handler goroutine and the timeout goroutine.
// After a timeout, all subsequent writes are silently discarded.
type timeoutWriter struct {
	http.ResponseWriter
	mu       sync.Mutex
	wrote    bool
	timedOut bool
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wrote {
		return
	}
	tw.wrote = true
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, nil
	}
	if !tw.wrote {
		tw.wrote = true
		tw.ResponseWriter.WriteHeader(http.StatusOK)
	}
	return tw.ResponseWriter.Write(b)
}

// Unwrap exposes the underlying ResponseWriter for http.ResponseController.
func (tw *timeoutWriter) Unwrap() http.ResponseWriter {
	return tw.ResponseWriter
}
