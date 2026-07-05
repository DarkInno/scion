package rbac

import (
	"strings"
)

// Permission represents a granular action on a resource.
// Format: "<resource>:<action>", e.g. "article:read", "user:delete".
// Wildcard "*" matches any resource or action: "article:*", "*:*".
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// String returns the permission string "resource:action".
func (p Permission) String() string {
	return p.Resource + ":" + p.Action
}

// ParsePermission parses a "resource:action" string into a Permission.
func ParsePermission(s string) Permission {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return Permission{Resource: parts[0], Action: parts[1]}
	}
	return Permission{Resource: s, Action: "*"}
}

// Role represents a named collection of permissions with optional inheritance.
type Role struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
	Parents     []string     `json:"parents,omitempty"` // parent role names
}

// User holds the roles assigned to a user.
type User struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}

// maxNameLen limits role and permission name lengths.
const maxNameLen = 128

// validateName checks that a name is safe: non-empty, within length, no CRLF/null.
func validateName(name string) bool {
	if name == "" || len(name) > maxNameLen {
		return false
	}
	return !strings.ContainsAny(name, "\r\n\x00")
}

// matches checks if a permission p matches the required permission,
// supporting wildcard "*" for resource and action.
func matches(p, required Permission) bool {
	resourceMatch := p.Resource == "*" || p.Resource == required.Resource
	actionMatch := p.Action == "*" || p.Action == required.Action
	return resourceMatch && actionMatch
}
