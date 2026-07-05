# RBAC Module

Role-based access control with wildcard permissions and hierarchy inheritance.

## What's Included

- Role and permission models
- Permission wildcard matching
- Role inheritance
- Cycle detection
- Context helpers
- `net/http` authorization middleware

## Quick Copy

```bash
cp -r registry/rbac/src/go/*.go yourproject/internal/rbac/
```

Or with the Scion CLI:

```bash
scion add rbac --to internal/rbac
```

## Usage

```go
manager := rbac.NewManager()
_ = manager.AddRole(&rbac.Role{
	Name: "admin",
	Permissions: []rbac.Permission{
		rbac.ParsePermission("*:*"),
	},
})
_ = manager.AssignRole("user-1", "admin")

handler := manager.RequirePermission("posts:write")(next)
ctx := rbac.WithUser(r.Context(), "user-1")
handler.ServeHTTP(w, r.WithContext(ctx))
```

## File Reference

| File | Purpose |
|------|---------|
| `model.go` | Role and permission model |
| `manager.go` | Role registration, matching, and hierarchy |
| `context.go` | Context role helpers |
| `middleware.go` | HTTP authorization middleware |
| `pentest_test.go` | Permission bypass tests |

## Tests

```bash
cd registry/rbac/src/go
go test -v ./...
```
