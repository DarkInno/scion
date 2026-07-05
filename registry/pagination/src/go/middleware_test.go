package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareStoresPaginationContext(t *testing.T) {
	mw := Middleware(Options{DefaultLimit: 5, MaxLimit: 10})
	req := httptest.NewRequest(http.MethodGet, "/items?limit=9&cursor=bad!!!", nil)
	rec := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if opts, ok := OptionsFromContext(r.Context()); !ok || opts.MaxLimit != 10 {
			t.Fatalf("missing options: %+v %v", opts, ok)
		}
		if offset, ok := OffsetFromContext(r.Context()); !ok || offset.Limit != 9 {
			t.Fatalf("missing offset: %+v %v", offset, ok)
		}
		if cursor, ok := CursorFromContext(r.Context()); !ok || cursor.Limit != 9 {
			t.Fatalf("missing cursor: %+v %v", cursor, ok)
		}
		if CursorErrorFromContext(r.Context()) == nil {
			t.Fatal("expected cursor parse error")
		}
	})).ServeHTTP(rec, req)
	if !called {
		t.Fatal("next handler was not called")
	}
}
