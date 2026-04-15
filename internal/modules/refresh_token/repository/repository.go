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

// RefreshTokenRepository defines the interface for refresh token repository operations
type RefreshTokenRepository interface {
	repository.IRepository[models.RefreshToken]
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	GetValidCountByUserID(ctx context.Context, userID uint) (int64, error)
	DeleteInvalidToken(ctx context.Context) error
	RevokeAllByUserID(ctx context.Context, userID uint) error
	RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error
	RevokeByToken(ctx context.Context, token string) error
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

// FindByToken finds a refresh token without locking (for read operations)
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken

	if err := r.GetDB(ctx).WithContext(ctx).
		Where("token = ? AND expires_at > ? AND revoked_at IS NULL", token, time.Now()).
		First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMessage := "invalid refresh token"
			return nil, cerrors.NewNotFoundError(errMessage)
		}
		errMessage := fmt.Sprintf("failed to find refresh token record by token %v", token)
		return nil, cerrors.NewInternalServerError(errMessage, err)
	}
	return &refreshToken, nil
}

// GetValidCountByUserID counts valid (non-revoked, non-expired) tokens for a user
func (r *refreshTokenRepository) GetValidCountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.GetDB(ctx).WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Count(&count).Error

	if err != nil {
		return 0, cerrors.NewInternalServerError("failed to count valid refresh tokens", err)
	}

	return count, nil
}

// DeleteInvalidToken deletes expired or revoked tokens
func (r *refreshTokenRepository) DeleteInvalidToken(ctx context.Context) error {
	err := r.GetDB(ctx).WithContext(ctx).
		Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now()).
		Delete(&models.RefreshToken{}).Error

	if err != nil {
		errMessage := "failed to delete invalid refresh token"
		return cerrors.NewInternalServerError(errMessage, err)
	}

	return nil
}

// RevokeAllByUserIDExcept revokes all active refresh tokens for a specific user except the specified token
func (r *refreshTokenRepository) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error {
	now := time.Now()

	result := r.GetDB(ctx).WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND token != ? AND revoked_at IS NULL AND expires_at > ?", userID, exceptToken, now).
		Update("revoked_at", now)

	if result.Error != nil {
		errMessage := fmt.Sprintf("failed to revoke refresh tokens for user id %v", userID)
		return cerrors.NewInternalServerError(errMessage, result.Error)
	}

	return nil
}

// RevokeAllByUserID revokes all active refresh tokens for a specific user
func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	now := time.Now()

	result := r.GetDB(ctx).WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Update("revoked_at", now)

	if result.Error != nil {
		errMessage := fmt.Sprintf("failed to revoke all refresh tokens for user id %v", userID)
		return cerrors.NewInternalServerError(errMessage, result.Error)
	}

	return nil
}

// RevokeByToken revokes a specific refresh token
func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	now := time.Now()

	result := r.GetDB(ctx).WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("token = ? AND revoked_at IS NULL", token).
		Update("revoked_at", now)

	if result.Error != nil {
		errMessage := "failed to revoke by token"
		return cerrors.NewInternalServerError(errMessage, result.Error)
	}

	if result.RowsAffected == 0 {
		return cerrors.NewNotFoundError("refresh token not found or already revoked")
	}

	return nil
}
