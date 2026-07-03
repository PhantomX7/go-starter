package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/internal/models"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AdminRole{}, &models.User{}, &models.RefreshToken{}))

	return db
}

func seedUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()

	user := &models.User{
		Username: username,
		Name:     username,
		Email:    username + "@example.com",
		Phone:    "08123456789",
		IsActive: true,
		Role:     models.UserRoleUser,
		Password: "secret",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// seedToken inserts a refresh-token row directly via gorm, bypassing the
// repository's Create override. We hash the plaintext here so the row matches
// what FindByToken / RevokeByToken will look up (they hash their inputs). Tests
// that need pre-revoked or expired rows still get full control over RevokedAt
// and ExpiresAt without going through the production Create path.
func seedToken(t *testing.T, db *gorm.DB, userID uint, token string, expiresAt time.Time, revokedAt *time.Time) *models.RefreshToken {
	t.Helper()

	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     refreshtokenrepository.HashRefreshToken(token),
		ExpiresAt: expiresAt,
		RevokedAt: revokedAt,
	}
	require.NoError(t, db.Create(rt).Error)
	return rt
}

func TestRefreshTokenRepositoryCreateStoresHashedTokenAndRoundTrips(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "alma")

	plaintext := "wire-value-from-client"
	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     plaintext,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(context.Background(), rt))

	var stored models.RefreshToken
	require.NoError(t, db.First(&stored, "id = ?", rt.ID).Error)
	require.NotEqual(t, plaintext, stored.Token, "plaintext must not be stored at rest")
	require.Equal(t, refreshtokenrepository.HashRefreshToken(plaintext), stored.Token)

	got, err := repo.FindByToken(context.Background(), plaintext)
	require.NoError(t, err)
	require.Equal(t, rt.ID, got.ID)

	require.NoError(t, repo.RevokeByToken(context.Background(), plaintext))

	_, err = repo.FindByToken(context.Background(), plaintext)
	require.Error(t, err)
}

func TestRefreshTokenRepositoryFindByTokenReturnsActiveToken(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "alice")
	seed := seedToken(t, db, user.ID, "active-token", time.Now().Add(time.Hour), nil)

	got, err := repo.FindByToken(context.Background(), "active-token")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	// Token column stores the hash, not the plaintext — see Create override.
	require.Equal(t, refreshtokenrepository.HashRefreshToken("active-token"), got.Token)
}

func TestRefreshTokenRepositoryFindByTokenRejectsExpiredOrRevoked(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "bob")
	now := time.Now()
	seedToken(t, db, user.ID, "expired-token", now.Add(-time.Hour), nil)
	seedToken(t, db, user.ID, "revoked-token", now.Add(time.Hour), &now)

	gotExpired, errExpired := repo.FindByToken(context.Background(), "expired-token")
	gotRevoked, errRevoked := repo.FindByToken(context.Background(), "revoked-token")

	require.Nil(t, gotExpired)
	require.Error(t, errExpired)
	require.True(t, errors.Is(errExpired, cerrors.ErrNotFound))
	require.Nil(t, gotRevoked)
	require.Error(t, errRevoked)
	require.True(t, errors.Is(errRevoked, cerrors.ErrNotFound))
}

func TestRefreshTokenRepositoryGetValidCountByUserIDCountsOnlyActive(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "charlie")
	other := seedUser(t, db, "david")
	now := time.Now()

	seedToken(t, db, user.ID, "active-1", now.Add(time.Hour), nil)
	seedToken(t, db, user.ID, "active-2", now.Add(2*time.Hour), nil)
	seedToken(t, db, user.ID, "expired", now.Add(-time.Hour), nil)
	seedToken(t, db, user.ID, "revoked", now.Add(time.Hour), &now)
	seedToken(t, db, other.ID, "other-user", now.Add(time.Hour), nil)

	count, err := repo.GetValidCountByUserID(context.Background(), user.ID)

	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}

