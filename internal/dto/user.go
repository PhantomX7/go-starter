package dto

import (
	"time"
)

// UserUpdateRequest defines the structure for updating a user
type UserUpdateRequest struct {
	Role *string `json:"role" form:"role" binding:"omitempty,oneof=user reseller"`
	Name *string `json:"name" form:"name"`
}

// UserAssignAdminRoleRequest defines the structure for assigning admin role
type UserAssignAdminRoleRequest struct {
	AdminRoleID uint `json:"admin_role_id" form:"admin_role_id" binding:"required,exist=admin_roles.id"`
}

// ChangeAdminPasswordRequest defines the structure for root changing an admin's password
type ChangeAdminPasswordRequest struct {
	NewPassword string `json:"new_password" form:"new_password" binding:"required,min=8"`
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
