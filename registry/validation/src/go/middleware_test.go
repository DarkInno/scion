package validation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareRejectsInvalidRequest(t *testing.T) {
	schema := New(WithSource(QuerySource)).Field("email").Required().Email()
	handler := Middleware(schema)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?email=bad", nil))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "email") {
		t.Fatalf("invalid response code=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?email=a@example.com", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("valid response code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestMiddlewareRecoversValidationPanic(t *testing.T) {
	schema := New(WithSource(QuerySource)).
		Field("value").
		Custom("panic", func(string) error { panic("boom") })
	handler := Middleware(schema)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?value=x", nil))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "_server") {
		t.Fatalf("panic response code=%d body=%q", rec.Code, rec.Body.String())
	}
}
