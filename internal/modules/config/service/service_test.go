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
	configrepository "github.com/PhantomX7/athleton/internal/modules/config/repository"
	"github.com/PhantomX7/athleton/internal/modules/config/service"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type mockConfigRepository struct {
	findAllFn   func(context.Context, *pagination.Pagination) ([]*models.Config, error)
	countFn     func(context.Context, *pagination.Pagination) (int64, error)
	findByIDFn  func(context.Context, uint, ...repository.Association) (*models.Config, error)
	updateFn    func(context.Context, *models.Config) error
	findByKeyFn func(context.Context, string) (*models.Config, error)
}

func (m *mockConfigRepository) Create(context.Context, *models.Config) error {
	panic("unexpected Create call")
}

func (m *mockConfigRepository) Update(ctx context.Context, entity *models.Config) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}

func (m *mockConfigRepository) Delete(context.Context, *models.Config) error {
	panic("unexpected Delete call")
}

func (m *mockConfigRepository) FindById(ctx context.Context, id uint, preloads ...repository.Association) (*models.Config, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mockConfigRepository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mockConfigRepository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
}

func (m *mockConfigRepository) FindByKey(ctx context.Context, key string) (*models.Config, error) {
	if m.findByKeyFn == nil {
		panic("unexpected FindByKey call")
	}
	return m.findByKeyFn(ctx, key)
}

var _ configrepository.ConfigRepository = (*mockConfigRepository)(nil)

type mockLogRepository struct {
	createFn func(context.Context, *models.Log) error
}

func (m *mockLogRepository) Create(ctx context.Context, entity *models.Log) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}

func (m *mockLogRepository) Update(context.Context, *models.Log) error {
	panic("unexpected Update call")
}

func (m *mockLogRepository) Delete(context.Context, *models.Log) error {
	panic("unexpected Delete call")
}

func (m *mockLogRepository) FindById(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindById call")
}

func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}

func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

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

	repo := &mockConfigRepository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.Config, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return expectedConfigs, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 15, nil
		},
	}

	svc := service.NewConfigService(repo, &mockLogRepository{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	configs, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Equal(t, expectedConfigs, configs)
	require.Equal(t, int64(15), meta.Total)
	require.Equal(t, 4, meta.Offset)
	require.Equal(t, 2, meta.Limit)
}

func TestConfigServiceUpdateUpdatesConfigAndCreatesAuditLog(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	current := &models.Config{
		Model: gorm.Model{ID: 7},
		Key:   "site_name",
		Value: "Old Value",
	}

	repo := &mockConfigRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Config, error) {
			require.Equal(t, uint(7), id)
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.Config) error {
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			require.Equal(t, "New Value", entity.Value)
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
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
	repo := &mockConfigRepository{
		findByKeyFn: func(ctx context.Context, key string) (*models.Config, error) {
			require.Equal(t, "timezone", key)
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			return expected, nil
		},
	}

	svc := service.NewConfigService(repo, &mockLogRepository{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")

	got, err := svc.FindByKey(ctx, "timezone")

	require.NoError(t, err)
	require.Same(t, expected, got)
}

func TestConfigServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mockConfigRepository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.Config, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewConfigService(repo, &mockLogRepository{})

	configs, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, configs)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
