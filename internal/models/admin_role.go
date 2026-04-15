// internal/models/admin_role.go
package models

import (
	"github.com/PhantomX7/athleton/internal/dto"
)

type AdminRole struct {
	ID          uint     `json:"id" gorm:"primaryKey"`
	Name        string   `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Description string   `json:"description" gorm:"type:varchar(255);null"`
	IsActive    bool     `json:"is_active" gorm:"not null;default:true"`
	Permissions []string `json:"permissions" gorm:"-"`
	Timestamp

	// Polymorphic Logs
	Logs []Log `json:"-" gorm:"polymorphic:Entity;polymorphicValue:admin_roles"`
}

func (a *AdminRole) ToResponse() *dto.AdminRoleResponse {
	return &dto.AdminRoleResponse{
		ID:          a.ID,
		Name:        a.Name,
		Description: a.Description,
		IsActive:    a.IsActive,
		Permissions: a.Permissions,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
