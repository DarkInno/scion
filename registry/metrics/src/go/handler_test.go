package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerExposesMetrics(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	m.observe("GET", "/health", http.StatusOK, 0.01)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "scion_http_requests_total") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestNilHandlerIsNotFound(t *testing.T) {
	var m *Metrics
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
