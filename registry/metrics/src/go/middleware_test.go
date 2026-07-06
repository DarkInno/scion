package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRecordsStatus(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	h := m.Middleware("/users/{id}")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/users/123", nil))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `scion_http_requests_total{method="POST",route="/users/{id}",status="201"} 1`) {
		t.Fatalf("metric missing from body:\n%s", body)
	}
}

func TestMiddlewareDefaultsImplicitStatus(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	h := m.Instrument("/ok", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rec.Body.String(), `status="200"`) {
		t.Fatalf("expected implicit 200 metric:\n%s", rec.Body.String())
	}
}

func TestMiddlewareRecordsPanicBeforeReraising(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	h := m.Instrument("/panic", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("expected panic")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
	}()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rec.Body.String(), `route="/panic",status="500"`) {
		t.Fatalf("panic metric missing:\n%s", rec.Body.String())
	}
}

func TestResponseWriterSuppressesDuplicateHeaderAndUnwraps(t *testing.T) {
	rec := httptest.NewRecorder()
	lw := &responseWriter{ResponseWriter: rec}
	lw.WriteHeader(http.StatusCreated)
	lw.WriteHeader(http.StatusInternalServerError)
	if rec.Code != http.StatusCreated || lw.status != http.StatusCreated {
		t.Fatalf("status rec=%d wrapped=%d", rec.Code, lw.status)
	}
	if lw.Unwrap() != rec {
		t.Fatalf("unwrap did not return underlying writer")
	}
}
