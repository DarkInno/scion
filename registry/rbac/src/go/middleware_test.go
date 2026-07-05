package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareRequirePermissionHTTPFlow(t *testing.T) {
	manager := NewManager()
	if err := manager.AddRole(&Role{Name: "editor", Permissions: []Permission{ParsePermission("posts:write")}}); err != nil {
		t.Fatalf("AddRole: %v", err)
	}
	if err := manager.AssignRole("user-1", "editor"); err != nil {
		t.Fatalf("AssignRole: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := manager.RequirePermission("posts:write")(next)

	unauth := httptest.NewRecorder()
	handler.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/", nil))
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth code = %d", unauth.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(req.Context(), "user-1"))
	ok := httptest.NewRecorder()
	handler.ServeHTTP(ok, req)
	if ok.Code != http.StatusNoContent {
		t.Fatalf("allowed code = %d", ok.Code)
	}
}
