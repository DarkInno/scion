package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ctxKey is used to store/retrieve the call log slice from context
// without colliding with the package's own contextKey.
type ctxKey int

const callLogKey ctxKey = iota

// middleware returns a func(http.Handler)http.Handler that appends name+"-in"
// on entry and name+"-out" on exit to a shared *[]string attached via context.
func middleware(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := r.Context().Value(callLogKey).(*[]string)
			*log = append(*log, name+"-in")
			next.ServeHTTP(w, r)
			*log = append(*log, name+"-out")
		})
	}
}

// handler returns an http.HandlerFunc that appends "h" to the shared call log.
func handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := r.Context().Value(callLogKey).(*[]string)
		*log = append(*log, "h")
		w.WriteHeader(http.StatusOK)
	}
}

// newRequest creates a GET request with the shared call-log slice in its context.
func newRequest(log *[]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), callLogKey, log)
	return req.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestChainOrder(t *testing.T) {
	var log []string
	c := Chain(middleware("m1"), middleware("m2"))
	h := c.Then(handler())

	req := newRequest(&log)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	want := []string{"m1-in", "m2-in", "h", "m2-out", "m1-out"}
	if !slicesEqual(log, want) {
		t.Fatalf("call order mismatch\n got: %v\nwant: %v", log, want)
	}
}

func TestChainNilFinal(t *testing.T) {
	var log []string
	c := Chain(middleware("m1"))
	// Then(nil) should use http.NotFoundHandler -> 404
	h := c.Then(nil)

	req := newRequest(&log)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	// Middleware should still have run.
	want := []string{"m1-in", "m1-out"}
	if !slicesEqual(log, want) {
		t.Fatalf("call order mismatch\n got: %v\nwant: %v", log, want)
	}
}

func TestChainNilMiddleware(t *testing.T) {
	var log []string
	var nilMW func(http.Handler) http.Handler
	c := Chain(middleware("m1"), nilMW, middleware("m2"))
	h := c.Then(handler())

	req := newRequest(&log)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req) // must not panic

	want := []string{"m1-in", "m2-in", "h", "m2-out", "m1-out"}
	if !slicesEqual(log, want) {
		t.Fatalf("nil middleware was not skipped\n got: %v\nwant: %v", log, want)
	}
}

func TestChainImmutable(t *testing.T) {
	var log1, log2 []string
	c1 := Chain(middleware("m1"))
	c2 := c1.Append(middleware("m2"))

	h1 := c1.Then(handler())
	h2 := c2.Then(handler())

	// h1 should NOT include m2
	req1 := newRequest(&log1)
	rec1 := httptest.NewRecorder()
	h1.ServeHTTP(rec1, req1)
	want1 := []string{"m1-in", "h", "m1-out"}
	if !slicesEqual(log1, want1) {
		t.Fatalf("c1 should not include m2\n got: %v\nwant: %v", log1, want1)
	}

	// h2 should include both m1 and m2
	req2 := newRequest(&log2)
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, req2)
	want2 := []string{"m1-in", "m2-in", "h", "m2-out", "m1-out"}
	if !slicesEqual(log2, want2) {
		t.Fatalf("c2 should include m1 and m2\n got: %v\nwant: %v", log2, want2)
	}
}

func TestChainThenFunc(t *testing.T) {
	c := Chain(middleware("m1"))

	// ThenFunc should behave identically to Then(http.HandlerFunc(...))
	var tfLog, thLog []string

	hFunc := c.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		l := r.Context().Value(callLogKey).(*[]string)
		*l = append(*l, "h")
	})
	hHandler := c.Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := r.Context().Value(callLogKey).(*[]string)
		*l = append(*l, "h")
	}))

	reqF := newRequest(&tfLog)
	recF := httptest.NewRecorder()
	hFunc.ServeHTTP(recF, reqF)

	reqH := newRequest(&thLog)
	recH := httptest.NewRecorder()
	hHandler.ServeHTTP(recH, reqH)

	if !slicesEqual(tfLog, thLog) {
		t.Fatalf("ThenFunc and Then produced different call logs\nThenFunc: %v\nThen:     %v", tfLog, thLog)
	}
	if recF.Code != recH.Code {
		t.Fatalf("ThenFunc and Then produced different status codes: %d vs %d", recF.Code, recH.Code)
	}
}

func TestChainEmpty(t *testing.T) {
	var log []string
	c := Chain() // empty
	h := c.Then(handler())

	req := newRequest(&log)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	want := []string{"h"}
	if !slicesEqual(log, want) {
		t.Fatalf("empty chain should call handler directly\n got: %v\nwant: %v", log, want)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkChain10(b *testing.B) {
	// Build 10 empty (passthrough) middlewares.
	mws := make([]func(http.Handler) http.Handler, 10)
	for i := range mws {
		mws[i] = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(mws...).Then(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
// Utility
// ---------------------------------------------------------------------------

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
