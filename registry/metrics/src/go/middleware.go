package metrics

import (
	"net/http"
	"time"
)

// Middleware records request count, latency, in-flight count, method, route,
// and status. Pass a route template such as "/users/{id}", not r.URL.Path.
func (m *Metrics) Middleware(route string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.Instrument(route, next)
	}
}

// Instrument wraps a handler and records Prometheus metrics for it.
func (m *Metrics) Instrument(route string, next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		lw := &responseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				if lw.status == 0 {
					lw.status = http.StatusInternalServerError
				}
				m.observe(r.Method, route, lw.status, time.Since(start).Seconds())
				panic(recovered)
			}
			status := lw.status
			if status == 0 {
				status = http.StatusOK
			}
			m.observe(r.Method, route, status, time.Since(start).Seconds())
		}()
		next.ServeHTTP(lw, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
