// Package seed contains database seed helpers for local environments.
package seed

import (
	"errors"
	"fmt"

	"github.com/PhantomX7/athleton/internal/models"

	"gorm.io/gorm"
)

// SeedConfigs inserts default configuration records when they do not already exist.
//
//nolint:revive // SeedConfigs is kept for consistency with the seeder entrypoint naming.
func SeedConfigs(db *gorm.DB) error {
	configs := []models.Config{}

	for _, config := range configs {
		err := db.Where("key = ?", config.Key).First(&models.Config{}).Error
		switch {
		case err == nil:
			// Config already exists; nothing to seed.
			continue
		case errors.Is(err, gorm.ErrRecordNotFound):
			// Config is missing; proceed with creation.
		default:
			return fmt.Errorf("failed to check existing config %q: %w", config.Key, err)
		}

		if err := db.Create(&config).Error; err != nil {
			return err
		}
	}

	return nil
}
