package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/PhantomX7/athleton/internal/models"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
