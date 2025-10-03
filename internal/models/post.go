package models

import (
	"github.com/PhantomX7/go-starter/internal/modules/post/dto"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model

	Title       string `gorm:"type:varchar(255);not null" json:"title"`
	Content     string `gorm:"type:text;not null" json:"content"`
	Description string `gorm:"type:text" json:"description"`
	ImageUrl    string `gorm:"type:varchar(255)" json:"image_url"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
}

func (p Post) ToResponse() any {
	return dto.PostResponse{
		ID:          p.ID,
		Title:       p.Title,
		Content:     p.Content,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
