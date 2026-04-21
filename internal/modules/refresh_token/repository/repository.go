// Package repository provides refresh-token persistence primitives.
package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HashRefreshToken returns the deterministic at-rest form of a refresh-token
// value. Refresh tokens are stored as SHA-256 hex digests (not plaintext) so
// that a database read leak does not yield active sessions: the wire value
// only ever exists in the legitimate client and the response that minted it.
// SHA-256 is appropriate (no salt needed) because the input is a high-entropy
// random string, not a user-chosen secret.
func HashRefreshToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// RefreshTokenRepository defines the interface for refresh token repository operations.
type RefreshTokenRepository interface {
	repository.Repository[models.RefreshToken]
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	FindActiveByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error)
	GetValidCountByUserID(ctx context.Context, userID uint) (int64, error)
	DeleteInvalidToken(ctx context.Context) error
	RevokeAllByUserID(ctx context.Context, userID uint) error
	RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error
	RevokeByToken(ctx context.Context, token string) error
}

type refreshTokenRepository struct {
	repository.BaseRepository[models.RefreshToken]
}

// NewRefreshTokenRepository constructs a RefreshTokenRepository.
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		BaseRepository: repository.NewBaseRepository[models.RefreshToken](db),
	}
}

// Create overrides BaseRepository.Create to hash the plaintext token before
// persisting. Callers continue to pass the wire value; the stored row only
// ever contains the hash. The entity is mutated in place so a re-read would
// see the hash, but there are no current callers that re-read Token after
// Create.
func (r *refreshTokenRepository) Create(ctx context.Context, entity *models.RefreshToken) error {
	if entity != nil && entity.Token != "" {
		entity.Token = HashRefreshToken(entity.Token)
	}
	return r.BaseRepository.Create(ctx, entity)
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

// FindByToken returns the active refresh token matching the given plaintext
// value. The plaintext is hashed before the column lookup since that is what
// is stored at rest.
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	hashed := HashRefreshToken(token)
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(hashed))
	for _, p := range activeTokenPredicates(time.Now()) {
		q = q.Where(p)
	}

	rt, err := q.First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("invalid refresh token")
		}
		// NOTE: do not include the token (or its hash) in the error message —
		// hashes are not secret but they are still session identifiers and
		// should not be sprayed into logs.
		return nil, cerrors.NewInternalServerError("failed to find refresh token record", err)
	}
	return &rt, nil
}

// FindActiveByID returns the active refresh token whose primary key equals id.
// Used to bind an access JWT (jti claim) to a specific refresh-token session,
// so that revoking that session also kills its access tokens.
func (r *refreshTokenRepository) FindActiveByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.ID.Eq(id))
	for _, p := range activeTokenPredicates(time.Now()) {
		q = q.Where(p)
	}

	rt, err := q.First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("invalid refresh token session")
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find refresh token by id %v", id), err)
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
// one whose plaintext value equals exceptToken. The exception is matched on
// the stored hash; an empty exceptToken naturally matches no row and revokes
// all of the user's active tokens.
func (r *refreshTokenRepository) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error {
	now := time.Now()
	exceptHash := HashRefreshToken(exceptToken)
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.UserID.Eq(userID)).
		Where(generated.RefreshToken.Token.Neq(exceptHash))
	for _, p := range activeTokenPredicates(now) {
		q = q.Where(p)
	}

	if _, err := q.Set(generated.RefreshToken.RevokedAt.Set(now)).Update(ctx); err != nil {
		return cerrors.NewInternalServerError(fmt.Sprintf("failed to revoke refresh tokens for user id %v", userID), err)
	}
	return nil
}

// RevokeByToken revokes one specific refresh token by its plaintext value.
// The plaintext is hashed before matching since that is what is stored.
// NOTE: this does not return ErrNotFound when the token is already revoked or
// missing — current callers either call FindByToken first or treat 0-row
// updates as success. If you add a new caller that needs to detect reuse
// (e.g. defensive token-reuse mitigation in rotation), capture rows-affected
// from gorm and surface it.
func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	now := time.Now()
	hashed := HashRefreshToken(token)
	_, err := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(hashed)).
		Where(generated.RefreshToken.RevokedAt.IsNull()).
		Set(generated.RefreshToken.RevokedAt.Set(now)).
		Update(ctx)
	if err != nil {
		return cerrors.NewInternalServerError("failed to revoke by token", err)
	}
	return nil
}
