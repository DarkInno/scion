package rbac

import "context"

// contextKey is an unexported type so that context keys defined in this
// package cannot collide with keys from any other package.
type contextKey int

const (
	keyUser contextKey = iota
	keyRoles
	keyPermissions
)

// WithUser stores the authenticated user ID in the context.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, keyUser, userID)
}

// GetUserFromContext returns the authenticated user ID previously stored by
// WithUser. The boolean is false when no user is present.
func GetUserFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyUser).(string)
	return v, ok
}

// WithRoles stores the user's effective role names in the context.
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, keyRoles, roles)
}

// GetRolesFromContext returns the effective role names previously stored by
// WithRoles. The boolean is false when no roles are present.
func GetRolesFromContext(ctx context.Context) ([]string, bool) {
	v, ok := ctx.Value(keyRoles).([]string)
	return v, ok
}

// WithPermissions stores the user's effective permissions in the context.
func WithPermissions(ctx context.Context, perms []Permission) context.Context {
	return context.WithValue(ctx, keyPermissions, perms)
}

// GetPermissionsFromContext returns the effective permissions previously
// stored by WithPermissions. The boolean is false when none are present.
func GetPermissionsFromContext(ctx context.Context) ([]Permission, bool) {
	v, ok := ctx.Value(keyPermissions).([]Permission)
	return v, ok
}
