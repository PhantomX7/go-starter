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
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.AdminRole{}, &models.Log{}))

	return db
}

func seedUser(t *testing.T, db *gorm.DB) *models.User {
	t.Helper()

	user := &models.User{
		Username: "actor",
		Name:     "Actor",
		Email:    "actor@example.com",
		Phone:    "0812345",
		IsActive: true,
		Role:     models.UserRoleAdmin,
		Password: "hashed",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestLogRepositoryCreatePersistsLog(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	user := seedUser(t, db)
	log := &models.Log{
		UserID:     &user.ID,
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeConfig,
		EntityID:   7,
		Message:    "Actor created config",
	}

	require.NoError(t, repo.Create(context.Background(), log))
	require.NotZero(t, log.ID)

	var stored models.Log
	require.NoError(t, db.First(&stored, log.ID).Error)
	require.NotNil(t, stored.UserID)
	require.Equal(t, user.ID, *stored.UserID)
	require.Equal(t, models.LogActionCreate, stored.Action)
	require.Equal(t, models.LogEntityTypeConfig, stored.EntityType)
	require.Equal(t, uint(7), stored.EntityID)
	require.Equal(t, "Actor created config", stored.Message)
}

func TestLogRepositoryCreateAllowsNilUserID(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	log := &models.Log{
		Action:     models.LogActionLogout,
		EntityType: models.LogEntityTypeUser,
		EntityID:   1,
		Message:    "anonymous action",
	}

	require.NoError(t, repo.Create(context.Background(), log))

	var stored models.Log
	require.NoError(t, db.First(&stored, log.ID).Error)
	require.Nil(t, stored.UserID)
}

func TestLogRepositoryFindByIDReturnsLog(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	seed := &models.Log{
		Action:     models.LogActionUpdate,
		EntityType: models.LogEntityTypeAdminRole,
		EntityID:   3,
		Message:    "role updated",
	}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByID(context.Background(), seed.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, models.LogActionUpdate, got.Action)
	require.Equal(t, "role updated", got.Message)
}

func TestLogRepositoryFindByIDPreloadsUser(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	user := seedUser(t, db)
	seed := &models.Log{
		UserID:     &user.ID,
		Action:     models.LogActionLogin,
		EntityType: models.LogEntityTypeUser,
		EntityID:   user.ID,
		Message:    "Actor logged in",
	}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByID(context.Background(), seed.ID, repository.Preload("User"))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.User)
	require.Equal(t, user.ID, got.User.ID)
	require.Equal(t, "actor", got.User.Username)
}

func TestLogRepositoryFindByIDReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	got, err := repo.FindByID(context.Background(), 999)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestLogRepositoryUpdateChangesLog(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	seed := &models.Log{
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeConfig,
		EntityID:   1,
		Message:    "old message",
	}
	require.NoError(t, db.Create(seed).Error)

	seed.Message = "new message"
	require.NoError(t, repo.Update(context.Background(), seed))

	var stored models.Log
	require.NoError(t, db.First(&stored, seed.ID).Error)
	require.Equal(t, "new message", stored.Message)
}

func TestLogRepositoryDeleteRemovesLog(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	seed := &models.Log{
		Action:     models.LogActionDelete,
		EntityType: models.LogEntityTypeUser,
		EntityID:   2,
		Message:    "doomed",
	}
	require.NoError(t, db.Create(seed).Error)

	require.NoError(t, repo.Delete(context.Background(), seed))

	_, err := repo.FindByID(context.Background(), seed.ID)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestLogRepositoryFindAllAndCountHonorPagination(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	for i, action := range []models.LogAction{
		models.LogActionCreate,
		models.LogActionUpdate,
		models.LogActionDelete,
	} {
		require.NoError(t, db.Create(&models.Log{
			Action:     action,
			EntityType: models.LogEntityTypeConfig,
			EntityID:   uint(i + 1),
			Message:    string(action),
		}).Error)
	}

	pg := pagination.NewPagination(map[string][]string{
		"limit":  {"2"},
		"offset": {"0"},
	}, nil, pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, DefaultOrder: "id asc"})

	logs, err := repo.FindAll(context.Background(), pg)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Equal(t, models.LogActionCreate, logs[0].Action)
	require.Equal(t, models.LogActionUpdate, logs[1].Action)

	count, err := repo.Count(context.Background(), pg)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}

func TestLogRepositoryFindAllReturnsEmptySlice(t *testing.T) {
	db := setupDB(t)
	repo := logrepository.NewLogRepository(db)

	pg := pagination.NewPagination(map[string][]string{}, nil,
		pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, DefaultOrder: "id asc"})

	logs, err := repo.FindAll(context.Background(), pg)
	require.NoError(t, err)
	require.NotNil(t, logs)
	require.Empty(t, logs)

	count, err := repo.Count(context.Background(), pg)
	require.NoError(t, err)
	require.Zero(t, count)
}
