package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/cron/service"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
)

type mockRefreshTokenRepository struct {
	deleteInvalidTokenFn func(context.Context) error
}

func (m *mockRefreshTokenRepository) Create(context.Context, *models.RefreshToken) error {
	panic("unexpected Create call")
}
func (m *mockRefreshTokenRepository) Update(context.Context, *models.RefreshToken) error {
	panic("unexpected Update call")
}
func (m *mockRefreshTokenRepository) Delete(context.Context, *models.RefreshToken) error {
	panic("unexpected Delete call")
}
func (m *mockRefreshTokenRepository) FindByID(context.Context, uint, ...repository.Association) (*models.RefreshToken, error) {
	panic("unexpected FindByID call")
}
func (m *mockRefreshTokenRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.RefreshToken, error) {
	panic("unexpected FindAll call")
}
func (m *mockRefreshTokenRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockRefreshTokenRepository) FindByToken(context.Context, string) (*models.RefreshToken, error) {
	panic("unexpected FindByToken call")
}
func (m *mockRefreshTokenRepository) FindActiveByID(context.Context, uuid.UUID) (*models.RefreshToken, error) {
	panic("unexpected FindActiveByID call")
}
func (m *mockRefreshTokenRepository) GetValidCountByUserID(context.Context, uint) (int64, error) {
	panic("unexpected GetValidCountByUserID call")
}
func (m *mockRefreshTokenRepository) DeleteInvalidToken(ctx context.Context) error {
	if m.deleteInvalidTokenFn == nil {
		panic("unexpected DeleteInvalidToken call")
	}
	return m.deleteInvalidTokenFn(ctx)
}
func (m *mockRefreshTokenRepository) RevokeAllByUserID(context.Context, uint) error {
	panic("unexpected RevokeAllByUserID call")
}
func (m *mockRefreshTokenRepository) RevokeAllByUserIDExcept(context.Context, uint, string) error {
	panic("unexpected RevokeAllByUserIDExcept call")
}
func (m *mockRefreshTokenRepository) RevokeByToken(context.Context, string) error {
	panic("unexpected RevokeByToken call")
}

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func TestCronServiceClearRefreshTokenDeletesInvalidTokens(t *testing.T) {
	setupLogger(t)

	called := false
	repo := &mockRefreshTokenRepository{
		deleteInvalidTokenFn: func(ctx context.Context) error {
			called = true
			require.NotNil(t, ctx)
			return nil
		},
	}

	svc := service.NewCronService(nil, repo)

	err := svc.ClearRefreshToken(context.Background())

	require.NoError(t, err)
	require.True(t, called)
}

func TestCronServiceClearRefreshTokenReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("cleanup failed")
	repo := &mockRefreshTokenRepository{
		deleteInvalidTokenFn: func(context.Context) error {
			return expectedErr
		},
	}

	svc := service.NewCronService(nil, repo)

	err := svc.ClearRefreshToken(context.Background())

	require.ErrorIs(t, err, expectedErr)
}

func TestCronServiceRunAllCleanupJobsContinuesAfterError(t *testing.T) {
	setupLogger(t)

	callCount := 0
	repo := &mockRefreshTokenRepository{
		deleteInvalidTokenFn: func(context.Context) error {
			callCount++
			return errors.New("cleanup failed")
		},
	}

	svc := service.NewCronService(nil, repo)

	err := svc.RunAllCleanupJobs(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, callCount)
}
