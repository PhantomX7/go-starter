// Package controller exposes cron-scheduler wiring for background jobs.
package controller

import (
	"log"
	"time"

	"github.com/PhantomX7/athleton/internal/modules/cron/service"

	"github.com/go-co-op/gocron/v2"
)

// NewCron builds the shared cron scheduler and registers recurring jobs.
func NewCron(cronService service.CronService) gocron.Scheduler {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal("error creating cron scheduler", err)
	}

	// add a job to the scheduler to clear refresh tokens
	_, err = s.NewJob(
		gocron.DurationJob(1*time.Hour),
		gocron.NewTask(cronService.RunAllCleanupJobs),
	)
	if err != nil {
		log.Fatal("error creating cron job for clear refresh token", err)
	}

	return s
}
