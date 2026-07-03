// Package casbin provides authorization helpers backed by Casbin.
package casbin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// Policy types for Casbin
const (
	PolicyTypePermission = "p" // role -> permission mapping
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . Client

// Client defines the interface for Casbin operations
type Client interface {
	// GetEnforcer returns the underlying Casbin enforcer
	GetEnforcer() *casbin.Enforcer

	// Role-Permission management
	AddRolePermissions(roleID uint, permissions []string) error
	RemoveRolePermissions(roleID uint, permissions []string) error
	SetRolePermissions(roleID uint, permissions []string) error
	GetRolePermissions(roleID uint) []string

	// Permission checking
	CheckPermission(roleID uint, permission string) (bool, error)
	CheckPermissionWithRoot(userRole string, adminRoleID *uint, permission string) (bool, error)

	// Cleanup
	DeleteRole(roleID uint) error
}

type client struct {
	// enforcer is a SyncedEnforcer, not a bare Enforcer: this client is a
	// process-wide fx singleton whose policies are READ on every authorized
	// request (Enforce/GetFilteredPolicy) and WRITTEN whenever an admin edits a
	// role (Add/Remove/Set/DeleteRole). The base casbin.Enforcer guards its
	// policy store with plain maps and no locking, so a concurrent read+write
	// would trigger a fatal "concurrent map read and map write" and crash the
	// whole process. SyncedEnforcer wraps every method below with an RWMutex.
	enforcer *casbin.SyncedEnforcer
}

// New creates a new Casbin client instance
func New(db *gorm.DB) (Client, error) {
	// Initialize casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize casbin adapter: %w", err)
	}

	// Create casbin RBAC model
	// Simple model: role has permissions
	// Format: p, role_id, resource, action
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")                                                        // Request: role_id, resource, action
	m.AddDef("p", "p", "sub, obj, act")                                                        // Policy: role_id, resource, action
	m.AddDef("e", "e", "some(where (p.eft == allow))")                                         // Effect: allow if any policy matches
	m.AddDef("m", "m", "r.sub == p.sub && keyMatch2(r.obj, p.obj) && keyMatch2(r.act, p.act)") // Matcher

	// Create a synced (mutex-guarded) enforcer with model and adapter. See the
	// client.enforcer field comment for why the synced variant is required.
	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// Load policies from database
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policy from DB: %w", err)
	}

	return &client{
		enforcer: enforcer,
	}, nil
}

func (c *client) GetEnforcer() *casbin.Enforcer {
	// Return the embedded base enforcer to keep the interface signature stable.
	// Callers must not use it for concurrent policy mutation — go through the
	// client's locked methods instead.
	return c.enforcer.Enforcer
}

// roleSubject converts role ID to Casbin subject format
func roleSubject(roleID uint) string {
	return fmt.Sprintf("role:%d", roleID)
}

// parsePermission splits "resource:action" into resource and action
func parsePermission(permission string) (resource, action string, err error) {
	parts := strings.SplitN(permission, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid permission format: %s (expected 'resource:action')", permission)
	}
	return parts[0], parts[1], nil
}

// AddRolePermissions adds permissions to a role
func (c *client) AddRolePermissions(roleID uint, permissions []string) error {
	subject := roleSubject(roleID)

	for _, perm := range permissions {
		resource, action, err := parsePermission(perm)
		if err != nil {
			return err
		}

		// Add policy: role_id, resource, action
		_, err = c.enforcer.AddPolicy(subject, resource, action)
		if err != nil {
			return fmt.Errorf("failed to add permission %s: %w", perm, err)
		}
	}

	return nil
}

// RemoveRolePermissions removes permissions from a role
func (c *client) RemoveRolePermissions(roleID uint, permissions []string) error {
	subject := roleSubject(roleID)

	for _, perm := range permissions {
		resource, action, err := parsePermission(perm)
		if err != nil {
			return err
		}

		_, err = c.enforcer.RemovePolicy(subject, resource, action)
		if err != nil {
			return fmt.Errorf("failed to remove permission %s: %w", perm, err)
		}
	}

	return nil
}

