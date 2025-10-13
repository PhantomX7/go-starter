package repository

import (
	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/pkg/repository"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	repository.IRepository[models.User]
}

// userRepository implements the UserRepository interface
type userRepository struct {
	repository.Repository[models.User]
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		Repository: repository.Repository[models.User]{
			DB: db,
		},
	}
}
