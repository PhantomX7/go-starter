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
