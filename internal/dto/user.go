package dto

import (
	"time"
)

// UserUpdateRequest defines the structure for updating a user.
//
// Role deliberately accepts only "user": promotion to "admin" must go through
// the dedicated AssignAdminRole endpoint so Role and AdminRoleID always change
// together (Casbin ignores AdminRoleID unless Role == "admin"), and "root" is
// never assignable through the API. The only role transition Update supports
// is demoting an admin back to a plain user, which also clears AdminRoleID.
// ("reseller" was removed — it is not a role this application defines.)
type UserUpdateRequest struct {
	Role *string `json:"role" form:"role" binding:"omitempty,oneof=user" enums:"user"`
	Name *string `json:"name" form:"name"`
}

// AdminUserCreateRequest is the payload for an admin creating a new admin
// account directly (as opposed to self-registration + promotion).
//
// There is deliberately no role field: the created account is always "admin"
// — root can never be created through the API. The initial password is chosen
// by the creator, so the service leaves PasswordChangedAt nil and the
// must-change-default-password gate forces a rotation on first login.
type AdminUserCreateRequest struct {
	Username    string `json:"username" form:"username" binding:"required,min=3,max=255,unique=users.username"`
	Name        string `json:"name" form:"name" binding:"required,max=255"`
	Email       string `json:"email" form:"email" binding:"required,email,max=255,unique=users.email"`
	Phone       string `json:"phone" form:"phone" binding:"required,max=255"`
	Password    string `json:"password" form:"password" binding:"required,min=8,max=72" minLength:"8" maxLength:"72"`
	AdminRoleID uint   `json:"admin_role_id" form:"admin_role_id" binding:"required,exist=admin_roles.id"`
}

// UserAssignAdminRoleRequest defines the structure for assigning admin role
type UserAssignAdminRoleRequest struct {
	AdminRoleID uint `json:"admin_role_id" form:"admin_role_id" binding:"required,exist=admin_roles.id"`
}

// ChangeAdminPasswordRequest defines the structure for root changing an admin's password.
// max=72: bcrypt rejects passwords longer than 72 bytes, so validate up front
// instead of surfacing a 500 from the hasher.
type ChangeAdminPasswordRequest struct {
	NewPassword string `json:"new_password" form:"new_password" binding:"required,min=8,max=72" minLength:"8" maxLength:"72"`
}

// UserResponse defines the structure for user response
type UserResponse struct {
	ID           uint               `json:"id"`
	Username     string             `json:"username"`
	Name         string             `json:"name"`
	BusinessName string             `json:"business_name"`
	Email        string             `json:"email"`
	Phone        string             `json:"phone"`
	IsActive     bool               `json:"is_active"`
	AdminRoleID  *uint              `json:"admin_role_id"`
	Role         string             `json:"role"`
	CreatedAt    time.Time          `json:"created_at"`
	AdminRole    *AdminRoleResponse `json:"admin_role,omitempty"`
}
