// Package service contains the cron module business logic.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	"github.com/PhantomX7/athleton/pkg/logger"

	"go.uber.org/zap"
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . CronService

// CronService exposes the background cleanup jobs run by the scheduler.
type CronService interface {
	ClearRefreshToken(ctx context.Context) error
	RunAllCleanupJobs(ctx context.Context) error
}

type cronService struct {
	refreshTokenRepo repository.RefreshTokenRepository
}

// NewCronService builds a CronService from its dependencies.
func NewCronService(
	refreshTokenRepo repository.RefreshTokenRepository,
) CronService {
	return &cronService{
		refreshTokenRepo: refreshTokenRepo,
	}
}

// ClearRefreshToken removes expired and revoked refresh tokens
func (s *cronService) ClearRefreshToken(ctx context.Context) error {
	startTime := time.Now()
	logger.Info("Starting refresh token cleanup job")

	err := s.refreshTokenRepo.DeleteInvalidToken(ctx)
	if err != nil {
		logger.Error("Failed to clear invalid refresh tokens",
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)
		return err
	}

	logger.Info("Refresh token cleanup job completed successfully",
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// RunAllCleanupJobs runs all cleanup jobs in sequence. A failing job does not
// stop the remaining jobs, but every failure is joined into the returned
// error so the scheduler observes the run's real outcome.
func (s *cronService) RunAllCleanupJobs(ctx context.Context) error {
	startTime := time.Now()
	logger.Info("Starting all cleanup jobs")

	var errs []error

	// Run cleanup for invalid tokens
	if err := s.ClearRefreshToken(ctx); err != nil {
		logger.Error("Invalid token cleanup failed", zap.Error(err))
		// Continue to the next cleanup even if this one fails.
		errs = append(errs, err)
	}

	logger.Info("All cleanup jobs completed",
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return errors.Join(errs...)
}
