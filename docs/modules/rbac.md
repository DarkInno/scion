# RBAC Module

Role-based access control with wildcard permissions and hierarchy inheritance.

## What's Included

- Role definition and management
- Permission assignment with wildcard support
- Role hierarchy with inheritance
- Cycle detection
- HTTP middleware for permission checking

## Quick Copy

```bash
cp -r registry/rbac/src/go/* yourproject/internal/rbac/
```

## Usage

### Define Roles and Permissions

```go
manager := rbac.NewManager()

// Define roles
manager.AddRole(rbac.Role{
    ID: "admin",
    Permissions: []string{"*"},
})

manager.AddRole(rbac.Role{
    ID: "editor",
    Permissions: []string{"posts:*", "comments:read"},
})

manager.AddRole(rbac.Role{
    ID: "viewer",
    Permissions: []string{"*:read"},
})

// Set hierarchy (editor inherits viewer)
manager.SetParent("editor", "viewer")
```

### Use Middleware

```go
// Require specific permission
handler := rbac.Require("posts:write")(handler)

// Require any of multiple permissions
handler := rbac.RequireAny("posts:write", "posts:delete")(handler)

// Require all permissions
handler := rbac.RequireAll("posts:write", "comments:write")(handler)
```

### Set User Roles

```go
// In your auth middleware
ctx := rbac.WithRoles(ctx, []string{"editor"})
```

## Permission Format

Permissions use `resource:action` format with wildcard support:

- `posts:read` — read posts
- `posts:*` — all actions on posts
- `*:read` — read any resource
- `*` — full access

## File Reference

| File | Purpose |
|------|---------|
| `model.go` | Role and Permission types |
| `manager.go` | Role/permission management |
| `middleware.go` | HTTP middleware |
| `context.go` | Context helpers |

## Security Features

- Wildcard permission matching
- Cycle detection in role hierarchy
- Hierarchy inheritance

## Tests

```bash
cd registry/rbac/src/go
go test -v ./...
```
