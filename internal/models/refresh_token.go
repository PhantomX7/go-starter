package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"primary_key;not null"`
	UserID    uint       `json:"user_id" gorm:"type:bigint;not null"`
	Token     string     `json:"token" gorm:"not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" gorm:"null;default:null"`

	User User `json:"user" gorm:"foreignKey:UserID"`
}
