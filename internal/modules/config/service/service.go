package service

import (
	"context"
	"fmt"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/config/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type ConfigService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.Config, response.Meta, error)
	Update(ctx context.Context, configId uint, req *dto.ConfigUpdateRequest) (*models.Config, error)
	FindByKey(ctx context.Context, configKey string) (*models.Config, error)
}

type configService struct {
	configRepository repository.ConfigRepository
	logRepository    logRepository.LogRepository
}

func NewConfigService(
	configRepository repository.ConfigRepository,
	logRepository logRepository.LogRepository,
) ConfigService {
	return &configService{
		configRepository: configRepository,
		logRepository:    logRepository,
	}
}

// Index implements ConfigService.
func (s *configService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, response.Meta, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Fetching configs with pagination",
		zap.String("request_id", requestID),
		zap.Int("page", pg.GetPage()),
		zap.Int("limit", pg.Limit),
		zap.Int("offset", pg.Offset),
	)

	configs, err := s.configRepository.FindAll(ctx, pg)
	if err != nil {
		logger.Error("Failed to fetch configs",
			zap.String("request_id", requestID),
			zap.Int("page", pg.GetPage()),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	count, err := s.configRepository.Count(ctx, pg)
	if err != nil {
		logger.Error("Failed to count configs",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	logger.Info("Configs fetched successfully",
		zap.String("request_id", requestID),
		zap.Int("returned_count", len(configs)),
		zap.Int64("total_count", count),
	)

	return configs, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Update implements ConfigService.
func (s *configService) Update(ctx context.Context, configId uint, req *dto.ConfigUpdateRequest) (*models.Config, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Updating config",
		zap.String("request_id", requestID),
		zap.Uint("config_id", configId),
	)

	config, err := s.configRepository.FindById(ctx, configId)
	if err != nil {
		logger.Error("Failed to find config for update",
			zap.String("request_id", requestID),
			zap.Uint("config_id", configId),
			zap.Error(err),
		)
		return nil, err
	}

	err = copier.Copy(&config, &req)
	if err != nil {
		logger.Error("Failed to copy config data",
			zap.String("request_id", requestID),
			zap.Uint("config_id", configId),
			zap.Error(err),
		)
		return nil, err
	}

	err = s.configRepository.Update(ctx, config)
	if err != nil {
		logger.Error("Failed to update config",
			zap.String("request_id", requestID),
			zap.Uint("config_id", configId),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Info("Config updated successfully",
		zap.String("request_id", requestID),
		zap.Uint("config_id", configId),
	)

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, config.ID, config.Key)

	return config, nil
}

// FindByKey implements ConfigService.
func (s *configService) FindByKey(ctx context.Context, configKey string) (*models.Config, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Finding config by key",
		zap.String("request_id", requestID),
		zap.String("config_key", configKey),
	)

	config, err := s.configRepository.FindByKey(ctx, configKey)
	if err != nil {
		logger.Error("Failed to find config by key",
			zap.String("request_id", requestID),
			zap.String("config_key", configKey),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Debug("Found config by key successfully",
		zap.String("request_id", requestID),
		zap.String("config_key", configKey),
		zap.Uint("config_id", config.ID),
	)

	return config, nil
}

// createLog creates an audit log entry for config operations
func (s *configService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	userName := audit.UserName(ctx)
	var message string
	switch action {
	case models.LogActionUpdate:
		message = fmt.Sprintf("%s updated config: %s", userName, entityName)
	default:
		message = fmt.Sprintf("%s performed %s on config: %s", userName, action, entityName)
	}

	audit.Record(ctx, s.logRepository, audit.Entry{
		Action:     action,
		EntityType: models.LogEntityTypeConfig,
		EntityID:   entityID,
		Message:    message,
	})
}
