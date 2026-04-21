// Package seed contains database seed helpers for local environments.
package seed

import (
	"errors"

	"github.com/PhantomX7/athleton/internal/models"

	"gorm.io/gorm"
)

// SeedConfigs inserts default configuration records when they do not already exist.
//
//nolint:revive // SeedConfigs is kept for consistency with the seeder entrypoint naming.
func SeedConfigs(db *gorm.DB) error {
	configs := []models.Config{}

	for _, config := range configs {
		if !errors.Is(db.Where("key = ?", config.Key).First(&models.Config{}).Error, gorm.ErrRecordNotFound) {
			continue
		}

		err := db.Create(&config).Error
		if err != nil {
			return err
		}
	}

	return nil
}
