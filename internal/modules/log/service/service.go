package service

import (
	"context"
	"errors"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/log/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
)

type LogService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.Log, response.Meta, error)
	FindById(ctx context.Context, logId uint) (*models.Log, error)
}

type logService struct {
	logRepository repository.LogRepository
}

func NewLogService(logRepository repository.LogRepository) LogService {
	return &logService{
		logRepository: logRepository,
	}
}

// Index implements LogService.
func (s *logService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Log, response.Meta, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Fetching logs with pagination",
		zap.String("request_id", requestID),
		zap.Int("page", pg.GetPage()),
		zap.Int("limit", pg.Limit),
		zap.Int("offset", pg.Offset),
	)

	logs, err := s.logRepository.FindAll(ctx, pg)
	if err != nil {
		logger.Error("Failed to fetch logs",
			zap.String("request_id", requestID),
			zap.Int("page", pg.GetPage()),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	count, err := s.logRepository.Count(ctx, pg)
	if err != nil {
		logger.Error("Failed to count logs",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	logger.Info("Logs fetched successfully",
		zap.String("request_id", requestID),
		zap.Int("returned_count", len(logs)),
		zap.Int64("total_count", count),
	)

	return logs, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// FindById implements LogService.
func (s *logService) FindById(ctx context.Context, logId uint) (*models.Log, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Finding log by ID",
		zap.String("request_id", requestID),
		zap.Uint("log_id", logId),
	)

	log, err := s.logRepository.FindById(ctx, logId)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			logger.Error("Failed to find log by ID",
				zap.String("request_id", requestID),
				zap.Uint("log_id", logId),
				zap.Error(err),
			)
		}
		return log, err
	}

	logger.Debug("Found log by ID successfully",
		zap.String("request_id", requestID),
		zap.Uint("log_id", logId),
	)

	return log, nil
}
