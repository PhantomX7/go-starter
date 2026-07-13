package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/modules/cron/controller"
	cronservicemocks "github.com/PhantomX7/athleton/internal/modules/cron/service/mocks"
)

func TestNewCronDoesNotOverlapCleanupRuns(t *testing.T) {
	running := make(chan struct{})
	release := make(chan struct{})
	svc := &cronservicemocks.CronServiceMock{
		RunAllCleanupJobsFunc: func(ctx context.Context) error {
			running <- struct{}{}
			<-release
			return nil
		},
	}

	scheduler, err := controller.NewCron(svc)
	require.NoError(t, err)
	t.Cleanup(func() {
		close(release)
		_ = scheduler.Shutdown()
	})

	scheduler.Start()
	jobs := scheduler.Jobs()
	require.Len(t, jobs, 1)

	require.NoError(t, jobs[0].RunNow())
	<-running // first run is now in flight and blocked

	// Trigger again while the first run is still going: a cleanup that
	// overruns its interval must be skipped, not run concurrently against the
	// same tables.
	require.NoError(t, jobs[0].RunNow())
	select {
	case <-running:
		t.Fatal("second cleanup run started while the first was still running")
	case <-time.After(300 * time.Millisecond):
		// No overlapping run started.
	}
}

func TestNewCronRegistersJobAndRunsTask(t *testing.T) {
	called := make(chan struct{}, 1)
	svc := &cronservicemocks.CronServiceMock{
		RunAllCleanupJobsFunc: func(ctx context.Context) error {
			require.NotNil(t, ctx)
			called <- struct{}{}
			return nil
		},
	}

	scheduler, err := controller.NewCron(svc)
	require.NoError(t, err)
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
