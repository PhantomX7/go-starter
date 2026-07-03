package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	configrepomocks "github.com/PhantomX7/athleton/internal/modules/config/repository/mocks"
	"github.com/PhantomX7/athleton/internal/modules/config/service"
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func TestConfigServiceIndexReturnsConfigsAndMeta(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(
		map[string][]string{"limit": {"2"}, "offset": {"4"}},
		nil,
		pagination.PaginationOptions{},
	)
	expectedConfigs := []*models.Config{
		{Model: gorm.Model{ID: 1}, Key: "site_name", Value: "Athleton"},
		{Model: gorm.Model{ID: 2}, Key: "maintenance_mode", Value: "false"},
	}

	repo := &configrepomocks.ConfigRepositoryMock{
		FindAllFunc: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.Config, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return expectedConfigs, nil
		},
		CountFunc: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 15, nil
		},
	}

	svc := service.NewConfigService(repo, &logmocks.LogRepositoryMock{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	configs, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Equal(t, expectedConfigs, configs)
	require.Equal(t, int64(15), meta.Total)
	require.Equal(t, 4, meta.Offset)
	require.Equal(t, 2, meta.Limit)
}

func TestConfigServiceUpdateTogglesVisibilityOnlyWhenProvided(t *testing.T) {
	setupLogger(t)

	current := &models.Config{
		Model:    gorm.Model{ID: 7},
		Key:      "site_name",
		Value:    "v",
		IsPublic: false,
	}
	repo := &configrepomocks.ConfigRepositoryMock{
		FindByIDFunc: func(context.Context, uint, ...repository.Association) (*models.Config, error) {
			return current, nil
		},
		UpdateFunc: func(context.Context, *models.Config) error { return nil },
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(context.Context, *models.Log) error { return nil },
	}
	svc := service.NewConfigService(repo, logRepo)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root"})

	// Omitted is_public keeps the current visibility.
	_, err := svc.Update(ctx, 7, &dto.ConfigUpdateRequest{Value: "v2"})
	require.NoError(t, err)
	require.False(t, current.IsPublic)

	// An explicit true flips it.
	public := true
	_, err = svc.Update(ctx, 7, &dto.ConfigUpdateRequest{Value: "v3", IsPublic: &public})
	require.NoError(t, err)
	require.True(t, current.IsPublic)

	// An explicit false flips it back.
	private := false
	_, err = svc.Update(ctx, 7, &dto.ConfigUpdateRequest{Value: "v4", IsPublic: &private})
	require.NoError(t, err)
	require.False(t, current.IsPublic)
}

func TestConfigServicePublicIndexUsesPublicRepositoryVariants(t *testing.T) {
	setupLogger(t)

	repo := &configrepomocks.ConfigRepositoryMock{
		FindAllPublicFunc: func(_ context.Context, pg *pagination.Pagination) ([]*models.Config, error) {
			return []*models.Config{{Key: "site_name", Value: "Athleton", IsPublic: true}}, nil
		},
		CountPublicFunc: func(context.Context, *pagination.Pagination) (int64, error) {
			return 1, nil
		},
	}
	svc := service.NewConfigService(repo, &logmocks.LogRepositoryMock{})

	pg := pagination.NewPagination(nil, nil, pagination.PaginationOptions{DefaultLimit: 20})
	configs, meta, err := svc.PublicIndex(context.Background(), pg)

	require.NoError(t, err)
	require.Len(t, configs, 1)
	require.EqualValues(t, 1, meta.Total)
}

func TestConfigServiceUpdateUpdatesConfigAndCreatesAuditLog(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	current := &models.Config{
		Model: gorm.Model{ID: 7},
		Key:   "site_name",
		Value: "Old Value",
	}

	repo := &configrepomocks.ConfigRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Config, error) {
			require.Equal(t, uint(7), id)
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.Config) error {
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			require.Equal(t, "New Value", entity.Value)
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			require.NotNil(t, ctx)
			logCh <- entry
			return nil
		},
	}

	svc := service.NewConfigService(repo, logRepo)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{
		UserID:   42,
		UserName: "Alice",
	})

	updated, err := svc.Update(ctx, 7, &dto.ConfigUpdateRequest{Value: "New Value"})

	require.NoError(t, err)
	require.Same(t, current, updated)
	require.Equal(t, "New Value", updated.Value)

	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionUpdate, entry.Action)
		require.Equal(t, models.LogEntityTypeConfig, entry.EntityType)
		require.Equal(t, uint(7), entry.EntityID)
		require.Equal(t, "Alice updated config: site_name", entry.Message)
		require.NotNil(t, entry.UserID)
		require.Equal(t, uint(42), *entry.UserID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestConfigServiceFindByKeyReturnsConfig(t *testing.T) {
	setupLogger(t)

	expected := &models.Config{
		Model: gorm.Model{ID: 3},
		Key:   "timezone",
		Value: "Asia/Jakarta",
	}
	repo := &configrepomocks.ConfigRepositoryMock{
		FindByKeyFunc: func(ctx context.Context, key string) (*models.Config, error) {
			require.Equal(t, "timezone", key)
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			return expected, nil
		},
	}

	svc := service.NewConfigService(repo, &logmocks.LogRepositoryMock{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")

	got, err := svc.FindByKey(ctx, "timezone")

	require.NoError(t, err)
	require.Same(t, expected, got)
}

func TestConfigServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &configrepomocks.ConfigRepositoryMock{
		FindAllFunc: func(context.Context, *pagination.Pagination) ([]*models.Config, error) {
			return nil, expectedErr
		},
		CountFunc: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewConfigService(repo, &logmocks.LogRepositoryMock{})

	configs, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, configs)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
