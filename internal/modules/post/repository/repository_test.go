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
	postrepository "github.com/PhantomX7/athleton/internal/modules/post/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Post{}))

	return db
}

func TestPostRepositoryCreatePersistsPost(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	post := &models.Post{Name: "First Post", Description: "hello", IsActive: true}

	require.NoError(t, repo.Create(context.Background(), post))
	require.NotZero(t, post.ID)

	var stored models.Post
	require.NoError(t, db.First(&stored, post.ID).Error)
	require.Equal(t, "First Post", stored.Name)
	require.Equal(t, "hello", stored.Description)
}

func TestPostRepositoryFindByIDReturnsPost(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	seed := &models.Post{Name: "Seeded", Description: "seeded post"}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByID(context.Background(), seed.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "Seeded", got.Name)
}

func TestPostRepositoryFindByIDReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	got, err := repo.FindByID(context.Background(), 999)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestPostRepositoryUpdateChangesPost(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	seed := &models.Post{Name: "Old Name"}
	require.NoError(t, db.Create(seed).Error)

	seed.Name = "New Name"
	require.NoError(t, repo.Update(context.Background(), seed))

	var stored models.Post
	require.NoError(t, db.First(&stored, seed.ID).Error)
	require.Equal(t, "New Name", stored.Name)
}

func TestPostRepositoryDeleteRemovesPost(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	seed := &models.Post{Name: "Doomed"}
	require.NoError(t, db.Create(seed).Error)

	require.NoError(t, repo.Delete(context.Background(), seed))

	_, err := repo.FindByID(context.Background(), seed.ID)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestPostRepositoryFindAllAndCountHonorPagination(t *testing.T) {
	db := setupDB(t)
	repo := postrepository.NewPostRepository(db)

	for _, name := range []string{"a", "b", "c"} {
		require.NoError(t, db.Create(&models.Post{Name: name}).Error)
	}

	pg := pagination.NewPagination(map[string][]string{
		"limit":  {"2"},
		"offset": {"0"},
	}, nil, pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, DefaultOrder: "id asc"})

	posts, err := repo.FindAll(context.Background(), pg)
	require.NoError(t, err)
	require.Len(t, posts, 2)

	count, err := repo.Count(context.Background(), pg)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}
