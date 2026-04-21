// Package models defines the application's persistence models.
package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Timestamp contains the standard creation, update, and soft-delete fields.
type Timestamp struct {
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// AccessClaims is the JWT claim set used for signed access tokens.
type AccessClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

// RefreshClaims is the JWT claim set used for signed refresh tokens.
type RefreshClaims struct {
	UserID uint `json:"user_id"`

	jwt.RegisteredClaims
}
