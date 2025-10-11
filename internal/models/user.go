package models

import (
	"github.com/PhantomX7/go-starter/internal/modules/user/dto"
)

// User represents the user entity
type User struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Username        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
	Timestamp
}

// ToResponse converts the User model to a response DTO
func (m User) ToResponse() any {
	return dto.UserResponse{
		ID:          m.ID,
		// Username:        m.Username,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
