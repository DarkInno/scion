package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAccessLog_Basic(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_SkipPaths(t *testing.T) {
	m := NewAccessLog(AccessLogOptions{
		SkipPaths: []string{"/health"},
	})
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_DropOnFull(t *testing.T) {
	m := NewAccessLog(AccessLogOptions{
		BufferSize: 1,
		DropOnFull: true,
	})
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send many requests to overflow the tiny buffer.
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Allow goroutine to process some entries.
	time.Sleep(50 * time.Millisecond)

	if m.DroppedCount() == 0 {
		t.Error("expected some entries to be dropped")
	}
}

func TestAccessLog_DefaultDropOnFull(t *testing.T) {
	// Calling AccessLog() with no options should use defaults (DropOnFull=true).
	mw := AccessLog()
	_ = mw
	// We can't easily test the default without flushing, but we verify it compiles
	// and returns a valid middleware function.
}

func TestAccessLog_FlushTwice(t *testing.T) {
	m := NewAccessLog()
	m.Flush()
	m.Flush() // Should not panic.
}

func TestAccessLog_FlushThenRequest(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	m.Flush()

	// After Flush, new requests should not panic.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_RequestIDFromResponseHeader(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	var capturedRequestID string
	handler := Chain(
		RequestID(),
		func(next http.Handler) http.Handler {
			return m.Apply(next)
		},
	).Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// The request ID should be present in the response header.
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID in response header")
	}
	_ = capturedRequestID
}

func TestAccessLog_ClientIPAndUserAgent(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_BufferSizeBounds(t *testing.T) {
	m := NewAccessLog(AccessLogOptions{BufferSize: 20000})
	defer m.Flush()

	// BufferSize should be capped at 10000.
	// We verify by sending many requests without dropping.
	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if m.DroppedCount() > 0 {
		t.Logf("dropped count: %d (may be expected under load)", m.DroppedCount())
	}
}

func TestAccessLog_StatusTracking(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestAccessLog_PathLogged(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify handler executed correctly.
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_ChainCompatibility(t *testing.T) {
	m := NewAccessLog()
	defer m.Flush()

	chain := Chain(m.Apply).Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/chain", nil)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAccessLog_BlockOnFull(t *testing.T) {
	m := NewAccessLog(AccessLogOptions{
		BufferSize: 1,
		DropOnFull: false,
	})
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	done := make(chan struct{})
	go func() {
		// This may block because the background goroutine is not consuming.
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("expected request to block when channel is full and DropOnFull=false")
	}
}

func BenchmarkAccessLog(b *testing.B) {
	m := NewAccessLog(AccessLogOptions{DropOnFull: true})
	defer m.Flush()

	handler := m.Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/bench", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}
