// internal/models/user.go
package models

import (
	"github.com/PhantomX7/athleton/internal/dto"
)

type UserRole string

const (
	UserRoleUser     UserRole = "user"
	UserRoleAdmin    UserRole = "admin"
	UserRoleRoot     UserRole = "root"
	UserRoleReseller UserRole = "reseller"
)

// Note: Removed UserRoleWriter - writers are now admins with specific permissions

func (u UserRole) ToString() string {
	return string(u)
}

// IsAdminType returns true if the role is admin or root
func (u UserRole) IsAdminType() bool {
	return u == UserRoleAdmin || u == UserRoleRoot
}

type User struct {
	ID           uint     `json:"id" gorm:"primaryKey"`
	Username     string   `json:"username" gorm:"type:varchar(255);not null"`
	Name         string   `json:"name" gorm:"type:varchar(255);null"`
	BusinessName string   `json:"business_name" gorm:"type:varchar(255);null"`
	Email        string   `json:"email" gorm:"type:varchar(255);not null"`
	Phone        string   `json:"phone" gorm:"type:varchar(255);not null"`
	IsActive     bool     `json:"is_active" gorm:"not null;default:true"`
	Role         UserRole `json:"role" gorm:"type:user_role;not null"`
	AdminRoleID  *uint    `json:"admin_role_id" gorm:"type:bigint;null;index"`
	Password     string   `json:"-" gorm:"type:varchar(255);not null"`
	Timestamp

	// Relationships
	AdminRole *AdminRole `json:"admin_role,omitempty" gorm:"foreignKey:AdminRoleID"`

	// Polymorphic Logs
	Logs []Log `json:"-" gorm:"polymorphic:Entity;polymorphicValue:users"`
}

func (u User) ToResponse() *dto.UserResponse {
	response := dto.UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		BusinessName: u.BusinessName,
		Username:     u.Username,
		Email:        u.Email,
		Phone:        u.Phone,
		IsActive:     u.IsActive,
		Role:         u.Role.ToString(),
		AdminRoleID:  u.AdminRoleID,
		CreatedAt:    u.CreatedAt,
	}

	if u.AdminRole != nil {
		response.AdminRole = u.AdminRole.ToResponse()
	}

	return &response
}
