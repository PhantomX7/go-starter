package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/go-starter/internal/models"
	c_errors "github.com/PhantomX7/go-starter/pkg/errors"
	"github.com/PhantomX7/go-starter/pkg/repository"

	"gorm.io/gorm"
)

// RefreshTokenRepository defines the interface for refresh token repository operations
type RefreshTokenRepository interface {
	repository.IRepository[models.RefreshToken]
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	GetValidCountByUserID(ctx context.Context, userID uint) (int64, error)
}

// refreshTokenRepository implements the RefreshTokenRepository interface
type refreshTokenRepository struct {
	repository.Repository[models.RefreshToken]
}

// NewRefreshTokenRepository creates a new instance of RefreshTokenRepository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		Repository: repository.Repository[models.RefreshToken]{
			DB: db,
		},
	}
}

func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken

	if err := r.GetDB(ctx).WithContext(ctx).
		Where("token = ? AND expires_at > ? AND revoked_at IS NULL", token, time.Now()).
		First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMessage := "invalid refresh token"
			return nil, c_errors.NewNotFoundError(errMessage)
		}
		errMessage := fmt.Sprintf("failed to find refresh token record by id %v", token)
		return nil, c_errors.NewInternalServerError(errMessage, err)
	}
	return &refreshToken, nil
}

func (r *refreshTokenRepository) GetValidCountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.GetDB(ctx).WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Count(&count).Error

	if err != nil {
		return 0, c_errors.NewInternalServerError("failed to count valid refresh tokens", err)
	}

	return count, nil
}
