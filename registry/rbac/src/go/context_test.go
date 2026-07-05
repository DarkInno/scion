package rbac

import (
	"context"
	"testing"
)

func TestContextHelpersRoundTrip(t *testing.T) {
	ctx := context.Background()
	ctx = WithUser(ctx, "user-1")
	ctx = WithRoles(ctx, []string{"admin"})
	ctx = WithPermissions(ctx, []Permission{ParsePermission("posts:write")})

	if user, ok := GetUserFromContext(ctx); !ok || user != "user-1" {
		t.Fatalf("user = %q %v", user, ok)
	}
	if roles, ok := GetRolesFromContext(ctx); !ok || len(roles) != 1 || roles[0] != "admin" {
		t.Fatalf("roles = %+v %v", roles, ok)
	}
	if perms, ok := GetPermissionsFromContext(ctx); !ok || len(perms) != 1 || perms[0].String() != "posts:write" {
		t.Fatalf("perms = %+v %v", perms, ok)
	}
}
