package migrations

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStatusHandlerReturnsJSON(t *testing.T) {
	db, _ := openFakeDB(t)
	fsys := fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("CREATE TABLE users(id BIGINT);")},
	}
	m, err := New(fsys)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	rec := httptest.NewRecorder()
	m.StatusHandler(db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/migrations", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"version":20260101000001`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestStatusHandlerRecoversPanic(t *testing.T) {
	m, err := New(fstest.MapFS{
		"20260101000001_add_users.up.sql": {Data: []byte("CREATE TABLE users(id BIGINT);")},
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	rec := httptest.NewRecorder()
	m.StatusHandler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/migrations", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
