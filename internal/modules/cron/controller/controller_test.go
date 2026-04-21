package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/modules/cron/controller"
	cronservice "github.com/PhantomX7/athleton/internal/modules/cron/service"
)

type mockCronService struct {
	runAllCleanupJobsFn func(context.Context) error
}

func (m *mockCronService) ClearRefreshToken(context.Context) error {
	panic("unexpected ClearRefreshToken call")
}

func (m *mockCronService) RunAllCleanupJobs(ctx context.Context) error {
	if m.runAllCleanupJobsFn == nil {
		panic("unexpected RunAllCleanupJobs call")
	}
	return m.runAllCleanupJobsFn(ctx)
}

var _ cronservice.CronService = (*mockCronService)(nil)

func TestNewCronRegistersJobAndRunsTask(t *testing.T) {
	called := make(chan struct{}, 1)
	svc := &mockCronService{
		runAllCleanupJobsFn: func(ctx context.Context) error {
			require.NotNil(t, ctx)
			called <- struct{}{}
			return nil
		},
	}

	scheduler := controller.NewCron(svc)
	t.Cleanup(func() {
		_ = scheduler.Shutdown()
	})

	jobs := scheduler.Jobs()
	require.Len(t, jobs, 1)

	scheduler.Start()

	nextRun, err := jobs[0].NextRun()
	require.NoError(t, err)
	require.True(t, nextRun.After(time.Now().Add(50*time.Minute)))

	require.NoError(t, jobs[0].RunNow())

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cron task to run")
	}
}
