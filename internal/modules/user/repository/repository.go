package repository

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/pkg/repository"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	repository.IRepository[models.User]
	FindByToken(ctx context.Context, user *models.User, token string) error
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

// FindByToken finds a user by their token
func (r *userRepository) FindByToken(ctx context.Context, user *models.User, token string) error {
	return r.DB.WithContext(ctx).Where("token = ?", token).First(user).Error
}
