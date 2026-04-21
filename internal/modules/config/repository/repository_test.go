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
	configrepository "github.com/PhantomX7/athleton/internal/modules/config/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}))

	return db
}

func TestConfigRepositoryFindByKeyReturnsConfig(t *testing.T) {
	db := setupDB(t)
	repo := configrepository.NewConfigRepository(db)

	seed := &models.Config{Key: "site_name", Value: "Athleton"}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByKey(context.Background(), "site_name")

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "site_name", got.Key)
	require.Equal(t, "Athleton", got.Value)
}

func TestConfigRepositoryFindByKeyReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := configrepository.NewConfigRepository(db)

	got, err := repo.FindByKey(context.Background(), "missing")

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}
