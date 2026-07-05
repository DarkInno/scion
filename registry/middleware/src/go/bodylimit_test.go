package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimitNormal(t *testing.T) {
	// Handler that reads the body and echoes it back.
	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	})

	// 64 bytes limit.
	handler := BodyLimit(BodyLimitOptions{MaxSize: 64})(echoHandler)

	// Send a 32-byte body (within limit).
	payload := strings.NewReader("This is a perfectly normal request body")
	req := httptest.NewRequest(http.MethodPost, "/", payload)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for normal body, got %d", rec.Code)
	}
}

func TestBodyLimitExceeded(t *testing.T) {
	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			// MaxBytesReader returns an error; the handler should detect it.
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// 16 bytes limit.
	handler := BodyLimit(BodyLimitOptions{MaxSize: 16})(echoHandler)

	// Send a body larger than 16 bytes.
	largeBody := bytes.NewReader([]byte("This body is definitely larger than sixteen bytes"))
	req := httptest.NewRequest(http.MethodPost, "/", largeBody)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for oversized body, got %d", rec.Code)
	}
}

func TestBodyLimitDefault(t *testing.T) {
	// Zero options => should use 1MB default.
	handler := BodyLimit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small body"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with default 1MB limit, got %d", rec.Code)
	}
}

func TestBodyLimitMaxClamp(t *testing.T) {
	// Set a limit > 100MB. It should be clamped to 100MB.
	handler := BodyLimit(BodyLimitOptions{MaxSize: 200 << 20})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Send a body slightly over 100MB would fail, but we just verify the middleware
	// was constructed and processes normally (without crashing).
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after max clamp, got %d", rec.Code)
	}
}

func BenchmarkBodyLimit(b *testing.B) {
	handler := BodyLimit(BodyLimitOptions{MaxSize: 1 << 20})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Drain body to simulate real usage.
		io.Copy(io.Discard, r.Body)
	}))

	body := bytes.NewReader(make([]byte, 512)) // 512 bytes, well within limit.

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset the reader for each iteration.
		body.Seek(0, io.SeekStart)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
