package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RefreshTokenRepository defines the interface for refresh token repository operations.
type RefreshTokenRepository interface {
	repository.Repository[models.RefreshToken]
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	GetValidCountByUserID(ctx context.Context, userID uint) (int64, error)
	DeleteInvalidToken(ctx context.Context) error
	RevokeAllByUserID(ctx context.Context, userID uint) error
	RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error
	RevokeByToken(ctx context.Context, token string) error
}

type refreshTokenRepository struct {
	repository.BaseRepository[models.RefreshToken]
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		BaseRepository: repository.NewBaseRepository[models.RefreshToken](db),
	}
}

// activeTokenPredicates returns the two column conditions that make a token
// "active": not revoked and not expired. Centralized so adding/removing a
// condition changes every caller in one place.
func activeTokenPredicates(now time.Time) []clause.Expression {
	return []clause.Expression{
		generated.RefreshToken.RevokedAt.IsNull(),
		generated.RefreshToken.ExpiresAt.Gt(now),
	}
}

// FindByToken returns the active refresh token matching the given value.
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(token))
	for _, p := range activeTokenPredicates(time.Now()) {
		q = q.Where(p)
	}

	rt, err := q.First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("invalid refresh token")
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find refresh token record by token %v", token), err)
	}
	return &rt, nil
}

// GetValidCountByUserID counts active tokens for a user.
func (r *refreshTokenRepository) GetValidCountByUserID(ctx context.Context, userID uint) (int64, error) {
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.UserID.Eq(userID))
	for _, p := range activeTokenPredicates(time.Now()) {
		q = q.Where(p)
	}

	count, err := q.Count(ctx, "*")
	if err != nil {
		return 0, cerrors.NewInternalServerError("failed to count valid refresh tokens", err)
	}
	return count, nil
}

// DeleteInvalidToken hard-deletes tokens that have expired or been revoked.
func (r *refreshTokenRepository) DeleteInvalidToken(ctx context.Context) error {
	now := time.Now()
	_, err := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.ExpiresAt.Lt(now)).
		Or(generated.RefreshToken.RevokedAt.IsNotNull()).
		Delete(ctx)
	if err != nil {
		return cerrors.NewInternalServerError("failed to delete invalid refresh token", err)
	}
	return nil
}

// RevokeAllByUserID stamps revoked_at on every active token for the user.
func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	now := time.Now()
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.UserID.Eq(userID))
	for _, p := range activeTokenPredicates(now) {
		q = q.Where(p)
	}

	if _, err := q.Set(generated.RefreshToken.RevokedAt.Set(now)).Update(ctx); err != nil {
		return cerrors.NewInternalServerError(fmt.Sprintf("failed to revoke all refresh tokens for user id %v", userID), err)
	}
	return nil
}

// RevokeAllByUserIDExcept revokes every active token for the user except the
// one whose value equals exceptToken.
func (r *refreshTokenRepository) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error {
	now := time.Now()
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.UserID.Eq(userID)).
		Where(generated.RefreshToken.Token.Neq(exceptToken))
	for _, p := range activeTokenPredicates(now) {
		q = q.Where(p)
	}

	if _, err := q.Set(generated.RefreshToken.RevokedAt.Set(now)).Update(ctx); err != nil {
		return cerrors.NewInternalServerError(fmt.Sprintf("failed to revoke refresh tokens for user id %v", userID), err)
	}
	return nil
}

// RevokeByToken revokes one specific refresh token by value.
// NOTE: unlike the original, this does not return ErrNotFound when the token
// is already revoked or missing — both current callers either call FindByToken
// first or ignore the error. If you add a new caller that needs the signal,
// call FindByToken first.
func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	now := time.Now()
	_, err := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(token)).
		Where(generated.RefreshToken.RevokedAt.IsNull()).
		Set(generated.RefreshToken.RevokedAt.Set(now)).
		Update(ctx)
	if err != nil {
		return cerrors.NewInternalServerError("failed to revoke by token", err)
	}
	return nil
}
