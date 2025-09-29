package models

import (
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model

	Title       string `gorm:"type:varchar(255);not null" json:"title"`
	Content     string `gorm:"type:text;not null" json:"content"`
	Description string `gorm:"type:text" json:"description"`
	ImageUrl    string `gorm:"type:varchar(255)" json:"image_url"`
}
