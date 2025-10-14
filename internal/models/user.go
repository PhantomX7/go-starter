package models

import (
	"github.com/PhantomX7/go-starter/internal/modules/user/dto"
)

type UserRole string

const (
	UserRoleUser     UserRole = "user"
	UserRoleAdmin    UserRole = "admin"
	UserRoleReseller UserRole = "reseller"
)

func (u UserRole) ToString() string {
	return string(u)
}

type User struct {
	ID       uint     `json:"id" gorm:"primaryKey"`
	Username string   `json:"username" gorm:"type:varchar(255);not null"`
	Email    string   `json:"email" gorm:"type:varchar(255);not null"`
	Phone    string   `json:"phone" gorm:"type:varchar(255);not null"`
	IsActive bool     `json:"is_active" gorm:"not null;default:true" `
	Role     UserRole `json:"role" gorm:"type:user_role;not null"`
	Password string   `json:"-" gorm:"type:varchar(255);not null"`
	Timestamp
}

func (u User) ToResponse() dto.UserResponse {
	return dto.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     u.Phone,
		IsActive:  u.IsActive,
		Role:      u.Role.ToString(),
		CreatedAt: u.CreatedAt,
	}
}