func TestRefreshTokenRepositoryDeleteInvalidTokenRemovesExpiredAndRevoked(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "eve")
	now := time.Now()

	active := seedToken(t, db, user.ID, "active", now.Add(time.Hour), nil)
	expired := seedToken(t, db, user.ID, "expired", now.Add(-time.Hour), nil)
	revoked := seedToken(t, db, user.ID, "revoked", now.Add(time.Hour), &now)

	require.NoError(t, repo.DeleteInvalidToken(context.Background()))

	var count int64
	require.NoError(t, db.Model(&models.RefreshToken{}).Count(&count).Error)
	require.EqualValues(t, 1, count)

	var kept models.RefreshToken
	require.NoError(t, db.First(&kept, "id = ?", active.ID).Error)

	var removedCount int64
	require.NoError(t, db.Model(&models.RefreshToken{}).Where("id IN ?", []uuid.UUID{expired.ID, revoked.ID}).Count(&removedCount).Error)
	require.EqualValues(t, 0, removedCount)
}

func TestRefreshTokenRepositoryRevokeAllByUserIDRevokesOnlyActiveUserTokens(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "frank")
	other := seedUser(t, db, "grace")
	now := time.Now()

	active1 := seedToken(t, db, user.ID, "active-1", now.Add(time.Hour), nil)
	active2 := seedToken(t, db, user.ID, "active-2", now.Add(2*time.Hour), nil)
	expired := seedToken(t, db, user.ID, "expired", now.Add(-time.Hour), nil)
	otherActive := seedToken(t, db, other.ID, "other-active", now.Add(time.Hour), nil)

	require.NoError(t, repo.RevokeAllByUserID(context.Background(), user.ID))

	var gotActive1, gotActive2, gotExpired, gotOther models.RefreshToken
	require.NoError(t, db.First(&gotActive1, "id = ?", active1.ID).Error)
	require.NoError(t, db.First(&gotActive2, "id = ?", active2.ID).Error)
	require.NoError(t, db.First(&gotExpired, "id = ?", expired.ID).Error)
	require.NoError(t, db.First(&gotOther, "id = ?", otherActive.ID).Error)

	require.NotNil(t, gotActive1.RevokedAt)
	require.NotNil(t, gotActive2.RevokedAt)
	require.Nil(t, gotExpired.RevokedAt)
	require.Nil(t, gotOther.RevokedAt)
}

func TestRefreshTokenRepositoryRevokeAllByUserIDExceptKeepsSpecifiedToken(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "henry")
	now := time.Now()

	keep := seedToken(t, db, user.ID, "keep-me", now.Add(time.Hour), nil)
	revoke := seedToken(t, db, user.ID, "revoke-me", now.Add(time.Hour), nil)

	require.NoError(t, repo.RevokeAllByUserIDExcept(context.Background(), user.ID, "keep-me"))

	var keptToken, revokedToken models.RefreshToken
	require.NoError(t, db.First(&keptToken, "id = ?", keep.ID).Error)
	require.NoError(t, db.First(&revokedToken, "id = ?", revoke.ID).Error)

	require.Nil(t, keptToken.RevokedAt)
	require.NotNil(t, revokedToken.RevokedAt)
}

func TestRefreshTokenRepositoryFindActiveByIDReturnsOnlyActive(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "judy")
	now := time.Now()

	active := seedToken(t, db, user.ID, "active", now.Add(time.Hour), nil)
	expired := seedToken(t, db, user.ID, "expired", now.Add(-time.Hour), nil)
	revoked := seedToken(t, db, user.ID, "revoked", now.Add(time.Hour), &now)

	got, err := repo.FindActiveByID(context.Background(), active.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, active.ID, got.ID)

	_, err = repo.FindActiveByID(context.Background(), expired.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))

	_, err = repo.FindActiveByID(context.Background(), revoked.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))

	_, err = repo.FindActiveByID(context.Background(), uuid.New())
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestRefreshTokenRepositoryRevokeByTokenRevokesMatchingToken(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "ivan")
	token := seedToken(t, db, user.ID, "single-token", time.Now().Add(time.Hour), nil)

	require.NoError(t, repo.RevokeByToken(context.Background(), "single-token"))

	var got models.RefreshToken
	require.NoError(t, db.First(&got, "id = ?", token.ID).Error)
	require.NotNil(t, got.RevokedAt)
}

