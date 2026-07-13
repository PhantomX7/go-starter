// Package controller exposes cron-scheduler wiring for background jobs.
package controller

import (
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/modules/cron/service"

	"github.com/go-co-op/gocron/v2"
)

// NewCron builds the shared cron scheduler and registers recurring jobs.
// Errors are returned (not log.Fatal-ed) so the fx container can surface them
// through its normal error handling and shutdown path.
func NewCron(cronService service.CronService) (gocron.Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create cron scheduler: %w", err)
	}

	// Hourly cleanup: removes expired/revoked refresh tokens (and any future
	// cleanup jobs added to RunAllCleanupJobs). Singleton mode skips a tick
	// that fires while the previous run is still going, so a cleanup that ever
	// overruns its interval cannot run concurrently against the same tables.
	_, err = s.NewJob(
		gocron.DurationJob(1*time.Hour),
		gocron.NewTask(cronService.RunAllCleanupJobs),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		// Best effort: don't leak the scheduler we just created.
		_ = s.Shutdown()
		return nil, fmt.Errorf("failed to register cleanup cron job: %w", err)
	}

	return s, nil
}
