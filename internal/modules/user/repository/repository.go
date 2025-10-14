package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/PhantomX7/go-starter/internal/models"
	c_errors "github.com/PhantomX7/go-starter/pkg/errors"
	"github.com/PhantomX7/go-starter/pkg/repository"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	repository.IRepository[models.User]
	FindByUsername(ctx context.Context, username string) (*models.User, error)
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

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var entity models.User

	db := r.GetDB(ctx)
	err := db.Where("username = ?", username).Take(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMessage := fmt.Sprintf("user with username %v not found", username)
			return nil, c_errors.NewNotFoundError(errMessage)
		}
		errMessage := fmt.Sprintf("failed to find user with username %v", username)
		return nil, c_errors.NewInternalServerError(errMessage, err)
	}
	return &entity, nil
}