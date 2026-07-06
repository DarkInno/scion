package problem

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteProblemResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("X-Request-ID", "req-123")
	rec := httptest.NewRecorder()
	Write(rec, req, New(http.StatusNotFound, "Not found", "missing"), Options{IncludeRequestID: true})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != mediaType {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"request_id":"req-123"`) || !strings.Contains(body, `"detail":"missing"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestHandlerMapsHTTPError(t *testing.T) {
	h := Handler(func(http.ResponseWriter, *http.Request) error {
		return Error(http.StatusConflict, "Conflict", "already exists")
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerHidesUnknownErrors(t *testing.T) {
	h := Handler(func(http.ResponseWriter, *http.Request) error {
		return errors.New("database password leaked")
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "database password") {
		t.Fatalf("leaked internal error: %s", rec.Body.String())
	}
}

func TestRecovererConvertsPanic(t *testing.T) {
	h := Recoverer()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRecovererDoesNotWriteAfterCommittedHeader(t *testing.T) {
	h := Recoverer()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Fatalf("problem body was appended after committed header: %s", rec.Body.String())
	}
}

func TestHandlerDoesNotWriteErrorAfterCommittedHeader(t *testing.T) {
	h := Handler(func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusAccepted)
		return Error(http.StatusConflict, "Conflict", "already exists")
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Conflict") {
		t.Fatalf("problem body was appended after committed header: %s", rec.Body.String())
	}
}
