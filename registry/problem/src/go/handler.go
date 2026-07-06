package problem

import (
	"encoding/json"
	"errors"
	"net/http"
)

// HTTPError wraps a Problem as an error value for Handler.
type HTTPError struct {
	Problem Problem
	Err     error
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Problem.Title != "" {
		return e.Problem.Title
	}
	return http.StatusText(e.Problem.Status)
}

func (e *HTTPError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Error creates an error value that Handler renders as a problem response.
func Error(status int, title, detail string) error {
	return &HTTPError{Problem: New(status, title, detail)}
}

// Write writes a sanitized problem response.
func Write(w http.ResponseWriter, r *http.Request, p Problem, opts ...Options) {
	opt := Defaults()
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt = opt.normalize()
	if opt.IncludeRequestID && r != nil {
		p.RequestID = r.Header.Get(opt.RequestIDHeader)
	}
	p = sanitizeProblem(p, opt)
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// Handler converts returned errors into problem responses and recovers panics.
func Handler(fn func(http.ResponseWriter, *http.Request) error, opts ...Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &responseWriter{ResponseWriter: w}
		defer func() {
			if recover() != nil {
				if !lw.wrote {
					Write(lw, r, Internal(), opts...)
				}
			}
		}()
		if fn == nil {
			Write(lw, r, Internal(), opts...)
			return
		}
		if err := fn(lw, r); err != nil {
			if lw.wrote {
				return
			}
			var httpErr *HTTPError
			if errors.As(err, &httpErr) {
				Write(lw, r, httpErr.Problem, opts...)
				return
			}
			Write(lw, r, Internal(), opts...)
		}
	})
}

// Recoverer returns middleware that converts panics into generic 500 problem
// responses.
func Recoverer(opts ...Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &responseWriter{ResponseWriter: w}
			defer func() {
				if recover() != nil {
					if !lw.wrote {
						Write(lw, r, Internal(), opts...)
					}
				}
			}()
			next.ServeHTTP(lw, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (w *responseWriter) WriteHeader(status int) {
	if !w.wrote {
		w.wrote = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
