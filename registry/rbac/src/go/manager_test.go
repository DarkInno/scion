package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestAddRole(t *testing.T) {
	m := NewManager()
	err := m.AddRole(&Role{
		Name: "admin",
		Permissions: []Permission{
			{Resource: "*", Action: "*"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate should fail.
	err = m.AddRole(&Role{Name: "admin"})
	if err != ErrRoleExists {
		t.Errorf("expected ErrRoleExists, got %v", err)
	}
}

func TestAddRoleInvalidName(t *testing.T) {
	m := NewManager()
	tests := []string{
		"",
		strings.Repeat("a", 129),
		"name\r\n",
		"name\x00",
	}
	for _, name := range tests {
		err := m.AddRole(&Role{Name: name})
		if err != ErrInvalidName {
			t.Errorf("name %q: expected ErrInvalidName, got %v", name, err)
		}
	}
}

func TestRoleInheritance(t *testing.T) {
	m := NewManager()

	// Base role: reader
	_ = m.AddRole(&Role{
		Name: "reader",
		Permissions: []Permission{
			{Resource: "article", Action: "read"},
		},
	})

	// Derived role: editor inherits reader, adds write
	_ = m.AddRole(&Role{
		Name: "editor",
		Permissions: []Permission{
			{Resource: "article", Action: "write"},
		},
		Parents: []string{"reader"},
	})

	_ = m.AssignRole("user1", "editor")

	if !m.HasPermission("user1", "article:write") {
		t.Error("editor should have write permission")
	}
	if !m.HasPermission("user1", "article:read") {
		t.Error("editor should inherit read permission from reader")
	}
	if m.HasPermission("user1", "article:delete") {
		t.Error("editor should not have delete permission")
	}
}

func TestCircularDependency(t *testing.T) {
	m := NewManager()

	_ = m.AddRole(&Role{
		Name:    "roleA",
		Parents: []string{},
	})

	_ = m.AddRole(&Role{
		Name:    "roleB",
		Parents: []string{"roleA"},
	})

	// roleA -> roleB would create a cycle (roleB already inherits roleA).
	err := m.AddRole(&Role{
		Name:    "roleA-updated",
		Parents: []string{"roleB"},
	})
	_ = err // This is a new role, not updating roleA.

	// Direct self-reference.
	err = m.AddRole(&Role{
		Name:    "selfRef",
		Parents: []string{"selfRef"},
	})
	if err != ErrCircularDependency {
		t.Errorf("expected ErrCircularDependency for self-reference, got %v", err)
	}
}

func TestWildcardPermission(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{
		Name: "superadmin",
		Permissions: []Permission{
			{Resource: "*", Action: "*"},
		},
	})
	_ = m.AssignRole("root", "superadmin")

	tests := []string{
		"article:read",
		"article:write",
		"article:delete",
		"user:read",
		"user:delete",
		"anything:anything",
	}
	for _, perm := range tests {
		if !m.HasPermission("root", perm) {
			t.Errorf("superadmin should have %s", perm)
		}
	}
}

func TestRevokeRole(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "admin", Permissions: []Permission{{Resource: "*", Action: "*"}}})
	_ = m.AssignRole("user1", "admin")

	if !m.HasPermission("user1", "article:read") {
		t.Error("should have permission before revoke")
	}

	_ = m.RevokeRole("user1", "admin")

	if m.HasPermission("user1", "article:read") {
		t.Error("should not have permission after revoke")
	}
}

func TestDeleteRole(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "temp", Permissions: []Permission{{Resource: "x", Action: "y"}}})
	_ = m.AssignRole("user1", "temp")

	_ = m.DeleteRole("temp")

	if m.HasPermission("user1", "x:y") {
		t.Error("should not have permission after role deleted")
	}
}

func TestGetAllPermissions(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "reader", Permissions: []Permission{{Resource: "article", Action: "read"}}})
	_ = m.AddRole(&Role{Name: "writer", Permissions: []Permission{{Resource: "article", Action: "write"}}, Parents: []string{"reader"}})
	_ = m.AssignRole("user1", "writer")

	perms := m.GetAllPermissions("user1")
	if len(perms) < 2 {
		t.Errorf("expected at least 2 permissions, got %d", len(perms))
	}
}

func TestMiddlewareRequirePermission(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "editor", Permissions: []Permission{{Resource: "article", Action: "write"}}})
	_ = m.AssignRole("user1", "editor")

	handler := m.RequirePermission("article:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No user in context → 401.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without user, got %d", rec.Code)
	}

	// User with permission → 200.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(context.Background(), "user1"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with permission, got %d", rec.Code)
	}

	// User without permission → 403.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(context.Background(), "user2"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 without permission, got %d", rec.Code)
	}
}

func TestMiddlewareRequireRole(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "admin", Permissions: []Permission{{Resource: "*", Action: "*"}}})
	_ = m.AssignRole("user1", "admin")

	handler := m.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(context.Background(), "user1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := NewManager()
	_ = m.AddRole(&Role{Name: "user", Permissions: []Permission{{Resource: "data", Action: "read"}}})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := "user"
			_ = m.AssignRole(userID, "user")
			_ = m.HasPermission(userID, "data:read")
			_, _ = m.GetUserRoles(userID)
		}(i)
	}
	wg.Wait()
}

func TestManagerCapacityLimits(t *testing.T) {
	m := NewManager()
	for i := 0; i < maxRoles; i++ {
		name := "role" + strconv.Itoa(i)
		if err := m.AddRole(&Role{Name: name}); err != nil {
			t.Fatalf("AddRole %d: %v", i, err)
		}
	}
	if err := m.AddRole(&Role{Name: "overflow"}); err != ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded for role overflow, got %v", err)
	}

	users := NewManager()
	if err := users.AddRole(&Role{Name: "base"}); err != nil {
		t.Fatalf("AddRole base: %v", err)
	}
	for i := 0; i < maxUsers; i++ {
		if err := users.AssignRole("user"+strconv.Itoa(i), "base"); err != nil {
			t.Fatalf("AssignRole %d: %v", i, err)
		}
	}
	if err := users.AssignRole("overflow", "base"); err != ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded for user overflow, got %v", err)
	}
}

func TestRoleSliceLimits(t *testing.T) {
	m := NewManager()
	perms := make([]Permission, maxRolePermissions+1)
	for i := range perms {
		perms[i] = Permission{Resource: "r", Action: "a"}
	}
	if err := m.AddRole(&Role{Name: "too-many-perms", Permissions: perms}); err != ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded for permissions, got %v", err)
	}

	parents := make([]string, maxRoleParents+1)
	for i := range parents {
		parents[i] = "parent" + strconv.Itoa(i)
	}
	if err := m.AddRole(&Role{Name: "too-many-parents", Parents: parents}); err != ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded for parents, got %v", err)
	}
}
