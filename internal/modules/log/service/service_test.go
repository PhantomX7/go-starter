package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/PhantomX7/athleton/internal/models"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/internal/modules/log/service"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockLogRepository struct {
	findAllFn  func(context.Context, *pagination.Pagination) ([]*models.Log, error)
	countFn    func(context.Context, *pagination.Pagination) (int64, error)
	findByIDFn func(context.Context, uint, ...repository.Association) (*models.Log, error)
}

func (m *mockLogRepository) Create(context.Context, *models.Log) error {
	panic("unexpected Create call")
}

func (m *mockLogRepository) Update(context.Context, *models.Log) error {
	panic("unexpected Update call")
}

func (m *mockLogRepository) Delete(context.Context, *models.Log) error {
	panic("unexpected Delete call")
}

func (m *mockLogRepository) FindById(ctx context.Context, id uint, preloads ...repository.Association) (*models.Log, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mockLogRepository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.Log, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mockLogRepository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
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

func TestLogServiceIndexReturnsLogsAndMeta(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(
		map[string][]string{"limit": {"5"}, "offset": {"10"}},
		nil,
		pagination.PaginationOptions{},
	)
	expectedLogs := []*models.Log{
		{ID: 1, Message: "created"},
		{ID: 2, Message: "updated"},
	}

	repo := &mockLogRepository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.Log, error) {
			require.Equal(t, "req-123", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return expectedLogs, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-123", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 42, nil
		},
	}

	svc := service.NewLogService(repo)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-123")

	logs, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Equal(t, expectedLogs, logs)
	require.Equal(t, int64(42), meta.Total)
	require.Equal(t, 10, meta.Offset)
	require.Equal(t, 5, meta.Limit)
}

func TestLogServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mockLogRepository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.Log, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewLogService(repo)

	logs, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, logs)
	require.Equal(t, int64(0), meta.Total)
	require.Equal(t, 0, meta.Offset)
	require.Equal(t, 0, meta.Limit)
}

func TestLogServiceFindByIDReturnsLog(t *testing.T) {
	setupLogger(t)

	expectedLog := &models.Log{ID: 7, Message: "found"}
	repo := &mockLogRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Log, error) {
			require.Equal(t, "req-456", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(7), id)
			return expectedLog, nil
		},
	}

	svc := service.NewLogService(repo)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-456")

	logEntry, err := svc.FindById(ctx, 7)

	require.NoError(t, err)
	require.Same(t, expectedLog, logEntry)
}

func TestLogServiceFindByIDReturnsNotFoundError(t *testing.T) {
	setupLogger(t)

	expectedErr := cerrors.NewNotFoundError("log not found")
	repo := &mockLogRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.Log, error) {
			return nil, expectedErr
		},
	}

	svc := service.NewLogService(repo)

	logEntry, err := svc.FindById(context.Background(), 99)

	require.Nil(t, logEntry)
	require.ErrorIs(t, err, cerrors.ErrNotFound)
	require.ErrorIs(t, err, expectedErr)
}
