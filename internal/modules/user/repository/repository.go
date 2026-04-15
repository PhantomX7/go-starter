// internal/modules/user/repository/user_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	repository.IRepository[models.User]
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error) // Added this
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

// FindByUsername finds a user by username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	start := time.Now()

	var user models.User

	err := r.GetDB(ctx).WithContext(ctx).Where("username = ?", username).Take(&user).Error
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("user with username %s not found", username))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find user by username %s", username), err)
	}

	r.LogSlowQuery(ctx, "FindByUsername", duration, 500*time.Millisecond)

	return &user, nil
}

// FindByEmail finds a user by email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	start := time.Now()

	var user models.User

	err := r.GetDB(ctx).WithContext(ctx).Where("email = ?", email).Take(&user).Error
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("user with email %s not found", email))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find user by email %s", email), err)
	}

	// Log if query was slow
	r.LogSlowQuery(ctx, "FindByEmail", duration, 500*time.Millisecond)

	return &user, nil
}
