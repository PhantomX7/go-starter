// Package permissions defines the permission registry used by authorization checks.
package permissions

import (
	"slices"
	"sort"
)

// Permission represents a single permission string (format: "resource:action")
type Permission string

func (p Permission) String() string {
	return string(p)
}

// Standard Actions (common across resources)
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionManage = "manage"
)

// Resources
const (
	ResourceAdminUser = "admin_user"
	ResourceAdminRole = "admin_role"
	ResourceConfig    = "config"
	ResourceLog       = "log"
	ResourceUser      = "user"
)

// ============================================================================
// ADMIN USER PERMISSIONS
// ============================================================================
const (
	AdminUserCreate         Permission = "admin_user:create"
	AdminUserRead           Permission = "admin_user:read"
	AdminUserUpdate         Permission = "admin_user:update"
	AdminUserDelete         Permission = "admin_user:delete"
	AdminUserChangePassword Permission = "admin_user:change_password"
)

// ============================================================================
// ADMIN ROLE PERMISSIONS
// ============================================================================
const (
	AdminRoleCreate Permission = "admin_role:create"
	AdminRoleRead   Permission = "admin_role:read"
	AdminRoleUpdate Permission = "admin_role:update"
	AdminRoleDelete Permission = "admin_role:delete"
)

// ============================================================================
// CONFIG PERMISSIONS (no create/delete — config rows are seeded)
// ============================================================================
const (
	ConfigRead   Permission = "config:read"
	ConfigUpdate Permission = "config:update"
)

// ============================================================================
// LOG PERMISSIONS (read only — audit logs)
// ============================================================================
const (
	LogRead Permission = "log:read"
)

// ============================================================================
// USER PERMISSIONS (no create — users register themselves)
// ============================================================================
const (
	UserRead       Permission = "user:read"
	UserUpdate     Permission = "user:update"
	UserAssignRole Permission = "user:assign_role"
	UserDelete     Permission = "user:delete"
)

// ============================================================================
// PERMISSION REGISTRY
// ============================================================================

// PermissionInfo contains metadata about a permission
type PermissionInfo struct {
	Permission  Permission
	Resource    string
	Action      string
	Description string
}

// AllPermissions maps resources to their available permissions with descriptions
var AllPermissions = map[string][]PermissionInfo{
	ResourceAdminUser: {
		{AdminUserCreate, ResourceAdminUser, ActionCreate, "Create admin users"},
		{AdminUserRead, ResourceAdminUser, ActionRead, "View admin users"},
		{AdminUserUpdate, ResourceAdminUser, ActionUpdate, "Update admin users"},
		{AdminUserDelete, ResourceAdminUser, ActionDelete, "Delete admin users"},
		{AdminUserChangePassword, ResourceAdminUser, "change_password", "Change admin user password"},
	},
	ResourceAdminRole: {
		{AdminRoleCreate, ResourceAdminRole, ActionCreate, "Create admin roles"},
		{AdminRoleRead, ResourceAdminRole, ActionRead, "View admin roles"},
		{AdminRoleUpdate, ResourceAdminRole, ActionUpdate, "Update admin roles"},
		{AdminRoleDelete, ResourceAdminRole, ActionDelete, "Delete admin roles"},
	},
	ResourceConfig: {
		{ConfigRead, ResourceConfig, ActionRead, "View configurations"},
		{ConfigUpdate, ResourceConfig, ActionUpdate, "Update configurations"},
	},
	ResourceLog: {
		{LogRead, ResourceLog, ActionRead, "View audit logs"},
	},
	ResourceUser: {
		{UserRead, ResourceUser, ActionRead, "View users"},
		{UserUpdate, ResourceUser, ActionUpdate, "Update users"},
		{UserAssignRole, ResourceUser, "assign_role", "Assign roles to user"},
		{UserDelete, ResourceUser, ActionDelete, "Delete users"},
	},
}

// permissionSet contains all valid permissions for quick lookup
var permissionSet map[string]bool

// resourceActions maps resource to its valid actions (for "manage" permission check)
var resourceActions map[string][]string

func init() {
	permissionSet = make(map[string]bool)
	resourceActions = make(map[string][]string)

	for resource, perms := range AllPermissions {
		actions := make([]string, 0)
		for _, p := range perms {
			permissionSet[p.Permission.String()] = true
			if p.Action != ActionManage {
				actions = append(actions, p.Action)
			}
		}
		resourceActions[resource] = actions
	}
}

// IsValidPermission checks if a permission string is valid
func IsValidPermission(perm string) bool {
	return permissionSet[perm]
}

// GetResourceActions returns all valid actions for a resource (excluding
// "manage"). The slice is a copy so callers cannot mutate the registry.
func GetResourceActions(resource string) []string {
	return slices.Clone(resourceActions[resource])
}

// GetAllPermissionsList returns a flat, sorted list of all permission strings.
// Sorting keeps seeding and diffing deterministic across process restarts.
func GetAllPermissionsList() []string {
	result := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		result = append(result, perm)
	}
	sort.Strings(result)
	return result
}

// GetPermissionsForFrontend returns permissions formatted for frontend use
func GetPermissionsForFrontend() map[string][]map[string]string {
	result := make(map[string][]map[string]string)
	for resource, perms := range AllPermissions {
		permList := make([]map[string]string, len(perms))
		for i, p := range perms {
			permList[i] = map[string]string{
				"permission":  p.Permission.String(),
				"action":      p.Action,
				"description": p.Description,
			}
		}
		result[resource] = permList
	}
	return result
}
