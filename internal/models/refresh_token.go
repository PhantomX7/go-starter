// Package models defines the application's persistence models.
package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken stores a refresh-token session and its lifecycle timestamps.
type RefreshToken struct {
	ID uuid.UUID `json:"id" gorm:"primary_key;not null"`
	// refresh_tokens never soft-deletes, so a plain unique index is correct here.
	UserID    uint       `json:"user_id" gorm:"type:bigint;not null;index"`
	Token     string     `json:"token" gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" gorm:"null;default:null"`

	User User `json:"user" gorm:"foreignKey:UserID"`
}
