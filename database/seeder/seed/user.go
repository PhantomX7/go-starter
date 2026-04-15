package seed

import (
	"errors"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB) error {
	users := []models.User{
		{
			Username: "root",
			Email:    "[EMAIL_ADDRESS]",
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
	password, err := bcrypt.GenerateFromPassword([]byte(config.Get().Admin.DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	for _, user := range users {
		user.Password = string(password)
		if !errors.Is(db.First(&models.User{}, models.User{
			Username: user.Username,
		}).Error, gorm.ErrRecordNotFound) {
			continue
		}

		err := db.Create(&user).Error
		if err != nil {
			return err
		}
	}

	return nil
}
