// internal/modules/user/repository/user_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations.
type UserRepository interface {
	repository.IRepository[models.User]
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type userRepository struct {
	repository.Repository[models.User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		Repository: repository.Repository[models.User]{DB: db},
	}
}

// FindByUsername looks up a user by exact username match.
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	start := time.Now()

	user, err := gorm.G[models.User](r.GetDB(ctx)).
		Where(generated.User.Username.Eq(username)).
		First(ctx)

	r.LogSlowQuery(ctx, "FindByUsername", time.Since(start), 500*time.Millisecond)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("user with username %s not found", username))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find user by username %s", username), err)
	}

	return &user, nil
}

// FindByEmail looks up a user by email. Callers may pass arbitrary casing /
// padding; we normalize to match auth.Register's write path so the lookup
// cannot silently miss a row that was stored in lowercase.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	start := time.Now()
	normalized := strings.ToLower(strings.TrimSpace(email))

	user, err := gorm.G[models.User](r.GetDB(ctx)).
		Where(generated.User.Email.Eq(normalized)).
		First(ctx)

	r.LogSlowQuery(ctx, "FindByEmail", time.Since(start), 500*time.Millisecond)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("user with email %s not found", email))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find user by email %s", email), err)
	}

	return &user, nil
}
