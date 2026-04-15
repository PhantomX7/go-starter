// internal/dto/admin_role.go
package dto

import "time"

// Request DTOs
type CreateAdminRoleRequest struct {
	Name        string   `json:"name" form:"name" binding:"required,min=2,max=100,unique=admin_roles.name"`
	Description string   `json:"description" form:"description" binding:"max=255"`
	Permissions []string `json:"permissions" form:"permissions[]" binding:"required,min=1,dive,required"`
}

type UpdateAdminRoleRequest struct {
	Name        *string  `json:"name" form:"name" binding:"omitempty,min=2,max=100,unique=admin_roles.name"`
	Description *string  `json:"description" form:"description" binding:"omitempty,max=255"`
	Permissions []string `json:"permissions" form:"permissions[]" binding:"omitempty,dive,required"`
}

// Response DTOs
type AdminRoleResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
