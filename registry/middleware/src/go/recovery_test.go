package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRecoveryNormal(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	recovered := Recovery()
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", body)
	}
}

func TestRecoveryPanic(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	recovered := Recovery()
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}
	if body["error"] != "internal server error" {
		t.Fatalf("unexpected error message: %v", body["error"])
	}
}

func TestRecoveryAfterWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	recovered := Recovery()
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial"))
		panic("after write")
	}))
	// Should NOT produce superfluous WriteHeader warnings.
	h.ServeHTTP(rec, req)

	// Status should remain 200 because headers were already written.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (headers already written), got %d", rec.Code)
	}
	// Body should contain the partial write, not the JSON error.
	if body := rec.Body.String(); !strings.Contains(body, "partial") {
		t.Fatalf("expected body to contain 'partial', got %q", body)
	}
}

func TestRecoveryCustomLogFunc(t *testing.T) {
	var logCalled bool
	var gotErr interface{}
	var gotStack []byte

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)

	recovered := Recovery(RecoveryOptions{
		LogFunc: func(err interface{}, stack []byte) {
			logCalled = true
			gotErr = err
			gotStack = stack
		},
	})
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("custom panic")
	}))
	h.ServeHTTP(rec, req)

	if !logCalled {
		t.Fatal("custom LogFunc was not called")
	}
	if gotErr != "custom panic" {
		t.Fatalf("LogFunc received error %v, want %v", gotErr, "custom panic")
	}
	if len(gotStack) == 0 {
		t.Fatal("LogFunc received empty stack")
	}
}

func TestRecoveryCustomResponseFunc(t *testing.T) {
	var responseCalled bool

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	recovered := Recovery(RecoveryOptions{
		ResponseFunc: func(w http.ResponseWriter) {
			responseCalled = true
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("custom 503"))
		},
	})
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	h.ServeHTTP(rec, req)

	if !responseCalled {
		t.Fatal("custom ResponseFunc was not called")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "custom 503" {
		t.Fatalf("expected body %q, got %q", "custom 503", body)
	}
}

func TestRecoveryStackSizeLimit(t *testing.T) {
	stackSize := 2 // 2 KB
	var gotStack []byte

	recovered := Recovery(RecoveryOptions{
		StackSize: stackSize,
		LogFunc: func(err interface{}, stack []byte) {
			gotStack = stack
		},
	})
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Trigger a deep call stack to generate a large stack trace.
		deepPanic(50)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	limit := stackSize * 1024
	if len(gotStack) > limit {
		t.Fatalf("stack size %d exceeds limit %d", len(gotStack), limit)
	}
	// Stack should be non-empty (at least contains some trace).
	if len(gotStack) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
}

func TestRecoveryMultiplePanics(t *testing.T) {
	recovered := Recovery()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("sequential panic")
	})
	h := recovered(panicHandler)

	// Serve multiple requests that each panic.
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("request %d: expected 500, got %d", i, rec.Code)
		}
	}
	// If we reach here, all panics were recovered.
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkRecoveryNoPanic(b *testing.B) {
	recovered := Recovery()
	h := recovered(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// deepPanic recursively calls itself to generate a deep stack trace, then panics.
func deepPanic(depth int) {
	if depth == 0 {
		panic("deep panic")
	}
	deepPanic(depth - 1)
}
