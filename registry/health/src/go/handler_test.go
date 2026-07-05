package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerLivenessAndReadiness(t *testing.T) {
	checker := New(WithCacheTTL(0))
	fail, err := NewCustomCheck("database", func(ctx context.Context) error {
		return errors.New("down")
	})
	if err != nil {
		t.Fatalf("NewCustomCheck: %v", err)
	}
	if err := checker.AddCheck(fail); err != nil {
		t.Fatalf("AddCheck: %v", err)
	}
	handler := NewHealthHandler(checker)

	live := httptest.NewRecorder()
	handler.Liveness(live, httptest.NewRequest(http.MethodGet, "/live", nil))
	if live.Code != http.StatusOK || !strings.Contains(live.Body.String(), StatusHealthy) {
		t.Fatalf("liveness response: code=%d body=%q", live.Code, live.Body.String())
	}

	ready := httptest.NewRecorder()
	handler.Readiness(ready, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if ready.Code != http.StatusServiceUnavailable || !strings.Contains(ready.Body.String(), StatusNotReady) {
		t.Fatalf("readiness response: code=%d body=%q", ready.Code, ready.Body.String())
	}
}
