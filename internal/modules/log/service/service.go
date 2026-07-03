// Package service contains the business logic for audit-log queries.
package service

import (
	"context"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"gorm.io/gorm"
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . LogService

// LogService exposes read-only audit-log operations.
type LogService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.Log, response.Meta, error)
	FindByID(ctx context.Context, logID uint) (*models.Log, error)
}

type logService struct {
	logRepository repository.LogRepository
}

// NewLogService builds a LogService from the provided repository.
func NewLogService(logRepository repository.LogRepository) LogService {
	return &logService{
		logRepository: logRepository,
	}
}

// Index implements LogService.
func (s *logService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Log, response.Meta, error) {
	// Preload the acting user so the response's user field is populated.
	pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
		return db.Preload("User")
	})

	logs, err := s.logRepository.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.logRepository.Count(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	return logs, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// FindByID implements LogService.
func (s *logService) FindByID(ctx context.Context, logID uint) (*models.Log, error) {
	log, err := s.logRepository.FindByID(ctx, logID, generated.Log.User)
	if err != nil {
		return nil, err
	}

	return log, nil
}
