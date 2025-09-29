package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Name       string `gorm:"type:varchar(100);not null" json:"name"`
	Email      string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	TelpNumber string `gorm:"type:varchar(20);index" json:"telp_number"`
	Password   string `gorm:"type:varchar(255);not null" json:"password"`
	Role       string `gorm:"type:varchar(50);not null;default:'user'" json:"role"`
	ImageUrl   string `gorm:"type:varchar(255)" json:"image_url"`
	IsVerified bool   `gorm:"default:false" json:"is_verified"`
}