// SetRolePermissions replaces all permissions for a role.
//
// It diffs the desired set against the current one and applies only the
// difference via casbin's batch policy APIs, rather than deleting every grant
// and re-adding one at a time. This matters for two reasons:
//   - No zero-permission window: unchanged grants are never touched, so a
//     concurrent Enforce never sees the role momentarily stripped of a
//     permission it should keep.
//   - Atomic bulk ops: AddPolicies/RemovePolicies each persist all-or-nothing,
//     so a failure cannot leave the role with a half-applied permission set.
//
// New grants are added before stale ones are removed, so a failure between the
// two steps can only ever leave the role briefly over-permissioned (logged via
// the returned error), never missing a permission it should have.
func (c *client) SetRolePermissions(roleID uint, permissions []string) error {
	subject := roleSubject(roleID)

	// Build the desired rule set, validating every permission up front so a bad
	// entry fails the whole operation before any write happens.
	desiredSet := make(map[string]struct{}, len(permissions))
	desiredRules := make(map[string][]string, len(permissions))
	for _, perm := range permissions {
		resource, action, err := parsePermission(perm)
		if err != nil {
			return err
		}
		key := resource + ":" + action
		if _, dup := desiredSet[key]; dup {
			continue
		}
		desiredSet[key] = struct{}{}
		desiredRules[key] = []string{subject, resource, action}
	}

	current, err := c.enforcer.GetFilteredPolicy(0, subject)
	if err != nil {
		return fmt.Errorf("failed to read current permissions for role %d: %w", roleID, err)
	}
	currentSet := make(map[string]struct{}, len(current))
	var toRemove [][]string
	for _, p := range current {
		if len(p) < 3 {
			continue
		}
		key := p[1] + ":" + p[2]
		currentSet[key] = struct{}{}
		if _, keep := desiredSet[key]; !keep {
			toRemove = append(toRemove, []string{p[0], p[1], p[2]})
		}
	}

	var toAdd [][]string
	for key, rule := range desiredRules {
		if _, exists := currentSet[key]; !exists {
			toAdd = append(toAdd, rule)
		}
	}

	if len(toAdd) > 0 {
		if _, err := c.enforcer.AddPolicies(toAdd); err != nil {
			return fmt.Errorf("failed to add permissions for role %d: %w", roleID, err)
		}
	}
	if len(toRemove) > 0 {
		if _, err := c.enforcer.RemovePolicies(toRemove); err != nil {
			return fmt.Errorf("failed to remove stale permissions for role %d: %w", roleID, err)
		}
	}

	return nil
}

// GetRolePermissions returns all permissions for a role
func (c *client) GetRolePermissions(roleID uint) []string {
	subject := roleSubject(roleID)
	policies, _ := c.enforcer.GetFilteredPolicy(0, subject)

	permissions := make([]string, 0, len(policies))
	for _, policy := range policies {
		if len(policy) >= 3 {
			// Reconstruct "resource:action" format
			perm := policy[1] + ":" + policy[2]
			permissions = append(permissions, perm)
		}
	}

	return permissions
}

// CheckPermission checks if a role has a specific permission
func (c *client) CheckPermission(roleID uint, permission string) (bool, error) {
	resource, action, err := parsePermission(permission)
	if err != nil {
		return false, err
	}

	subject := roleSubject(roleID)

	// Check exact permission first
	allowed, err := c.enforcer.Enforce(subject, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	if allowed {
		return true, nil
	}

	// Check "manage" permission (grants ALL actions for a resource)
	allowed, err = c.enforcer.Enforce(subject, resource, "manage")
	if err != nil {
		return false, fmt.Errorf("failed to check manage permission: %w", err)
	}

	return allowed, nil
}

// CheckPermissionWithRoot checks permission with root bypass
// userRole: the user's role type (root, admin, user, etc.)
// adminRoleID: the admin_role_id if user is admin (can be nil)
// permission: the permission to check (e.g., "product:create")
func (c *client) CheckPermissionWithRoot(userRole string, adminRoleID *uint, permission string) (bool, error) {
	// Root bypasses all permission checks
	if userRole == "root" {
		return true, nil
	}

	// If user is not admin or has no admin role, deny
	if userRole != "admin" || adminRoleID == nil {
		return false, nil
	}

	// Check permission for the admin role
	return c.CheckPermission(*adminRoleID, permission)
}

// DeleteRole removes all permissions for a role
func (c *client) DeleteRole(roleID uint) error {
	subject := roleSubject(roleID)
	_, err := c.enforcer.RemoveFilteredPolicy(0, subject)
	if err != nil {
		return fmt.Errorf("failed to delete role permissions: %w", err)
	}
	return nil
}

// ParseRoleIDFromSubject converts a Casbin role subject back into its numeric role ID.
func ParseRoleIDFromSubject(subject string) (uint, error) {
	if !strings.HasPrefix(subject, "role:") {
		return 0, fmt.Errorf("invalid subject format: %s", subject)
	}
	idStr := strings.TrimPrefix(subject, "role:")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid role ID: %s", idStr)
	}
	return uint(id), nil
}
