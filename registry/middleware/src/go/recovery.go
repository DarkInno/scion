package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns a middleware that recovers from panics in the handler chain.
// It logs the stack trace and returns a 500 response.
// Recovery should be the outermost middleware in the chain.
func Recovery(opts ...RecoveryOptions) func(http.Handler) http.Handler {
	var opt RecoveryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.StackSize <= 0 {
		opt.StackSize = 32
	}
	if opt.StackSize > 128 {
		opt.StackSize = 128
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap ResponseWriter to track whether headers were already written.
			lw := &wrappedWriter{ResponseWriter: w}

			defer func() {
				if err := recover(); err != nil {
					// Capture stack trace.
					stack := debug.Stack()
					if len(stack) > opt.StackSize*1024 {
						stack = stack[:opt.StackSize*1024]
					}

					// Log the panic.
					if opt.LogFunc != nil {
						opt.LogFunc(err, stack)
					} else {
						slog.Error("panic recovered",
							"error", err,
							"method", r.Method,
							"path", r.URL.Path,
							"request_id", GetRequestID(r.Context()),
							"stack", string(stack),
						)
					}

					// Send response only if headers have not been written yet.
					if !lw.wrote {
						if opt.ResponseFunc != nil {
							opt.ResponseFunc(w)
						} else {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusInternalServerError)
							_ = json.NewEncoder(w).Encode(map[string]string{
								"error": "internal server error",
							})
						}
					}
				}
			}()

			next.ServeHTTP(lw, r)
		})
	}
}
