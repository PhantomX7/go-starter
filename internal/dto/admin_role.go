// Package dto contains API request and response data-transfer objects.
package dto

import "time"

// CreateAdminRoleRequest is the payload for creating an admin role.
type CreateAdminRoleRequest struct {
	Name        string   `json:"name" form:"name" binding:"required,min=2,max=100,unique=admin_roles.name"`
	Description string   `json:"description" form:"description" binding:"max=255"`
	Permissions []string `json:"permissions" form:"permissions[]" binding:"required,min=1,dive,required"`
}

// UpdateAdminRoleRequest is the payload for updating an admin role.
type UpdateAdminRoleRequest struct {
	Name        *string  `json:"name" form:"name" binding:"omitempty,min=2,max=100,unique=admin_roles.name"`
	Description *string  `json:"description" form:"description" binding:"omitempty,max=255"`
	Permissions []string `json:"permissions" form:"permissions[]" binding:"omitempty,dive,required"`
}

// AdminRoleResponse is the API response shape for an admin role.
type AdminRoleResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
