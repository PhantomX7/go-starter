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
	adminrolerepository "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
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

func TestAdminRoleRepositoryFindByNameReturnsRole(t *testing.T) {
	db := setupDB(t)
	repo := adminrolerepository.NewAdminRoleRepository(db)

	seed := &models.AdminRole{Name: "Manager", Description: "Can manage products", IsActive: true}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByName(context.Background(), "Manager")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "Manager", got.Name)
}

func TestAdminRoleRepositoryFindByNameReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := adminrolerepository.NewAdminRoleRepository(db)

	got, err := repo.FindByName(context.Background(), "missing")

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestAdminRoleRepositoryCountUsersWithRoleCountsActiveUsersOnly(t *testing.T) {
	db := setupDB(t)
	repo := adminrolerepository.NewAdminRoleRepository(db)

	role := &models.AdminRole{Name: "Support", Description: "Support team", IsActive: true}
	require.NoError(t, db.Create(role).Error)

	active := &models.User{
		Username:    "alice",
		Name:        "Alice",
		Email:       "alice@example.com",
		Phone:       "08123456789",
		IsActive:    true,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &role.ID,
		Password:    "secret",
	}
	deleted := &models.User{
		Username:    "bob",
		Name:        "Bob",
		Email:       "bob@example.com",
		Phone:       "08123456780",
		IsActive:    true,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &role.ID,
		Password:    "secret",
	}
	otherRole := &models.AdminRole{Name: "Writer", Description: "Writer team", IsActive: true}
	require.NoError(t, db.Create(otherRole).Error)
	other := &models.User{
		Username:    "charlie",
		Name:        "Charlie",
		Email:       "charlie@example.com",
		Phone:       "08123456781",
		IsActive:    true,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &otherRole.ID,
		Password:    "secret",
	}

	require.NoError(t, db.Create(active).Error)
	require.NoError(t, db.Create(deleted).Error)
	require.NoError(t, db.Create(other).Error)
	require.NoError(t, db.Delete(deleted).Error)

	count, err := repo.CountUsersWithRole(context.Background(), role.ID)

	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
