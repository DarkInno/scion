package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrappedWriterTracksStatusOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &wrappedWriter{ResponseWriter: rec}
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusTeapot)
	if w.status != http.StatusCreated || !w.wrote {
		t.Fatalf("status tracking failed: %+v", w)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("recorder code = %d", rec.Code)
	}
	if w.Unwrap() != rec {
		t.Fatal("Unwrap should expose underlying writer")
	}
}

func TestWrappedWriterWriteImplicitOK(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &wrappedWriter{ResponseWriter: rec}
	if _, err := w.Write([]byte("ok")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if w.status != http.StatusOK || rec.Code != http.StatusOK {
		t.Fatalf("implicit status failed: wrapper=%d recorder=%d", w.status, rec.Code)
	}
}
