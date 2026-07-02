package seed

import (
	"errors"
	"fmt"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BcryptCost is the bcrypt work factor used when hashing seeded passwords.
const BcryptCost = 12

// SeedUsers inserts the default root and admin users when they do not already exist.
//
//nolint:revive // SeedUsers is kept for consistency with the seeder entrypoint naming.
func SeedUsers(db *gorm.DB, cfg *config.Config) error {
	users := []models.User{
		{
			Username: "root",
			Email:    cfg.Admin.Email,
			Phone:    "+6281123456789",
			IsActive: true,
			Role:     models.UserRoleRoot,
		},
		{
			Username: "admin",
			Email:    "admin@athleton.com",
			Phone:    "+6281123456789",
			IsActive: true,
			Role:     models.UserRoleAdmin,
		},
	}

	var password []byte
	password, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.DefaultPassword), BcryptCost)
	if err != nil {
		return err
	}

	for _, user := range users {
		user.Password = string(password)

		err := db.First(&models.User{}, models.User{
			Username: user.Username,
		}).Error
		switch {
		case err == nil:
			// User already exists; nothing to seed.
			continue
		case errors.Is(err, gorm.ErrRecordNotFound):
			// User is missing; proceed with creation.
		default:
			return fmt.Errorf("failed to check existing user %q: %w", user.Username, err)
		}

		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	return nil
}
