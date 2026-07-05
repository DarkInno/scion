package rbac

import (
	"encoding/json"
	"net/http"
)

// RequirePermission returns a middleware that checks if the user has the given permission.
// The user ID is extracted from the request context (set by WithUser or auth middleware).
// Returns 401 if not authenticated, 403 if lacking permission.
func (m *Manager) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			if !m.HasPermission(userID, permission) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "insufficient permissions",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns a middleware that checks if the user has the given role.
// Returns 401 if not authenticated, 403 if lacking the role.
func (m *Manager) RequireRole(roleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			roles, err := m.GetUserRoles(userID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "insufficient permissions",
				})
				return
			}

			hasRole := false
			for _, r := range roles {
				if r == roleName {
					hasRole = true
					break
				}
			}

			if !hasRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "insufficient permissions",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission checks if the user has at least one of the given permissions.
func (m *Manager) RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := GetUserFromContext(r.Context())
			if !ok || userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "authentication required",
				})
				return
			}

			for _, perm := range permissions {
				if m.HasPermission(userID, perm) {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "insufficient permissions",
			})
		})
	}
}