func TestRefreshTokenRepositoryRevokeByTokenIfActiveReportsReuse(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "ivan")
	seedToken(t, db, user.ID, "rotate-once", time.Now().Add(time.Hour), nil)

	// First revoke wins and reports a row was revoked.
	revoked, err := repo.RevokeByTokenIfActive(context.Background(), "rotate-once")
	require.NoError(t, err)
	require.True(t, revoked)

	// Second revoke of the same token finds nothing active: the reuse signal.
	revoked, err = repo.RevokeByTokenIfActive(context.Background(), "rotate-once")
	require.NoError(t, err)
	require.False(t, revoked)
}

// Rotate-in-place: the SAME row must survive with only its token hash
// replaced — the ID (access-token jti binding) and expires_at (absolute
// session lifetime) stay untouched, and afterwards only the new plaintext
// resolves the session.
func TestRefreshTokenRepositoryUpdateTokenHashIfActiveRotatesInPlace(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "kate")
	expiresAt := time.Now().Add(time.Hour)
	seed := seedToken(t, db, user.ID, "before-rotation", expiresAt, nil)

	updated, err := repo.UpdateTokenHashIfActive(context.Background(), "before-rotation", "after-rotation")
	require.NoError(t, err)
	require.True(t, updated)

	var stored models.RefreshToken
	require.NoError(t, db.First(&stored, "id = ?", seed.ID).Error)
	require.Equal(t, refreshtokenrepository.HashRefreshToken("after-rotation"), stored.Token)
	require.WithinDuration(t, expiresAt, stored.ExpiresAt, time.Second,
		"rotation must NOT extend the session: expires_at is the absolute lifetime set at login")
	require.Nil(t, stored.RevokedAt)

	// The new plaintext resolves to the same session row; the old one is gone.
	got, err := repo.FindByToken(context.Background(), "after-rotation")
	require.NoError(t, err)
	require.Equal(t, seed.ID, got.ID)

	_, err = repo.FindByToken(context.Background(), "before-rotation")
	require.Error(t, err)
}

// Rotation records the hash it replaced in previous_token_hash so a later
// replay of the old token can be recognized as reuse, and FindByPreviousToken
// resolves that superseded value back to the owning row.
func TestRefreshTokenRepositoryRotationRecordsPreviousHashForReuseDetection(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "mona")
	seed := seedToken(t, db, user.ID, "gen-1", time.Now().Add(time.Hour), nil)

	updated, err := repo.UpdateTokenHashIfActive(context.Background(), "gen-1", "gen-2")
	require.NoError(t, err)
	require.True(t, updated)

	// The rotated-away hash is stored on the row.
	var stored models.RefreshToken
	require.NoError(t, db.First(&stored, "id = ?", seed.ID).Error)
	require.NotNil(t, stored.PreviousTokenHash)
	require.Equal(t, refreshtokenrepository.HashRefreshToken("gen-1"), *stored.PreviousTokenHash)

	// Replaying the superseded token resolves to the owning row via the
	// previous-token lookup — this is what turns a replay into a reuse signal.
	superseded, err := repo.FindByPreviousToken(context.Background(), "gen-1")
	require.NoError(t, err)
	require.Equal(t, seed.ID, superseded.ID)
	require.Equal(t, user.ID, superseded.UserID)

	// A token that was never anyone's predecessor is not found.
	_, err = repo.FindByPreviousToken(context.Background(), "never-seen")
	require.ErrorIs(t, err, cerrors.ErrNotFound)
}

func TestRefreshTokenRepositoryUpdateTokenHashIfActiveReportsReuse(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "liam")
	now := time.Now()
	seedToken(t, db, user.ID, "rotate-me", now.Add(time.Hour), nil)
	seedToken(t, db, user.ID, "already-revoked", now.Add(time.Hour), &now)

	// First rotation wins.
	updated, err := repo.UpdateTokenHashIfActive(context.Background(), "rotate-me", "fresh-1")
	require.NoError(t, err)
	require.True(t, updated)

	// Replaying the pre-rotation value matches no row: the reuse signal. The
	// concurrent-rotation race collapses to this same case — the loser's WHERE
	// token = old-hash no longer matches after the winner rewrote it.
	updated, err = repo.UpdateTokenHashIfActive(context.Background(), "rotate-me", "fresh-2")
	require.NoError(t, err)
	require.False(t, updated)

	// A revoked row must never be rotated back to life.
	updated, err = repo.UpdateTokenHashIfActive(context.Background(), "already-revoked", "fresh-3")
	require.NoError(t, err)
	require.False(t, updated)

	// Nonexistent tokens report false rather than erroring.
	updated, err = repo.UpdateTokenHashIfActive(context.Background(), "never-existed", "fresh-4")
	require.NoError(t, err)
	require.False(t, updated)
}

