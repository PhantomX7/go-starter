// Package validator provides custom go-playground validators used by the application.
package validator

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// CustomValidator exposes the custom validator functions registered by the app.
type CustomValidator interface {
	Unique() validator.Func
	Exist() validator.Func
	FileSize() validator.Func
	FileExtension() validator.Func
}

type customValidator struct {
	validator *validator.Validate
	db        *gorm.DB
}

// New creates a CustomValidator backed by the provided database connection.
func New(db *gorm.DB) CustomValidator {
	return &customValidator{
		validator: validator.New(),
		db:        db,
	}
}
