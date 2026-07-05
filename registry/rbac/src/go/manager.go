package rbac

import (
	"errors"
	"sync"
)

// ErrRoleNotFound is returned when a role does not exist.
var ErrRoleNotFound = errors.New("role not found")

// ErrRoleExists is returned when a role already exists.
var ErrRoleExists = errors.New("role already exists")

// ErrCircularDependency is returned when adding a parent would create a cycle.
var ErrCircularDependency = errors.New("circular dependency detected in role hierarchy")

// ErrInvalidName is returned when a name fails validation.
var ErrInvalidName = errors.New("invalid name: must be non-empty, <= 128 chars, no CRLF or null bytes")

// Manager manages roles, users, and permission checks.
// All methods are safe for concurrent use.
type Manager struct {
	mu    sync.RWMutex
	roles map[string]*Role
	users map[string]*User
}

// NewManager creates a new RBAC manager.
func NewManager() *Manager {
	return &Manager{
		roles: make(map[string]*Role),
		users: make(map[string]*User),
	}
}

// AddRole adds a new role. Returns ErrRoleExists if the role already exists
// and ErrInvalidName if the name is invalid.
func (m *Manager) AddRole(role *Role) error {
	if role == nil || !validateName(role.Name) {
		return ErrInvalidName
	}
	for _, p := range role.Permissions {
		if !validateName(p.Resource) || !validateName(p.Action) {
			return ErrInvalidName
		}
	}
	for _, parent := range role.Parents {
		if !validateName(parent) {
			return ErrInvalidName
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.Name]; exists {
		return ErrRoleExists
	}

	// Check for circular dependencies before adding.
	for _, parent := range role.Parents {
		if parent == role.Name {
			return ErrCircularDependency
		}
		if m.wouldCreateCycle(parent, role.Name, make(map[string]bool)) {
			return ErrCircularDependency
		}
	}

	// Deep copy the role to prevent external mutation.
	r := &Role{
		Name:        role.Name,
		Permissions: make([]Permission, len(role.Permissions)),
		Parents:     make([]string, len(role.Parents)),
	}
	copy(r.Permissions, role.Permissions)
	copy(r.Parents, role.Parents)
	m.roles[role.Name] = r
	return nil
}

// wouldCreateCycle checks if making child a descendant of ancestor would create a cycle.
// Caller must hold the lock.
func (m *Manager) wouldCreateCycle(ancestor, child string, visited map[string]bool) bool {
	if ancestor == child {
		return true
	}
	if visited[ancestor] {
		return true
	}
	visited[ancestor] = true

	role, exists := m.roles[ancestor]
	if !exists {
		return false
	}
	for _, parent := range role.Parents {
		if m.wouldCreateCycle(parent, child, visited) {
			return true
		}
	}
	return false
}

// GetRole returns a role by name.
func (m *Manager) GetRole(name string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[name]
	if !exists {
		return nil, ErrRoleNotFound
	}
	// Return a copy.
	r := &Role{
		Name:        role.Name,
		Permissions: make([]Permission, len(role.Permissions)),
		Parents:     make([]string, len(role.Parents)),
	}
	copy(r.Permissions, role.Permissions)
	copy(r.Parents, role.Parents)
	return r, nil
}

// DeleteRole removes a role. Other roles referencing it as parent are unaffected
// (the reference simply becomes dangling).
func (m *Manager) DeleteRole(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[name]; !exists {
		return ErrRoleNotFound
	}
	delete(m.roles, name)
	return nil
}

// AssignRole assigns a role to a user.
func (m *Manager) AssignRole(userID, roleName string) error {
	if !validateName(userID) || !validateName(roleName) {
		return ErrInvalidName
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[roleName]; !exists {
		return ErrRoleNotFound
	}

	user, exists := m.users[userID]
	if !exists {
		user = &User{ID: userID}
		m.users[userID] = user
	}

	// Check if already assigned.
	for _, r := range user.Roles {
		if r == roleName {
			return nil // already assigned, idempotent
		}
	}
	user.Roles = append(user.Roles, roleName)
	return nil
}

// RevokeRole removes a role from a user.
func (m *Manager) RevokeRole(userID, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return nil // idempotent
	}

	for i, r := range user.Roles {
		if r == roleName {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			return nil
		}
	}
	return nil
}

// GetUserRoles returns all roles assigned to a user (including inherited).
func (m *Manager) GetUserRoles(userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, nil
	}

	visited := make(map[string]bool)
	var roles []string
	m.collectRoles(user.Roles, visited, &roles)
	return roles, nil
}

// collectRoles recursively collects roles including parents.
// Caller must hold the read lock.
func (m *Manager) collectRoles(roleNames []string, visited map[string]bool, result *[]string) {
	for _, name := range roleNames {
		if visited[name] {
			continue
		}
		visited[name] = true
		*result = append(*result, name)

		role, exists := m.roles[name]
		if !exists {
			continue
		}
		m.collectRoles(role.Parents, visited, result)
	}
}

// HasPermission checks if a user has a specific permission (directly or inherited).
func (m *Manager) HasPermission(userID, permission string) bool {
	required := ParsePermission(permission)

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return false
	}

	visited := make(map[string]bool)
	return m.checkPermission(user.Roles, required, visited)
}

// checkPermission recursively checks if any role (including parents) grants the permission.
// Caller must hold the read lock.
func (m *Manager) checkPermission(roleNames []string, required Permission, visited map[string]bool) bool {
	for _, name := range roleNames {
		if visited[name] {
			continue
		}
		visited[name] = true

		role, exists := m.roles[name]
		if !exists {
			continue
		}

		for _, p := range role.Permissions {
			if matches(p, required) {
				return true
			}
		}

		// Check parent roles.
		if m.checkPermission(role.Parents, required, visited) {
			return true
		}
	}
	return false
}

// GetAllPermissions returns all permissions for a user (including inherited).
func (m *Manager) GetAllPermissions(userID string) []Permission {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil
	}

	visited := make(map[string]bool)
	var perms []Permission
	m.collectPermissions(user.Roles, &perms, visited)
	return perms
}

// collectPermissions recursively collects all permissions from roles and their parents.
// Caller must hold the read lock.
func (m *Manager) collectPermissions(roleNames []string, perms *[]Permission, visited map[string]bool) {
	for _, name := range roleNames {
		if visited[name] {
			continue
		}
		visited[name] = true

		role, exists := m.roles[name]
		if !exists {
			continue
		}

		*perms = append(*perms, role.Permissions...)
		m.collectPermissions(role.Parents, perms, visited)
	}
}

// WithUser stores a user in the context.
// (Defined in context.go)
