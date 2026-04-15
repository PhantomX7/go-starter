package models

import (
	"github.com/PhantomX7/athleton/internal/dto"

	"gorm.io/gorm"
)

type ConfigKey string

const ()

func (c ConfigKey) ToString() string {
	return string(c)
}

// Config represents the config entity
type Config struct {
	gorm.Model

	Key   string `json:"key" gorm:"type:varchar(255);not null" `
	Value string `json:"value" gorm:"type:text;not null" `

	// Polymorphic Logs
	Logs []Log `json:"-" gorm:"polymorphic:Entity;polymorphicValue:configs"`
}

// ToResponse converts the Config model to a response DTO
func (m *Config) ToResponse() *dto.ConfigResponse {
	return &dto.ConfigResponse{
		ID:    m.ID,
		Key:   m.Key,
		Value: m.Value,
	}
}
