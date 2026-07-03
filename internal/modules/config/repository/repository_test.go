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
	"github.com/PhantomX7/athleton/pkg/pagination"
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

// seedPublicPrivate inserts one public and one private config row. IsPublic
// carries a default:false column tag, so the public row is flipped with an
// explicit Update (GORM drops false/true zero-value handling on insert).
func seedPublicPrivate(t *testing.T, db *gorm.DB) (public, private *models.Config) {
	t.Helper()

	public = &models.Config{Key: "site_name", Value: "Athleton"}
	private = &models.Config{Key: "smtp_password", Value: "hunter2"}
	require.NoError(t, db.Create(public).Error)
	require.NoError(t, db.Create(private).Error)
	require.NoError(t, db.Model(public).Update("is_public", true).Error)
	public.IsPublic = true
	return public, private
}

func TestConfigRepositoryFindAllPublicFiltersPrivateRows(t *testing.T) {
	db := setupDB(t)
	repo := configrepository.NewConfigRepository(db)

	public, _ := seedPublicPrivate(t, db)

	pg := pagination.NewPagination(nil, nil, pagination.PaginationOptions{DefaultLimit: 20})
	got, err := repo.FindAllPublic(context.Background(), pg)

	require.NoError(t, err)
	require.Len(t, got, 1, "private rows must never appear on the public listing")
	require.Equal(t, public.Key, got[0].Key)

	count, err := repo.CountPublic(context.Background(), pg)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func TestConfigRepositoryFindPublicByKeyHidesPrivateRows(t *testing.T) {
	db := setupDB(t)
	repo := configrepository.NewConfigRepository(db)

	public, private := seedPublicPrivate(t, db)

	got, err := repo.FindPublicByKey(context.Background(), public.Key)
	require.NoError(t, err)
	require.Equal(t, public.Key, got.Key)

	// A private key must be indistinguishable from a missing one.
	got, err = repo.FindPublicByKey(context.Background(), private.Key)
	require.Nil(t, got)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))

	got, err = repo.FindPublicByKey(context.Background(), "missing")
	require.Nil(t, got)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}
