package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Timestamp struct {
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type AccessClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID uint `json:"user_id"`

	jwt.RegisteredClaims
}
