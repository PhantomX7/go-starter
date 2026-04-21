// Package service contains the cron module business logic.
package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	"github.com/PhantomX7/athleton/pkg/logger"

	"go.uber.org/zap"
)

// CronService exposes the background cleanup jobs run by the scheduler.
type CronService interface {
	ClearRefreshToken(ctx context.Context) error
	RunAllCleanupJobs(ctx context.Context) error
}

type cronService struct {
	db               *gorm.DB
	refreshTokenRepo repository.RefreshTokenRepository
}

// NewCronService builds a CronService from its dependencies.
func NewCronService(
	db *gorm.DB,
	refreshTokenRepo repository.RefreshTokenRepository,
) CronService {
	return &cronService{
		db:               db,
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

// RunAllCleanupJobs runs all cleanup jobs in sequence
func (s *cronService) RunAllCleanupJobs(ctx context.Context) error {
	startTime := time.Now()
	logger.Info("Starting all cleanup jobs")

	// Run cleanup for invalid tokens
	if err := s.ClearRefreshToken(ctx); err != nil {
		logger.Error("Invalid token cleanup failed", zap.Error(err))
		// Continue to next cleanup even if this fails
	}

	logger.Info("All cleanup jobs completed",
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return nil
}
