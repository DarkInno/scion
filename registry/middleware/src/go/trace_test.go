package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTraceContext_GeneratesNew(t *testing.T) {
	mw := TraceContext()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		spanID := GetSpanID(r.Context())
		if traceID == "" {
			t.Error("expected trace ID in context")
		}
		if spanID == "" {
			t.Error("expected span ID in context")
		}
		if len(traceID) != 32 {
			t.Errorf("expected trace ID length 32, got %d", len(traceID))
		}
		if len(spanID) != 16 {
			t.Errorf("expected span ID length 16, got %d", len(spanID))
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Response should contain traceparent header.
	tp := rec.Header().Get("traceparent")
	if tp == "" {
		t.Error("expected traceparent in response header")
	}
}

func TestTraceContext_PropagatesIncoming(t *testing.T) {
	mw := TraceContext()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		if traceID != "0af7651916cd43dd8448eb211c80319c" {
			t.Errorf("expected trace ID to propagate, got %s", traceID)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tp := rec.Header().Get("traceparent")
	if !strings.Contains(tp, "0af7651916cd43dd8448eb211c80319c") {
		t.Errorf("expected propagated traceparent in response, got %s", tp)
	}
}

func TestTraceContext_InvalidVersion(t *testing.T) {
	mw := TraceContext()
	var gotTraceID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = GetTraceID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "ff-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Invalid version should cause generation of a new traceparent.
	if gotTraceID == "0af7651916cd43dd8448eb211c80319c" {
		t.Error("expected new trace ID to be generated for invalid version")
	}
	if gotTraceID == "" {
		t.Error("expected a new trace ID to be generated")
	}
}

func TestTraceContext_InvalidTraceID(t *testing.T) {
	mw := TraceContext()
	var gotTraceID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = GetTraceID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-INVALID-b7ad6b7169203331-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID == "INVALID" || gotTraceID == "" {
		t.Error("expected new trace ID to be generated for invalid trace ID")
	}
}

func TestTraceContext_InvalidSpanID(t *testing.T) {
	mw := TraceContext()
	var gotSpanID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSpanID = GetSpanID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-INVALID-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotSpanID == "INVALID" || gotSpanID == "" {
		t.Error("expected new span ID to be generated for invalid span ID")
	}
}

func TestTraceContext_InvalidFlags(t *testing.T) {
	mw := TraceContext()
	var gotTraceID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = GetTraceID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-GG")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID == "0af7651916cd43dd8448eb211c80319c" {
		t.Error("expected new trace ID to be generated for invalid flags")
	}
}

func TestTraceContext_BaggagePropagation(t *testing.T) {
	mw := TraceContext()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Context should contain baggage.
		if baggage, ok := r.Context().Value(baggageKey).(string); !ok || baggage == "" {
			t.Error("expected baggage in context")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	req.Header.Set("baggage", "key1=value1,key2=value2")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("baggage") == "" {
		t.Error("expected baggage in response header")
	}
}

func TestTraceContext_CustomHeaderName(t *testing.T) {
	mw := TraceContext(TraceOptions{HeaderName: "custom-traceparent"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := GetTraceID(r.Context())
		if traceID == "" {
			t.Error("expected trace ID")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("custom-traceparent") == "" {
		t.Error("expected custom traceparent header in response")
	}
}

func TestTraceContext_MalformedTraceparent(t *testing.T) {
	mw := TraceContext()
	var gotTraceID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = GetTraceID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", "not-a-valid-traceparent")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID == "" {
		t.Error("expected new trace ID to be generated for malformed traceparent")
	}
}

func TestGetTraceID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	if id := GetTraceID(ctx); id != "" {
		t.Errorf("expected empty trace ID, got %s", id)
	}
}

func TestGetSpanID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	if id := GetSpanID(ctx); id != "" {
		t.Errorf("expected empty span ID, got %s", id)
	}
}

func BenchmarkTraceContext(b *testing.B) {
	mw := TraceContext()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/bench", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
