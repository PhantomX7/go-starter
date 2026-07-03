package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/modules/cron/service"
	refreshtokenmocks "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository/mocks"
	"github.com/PhantomX7/athleton/pkg/logger"
)

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
	repo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		DeleteInvalidTokenFunc: func(ctx context.Context) error {
			called = true
			require.NotNil(t, ctx)
			return nil
		},
	}

	svc := service.NewCronService(repo)

	err := svc.ClearRefreshToken(context.Background())

	require.NoError(t, err)
	require.True(t, called)
}

func TestCronServiceClearRefreshTokenReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("cleanup failed")
	repo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		DeleteInvalidTokenFunc: func(context.Context) error {
			return expectedErr
		},
	}

	svc := service.NewCronService(repo)

	err := svc.ClearRefreshToken(context.Background())

	require.ErrorIs(t, err, expectedErr)
}

func TestCronServiceRunAllCleanupJobsReportsJobFailures(t *testing.T) {
	setupLogger(t)

	// Jobs keep running after an individual failure, but the aggregate error
	// must surface so the scheduler (and any job monitoring) observes it
	// instead of a false success.
	callCount := 0
	jobErr := errors.New("cleanup failed")
	repo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		DeleteInvalidTokenFunc: func(context.Context) error {
			callCount++
			return jobErr
		},
	}

	svc := service.NewCronService(repo)

	err := svc.RunAllCleanupJobs(context.Background())

	require.ErrorIs(t, err, jobErr)
	require.Equal(t, 1, callCount)
}

func TestCronServiceRunAllCleanupJobsSucceedsWhenJobsSucceed(t *testing.T) {
	setupLogger(t)

	repo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		DeleteInvalidTokenFunc: func(context.Context) error { return nil },
	}

	svc := service.NewCronService(repo)

	require.NoError(t, svc.RunAllCleanupJobs(context.Background()))
}