// setCreatedAt backdates a seeded row so ordering by creation time is
// deterministic regardless of insert timing.
func setCreatedAt(t *testing.T, db *gorm.DB, id uuid.UUID, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&models.RefreshToken{}).Where("id = ?", id).Update("created_at", createdAt).Error)
}

func TestRefreshTokenRepositoryRevokeOldestActiveByUserIDRevokesOldestFirst(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "mona")
	other := seedUser(t, db, "nick")
	now := time.Now()

	oldest := seedToken(t, db, user.ID, "oldest", now.Add(time.Hour), nil)
	middle := seedToken(t, db, user.ID, "middle", now.Add(time.Hour), nil)
	newest := seedToken(t, db, user.ID, "newest", now.Add(time.Hour), nil)
	// Inactive rows must be invisible to the cap even when they are the oldest.
	expired := seedToken(t, db, user.ID, "expired", now.Add(-time.Hour), nil)
	otherActive := seedToken(t, db, other.ID, "other-user", now.Add(time.Hour), nil)

	setCreatedAt(t, db, expired.ID, now.Add(-4*time.Hour))
	setCreatedAt(t, db, oldest.ID, now.Add(-3*time.Hour))
	setCreatedAt(t, db, middle.ID, now.Add(-2*time.Hour))
	setCreatedAt(t, db, newest.ID, now.Add(-time.Hour))
	setCreatedAt(t, db, otherActive.ID, now.Add(-5*time.Hour))

	require.NoError(t, repo.RevokeOldestActiveByUserID(context.Background(), user.ID, 2))

	var gotOldest, gotMiddle, gotNewest, gotExpired, gotOther models.RefreshToken
	require.NoError(t, db.First(&gotOldest, "id = ?", oldest.ID).Error)
	require.NoError(t, db.First(&gotMiddle, "id = ?", middle.ID).Error)
	require.NoError(t, db.First(&gotNewest, "id = ?", newest.ID).Error)
	require.NoError(t, db.First(&gotExpired, "id = ?", expired.ID).Error)
	require.NoError(t, db.First(&gotOther, "id = ?", otherActive.ID).Error)

	require.NotNil(t, gotOldest.RevokedAt, "the oldest active session must be revoked")
	require.NotNil(t, gotMiddle.RevokedAt, "the second-oldest active session must be revoked")
	require.Nil(t, gotNewest.RevokedAt, "newer sessions within the cap must survive")
	require.Nil(t, gotExpired.RevokedAt, "expired rows are already dead and must not be stamped")
	require.Nil(t, gotOther.RevokedAt, "other users' sessions must never be touched")
}

func TestRefreshTokenRepositoryRevokeOldestActiveByUserIDNoOpCases(t *testing.T) {
	db := setupDB(t)
	repo := refreshtokenrepository.NewRefreshTokenRepository(db)
	user := seedUser(t, db, "olga")
	active := seedToken(t, db, user.ID, "active", time.Now().Add(time.Hour), nil)

	// Zero or negative n asks for nothing.
	require.NoError(t, repo.RevokeOldestActiveByUserID(context.Background(), user.ID, 0))
	require.NoError(t, repo.RevokeOldestActiveByUserID(context.Background(), user.ID, -1))
	// A user without active tokens has nothing to revoke.
	require.NoError(t, repo.RevokeOldestActiveByUserID(context.Background(), user.ID+1000, 3))

	var got models.RefreshToken
	require.NoError(t, db.First(&got, "id = ?", active.ID).Error)
	require.Nil(t, got.RevokedAt)
}
