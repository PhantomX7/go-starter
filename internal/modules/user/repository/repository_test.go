package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/internal/models"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AdminRole{}, &models.User{}))

	return db
}

func TestUserRepositoryFindByUsernameReturnsUser(t *testing.T) {
	db := setupDB(t)
	repo := userrepository.NewUserRepository(db)

	seed := &models.User{
		Username: "alice",
		Name:     "Alice",
		Email:    "alice@example.com",
		Phone:    "08123456789",
		IsActive: true,
		Role:     models.UserRoleUser,
		Password: "secret",
	}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByUsername(context.Background(), "alice")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "alice", got.Username)
}

func TestUserRepositoryFindByUsernameReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := userrepository.NewUserRepository(db)

	got, err := repo.FindByUsername(context.Background(), "missing")

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestUserRepositoryFindByIDForUpdateReturnsUser(t *testing.T) {
	db := setupDB(t)
	repo := userrepository.NewUserRepository(db)

	seed := &models.User{
		Username: "carol",
		Name:     "Carol",
		Email:    "carol@example.com",
		Phone:    "08123456781",
		IsActive: true,
		Role:     models.UserRoleAdmin,
		Password: "secret",
	}
	require.NoError(t, db.Create(seed).Error)

	// Exercise the locked find inside a transaction context, as production
	// callers do (the FOR UPDATE clause is only meaningful inside one).
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := utils.SetTxToContext(context.Background(), tx)

		got, err := repo.FindByIDForUpdate(ctx, seed.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, seed.ID, got.ID)
		require.Equal(t, "carol", got.Username)
		return nil
	})
	require.NoError(t, err)
}

func TestUserRepositoryFindByIDForUpdateReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := userrepository.NewUserRepository(db)

	got, err := repo.FindByIDForUpdate(context.Background(), 999)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestUserRepositoryFindByEmailNormalizesInput(t *testing.T) {
	db := setupDB(t)
	repo := userrepository.NewUserRepository(db)

	seed := &models.User{
		Username: "bob",
		Name:     "Bob",
		Email:    "bob@example.com",
		Phone:    "08123456780",
		IsActive: true,
		Role:     models.UserRoleUser,
		Password: "secret",
	}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByEmail(context.Background(), "  BOB@EXAMPLE.COM ")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "bob@example.com", got.Email)
}
