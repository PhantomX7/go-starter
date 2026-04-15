package seed

import (
	"errors"

	"github.com/PhantomX7/athleton/internal/models"

	"gorm.io/gorm"
)

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
