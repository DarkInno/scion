package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// logEntry holds the data for a single access log entry.
type logEntry struct {
	method    string
	path      string
	status    int
	latency   time.Duration
	requestID string
	clientIP  string
	userAgent string
}

// accessLogState holds the shared state for an AccessLog instance.
type accessLogState struct {
	ch        chan logEntry
	dropCount atomic.Int64
	wg        sync.WaitGroup
	once      sync.Once
	closed    atomic.Bool
}

// Flush closes the log channel and waits for all queued entries to be written.
// Safe to call multiple times — only the first call has effect.
// After Flush, new log entries are silently dropped.
func (s *accessLogState) Flush() {
	s.once.Do(func() {
		s.closed.Store(true)
		close(s.ch)
	})
	s.wg.Wait()
}

// DroppedCount returns the total number of dropped log entries for this instance.
func (s *accessLogState) DroppedCount() int64 {
	return s.dropCount.Load()
}

// FlushableAccessLog wraps the access log middleware with lifecycle management.
// Use AccessLog() for a simple func(http.Handler) http.Handler compatible with Chain().
// Use NewAccessLog() if you need Flush() for graceful shutdown.
type FlushableAccessLog struct {
	applyFunc func(http.Handler) http.Handler
	state     *accessLogState
}

// Apply returns the middleware function for use with Chain() and other composition patterns.
func (m *FlushableAccessLog) Apply(next http.Handler) http.Handler {
	return m.applyFunc(next)
}

// Flush waits for all pending log entries to be written and stops the background goroutine.
// Safe to call multiple times.
func (m *FlushableAccessLog) Flush() {
	m.state.Flush()
}

// DroppedCount returns the number of log entries dropped due to a full channel buffer.
func (m *FlushableAccessLog) DroppedCount() int64 {
	return m.state.DroppedCount()
}

// AccessLog returns a func(http.Handler) http.Handler compatible with Chain().
// For graceful shutdown support, use NewAccessLog() and call Flush() on the returned value.
//
// Logs are written asynchronously via a buffered channel to minimize request latency.
// Only method, path, status, latency, request_id, client_ip, user_agent are logged.
// No headers (including Authorization), no query strings, no request/response bodies.
func AccessLog(opts ...AccessLogOptions) func(http.Handler) http.Handler {
	return NewAccessLog(opts...).applyFunc
}

// NewAccessLog returns a FlushableAccessLog with structured async access logging
// and Flush() support for graceful shutdown.
//
// Usage:
//
//	m := middleware.NewAccessLog()
//	defer m.Flush()
//	handler := middleware.Chain(m.Apply, ...).Then(h)
func NewAccessLog(opts ...AccessLogOptions) *FlushableAccessLog {
	var opt AccessLogOptions
	if len(opts) > 0 {
		opt = opts[0]
	} else {
		opt = AccessLogDefaults()
	}
	if opt.BufferSize <= 0 {
		opt.BufferSize = AccessLogDefaults().BufferSize
	}
	if opt.BufferSize > 10000 {
		opt.BufferSize = 10000
	}

	logger := slog.Default()

	skipMap := make(map[string]struct{}, len(opt.SkipPaths))
	for _, p := range opt.SkipPaths {
		skipMap[p] = struct{}{}
	}

	state := &accessLogState{
		ch: make(chan logEntry, opt.BufferSize),
	}

	state.wg.Add(1)
	go func() {
		defer state.wg.Done()
		for entry := range state.ch {
			logger.Info("access",
				"method", entry.method,
				"path", entry.path,
				"status", entry.status,
				"latency_ms", entry.latency.Milliseconds(),
				"request_id", entry.requestID,
				"client_ip", entry.clientIP,
				"user_agent", entry.userAgent,
			)
		}
	}()

	applyFunc := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, skip := skipMap[r.URL.Path]; skip {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			lw := &wrappedWriter{ResponseWriter: w}

			next.ServeHTTP(lw, r)

			// Go's http.Server sends 200 OK if the handler returns without
			// calling WriteHeader. Reflect that in the log entry.
			status := lw.status
			if status == 0 {
				status = http.StatusOK
			}

			// Read request ID from response header (set by RequestID middleware
			// which executes before AccessLog in the inner layers).
			// This ensures the generated UUID is logged even when the client
			// did not send a request ID header.
			requestID := w.Header().Get("X-Request-ID")
			if requestID == "" {
				requestID = r.Header.Get("X-Request-ID")
			}

			entry := logEntry{
				method:    r.Method,
				path:      r.URL.Path,
				status:    status,
				latency:   time.Since(start),
				requestID: requestID,
				clientIP:  ClientIP(r),
				userAgent: r.UserAgent(),
			}

			if state.closed.Load() {
				return
			}
			select {
			case state.ch <- entry:
			default:
				if opt.DropOnFull {
					state.dropCount.Add(1)
				} else {
					state.ch <- entry
				}
			}
		})
	}

	return &FlushableAccessLog{
		applyFunc: applyFunc,
		state:     state,
	}
}
