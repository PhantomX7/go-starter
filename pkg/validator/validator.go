package validator

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)


type CustomValidator interface {
	Unique() validator.Func
	Exist() validator.Func
}

type customValidator struct {
	validator *validator.Validate
	db        *gorm.DB
}

func New(db *gorm.DB) CustomValidator {
	return &customValidator{
		validator: validator.New(),
		db:        db,
	}
}
