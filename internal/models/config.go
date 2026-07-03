// Package models defines the application's persistence models.
package models

import (
	"github.com/PhantomX7/athleton/internal/dto"

	"gorm.io/gorm"
)

// ConfigKey is the strongly typed identifier for a configuration key.
type ConfigKey string

// ToString converts a ConfigKey to its raw string representation.
func (c ConfigKey) ToString() string {
	return string(c)
}

// Config represents the config entity
type Config struct {
	gorm.Model

	// Key uses a partial unique index (WHERE deleted_at IS NULL) so
	// soft-deleted rows do not block reuse of the key.
	Key   string `json:"key" gorm:"type:varchar(255);not null;uniqueIndex:idx_configs_key,where:deleted_at IS NULL"`
	Value string `json:"value" gorm:"type:text;not null"`
	// IsPublic gates the unauthenticated /public/config surface: only rows
	// explicitly marked public are served there. Default false — a config
	// table naturally accumulates secrets, so visibility is opt-in.
	IsPublic bool `json:"is_public" gorm:"not null;default:false"`

	// Polymorphic Logs. polymorphicValue must equal LogEntityTypeConfig
	// (the discriminator the audit writers store).
	Logs []Log `json:"-" gorm:"polymorphic:Entity;polymorphicValue:config"`
}

// ToResponse converts the Config model to a response DTO
func (m *Config) ToResponse() *dto.ConfigResponse {
	return &dto.ConfigResponse{
		ID:       m.ID,
		Key:      m.Key,
		Value:    m.Value,
		IsPublic: m.IsPublic,
	}
}
