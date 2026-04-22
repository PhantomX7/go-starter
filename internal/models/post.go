package models

import (
	"github.com/PhantomX7/athleton/internal/dto"

	"gorm.io/gorm"
)

// Post represents the post entity
type Post struct {
	gorm.Model

	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
}

// ToResponse converts the Post model to a response DTO
func (m Post) ToResponse() any {
	return dto.PostResponse{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
