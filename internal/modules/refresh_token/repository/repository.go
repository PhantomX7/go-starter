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
	RevokeByTokenIfActive(ctx context.Context, token string) (bool, error)
	RevokeOldestActiveByUserID(ctx context.Context, userID uint, n int) error
	UpdateTokenHashIfActive(ctx context.Context, oldToken, newToken string) (bool, error)
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
// The plaintext is hashed before matching since that is what is stored. A
// 0-row update (token already revoked or missing) is treated as success — this
// is the idempotent variant used by logout. Callers that must DETECT reuse
// (e.g. rotation) should use RevokeByTokenIfActive instead.
func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	_, err := r.RevokeByTokenIfActive(ctx, token)
	return err
}

// RevokeByTokenIfActive atomically revokes the active token matching the
// plaintext value and reports whether a row was actually revoked. The
// not-revoked + atomic update means two concurrent callers racing on the same
// token cannot both win: exactly one sees revoked == true. A false return
// signals the token was already revoked or never existed — the reuse signal a
// rotation caller needs to refuse and tear down the session family.
func (r *refreshTokenRepository) RevokeByTokenIfActive(ctx context.Context, token string) (bool, error) {
	now := time.Now()
	hashed := HashRefreshToken(token)
	rows, err := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(hashed)).
		Where(generated.RefreshToken.RevokedAt.IsNull()).
		Set(generated.RefreshToken.RevokedAt.Set(now)).
		Update(ctx)
	if err != nil {
		return false, cerrors.NewInternalServerError("failed to revoke by token", err)
	}
	return rows > 0, nil
}

// RevokeOldestActiveByUserID revokes the user's n oldest active tokens
// (ordered by creation time). Used to enforce the per-user session cap at
// login: revoking the oldest sessions makes room for the new one. It selects
// the victim IDs first and then revokes by ID rather than issuing a single
// UPDATE ... ORDER BY ... LIMIT, which is not portable across the supported
// databases (MySQL rejects a same-table subquery in UPDATE; SQLite only
// accepts UPDATE ... LIMIT with a non-default compile flag). The tiny window
// between the two statements is harmless: the revoke re-checks the active
// predicates, so at worst a concurrently revoked row is skipped.
func (r *refreshTokenRepository) RevokeOldestActiveByUserID(ctx context.Context, userID uint, n int) error {
	if n <= 0 {
		return nil
	}

	now := time.Now()
	q := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.UserID.Eq(userID))
	for _, p := range activeTokenPredicates(now) {
		q = q.Where(p)
	}

	oldest, err := q.Order(generated.RefreshToken.CreatedAt.Asc()).Limit(n).Find(ctx)
	if err != nil {
		return cerrors.NewInternalServerError(fmt.Sprintf("failed to find oldest refresh tokens for user id %v", userID), err)
	}
	if len(oldest) == 0 {
		return nil
	}

	ids := make([]any, 0, len(oldest))
	for _, rt := range oldest {
		ids = append(ids, rt.ID)
	}

	uq := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(clause.IN{Column: generated.RefreshToken.ID.Column(), Values: ids})
	for _, p := range activeTokenPredicates(now) {
		uq = uq.Where(p)
	}

	if _, err := uq.Set(generated.RefreshToken.RevokedAt.Set(now)).Update(ctx); err != nil {
		return cerrors.NewInternalServerError(fmt.Sprintf("failed to revoke oldest refresh tokens for user id %v", userID), err)
	}
	return nil
}

// UpdateTokenHashIfActive atomically replaces the stored hash of the
// not-yet-revoked token matching oldToken with newToken's hash, reporting
// whether a row was actually rewritten. This is the rotate-in-place
// primitive: the row (and therefore its ID, used as the access-token jti, and
// its expires_at, the session's absolute lifetime) is preserved — only the
// presentable secret changes.
//
// The not-revoked + atomic single-statement update gives the same race
// guarantee as RevokeByTokenIfActive: two concurrent rotations of the same
// token cannot both win, because the loser's WHERE token = old-hash no longer
// matches after the winner rewrites it. A false return is therefore the reuse
// signal — the presented token was already rotated away or revoked — and the
// caller must refuse and tear down the session family.
func (r *refreshTokenRepository) UpdateTokenHashIfActive(ctx context.Context, oldToken, newToken string) (bool, error) {
	oldHash := HashRefreshToken(oldToken)
	newHash := HashRefreshToken(newToken)
	rows, err := gorm.G[models.RefreshToken](r.GetDB(ctx)).
		Where(generated.RefreshToken.Token.Eq(oldHash)).
		Where(generated.RefreshToken.RevokedAt.IsNull()).
		Set(generated.RefreshToken.Token.Set(newHash)).
		Update(ctx)
	if err != nil {
		return false, cerrors.NewInternalServerError("failed to rotate refresh token hash", err)
	}
	return rows > 0, nil
}
