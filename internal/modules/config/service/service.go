// Package service contains the config module business logic.
package service

import (
	"context"
	"fmt"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/config/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

// ConfigService exposes the config use cases used by handlers.
type ConfigService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.Config, response.Meta, error)
	Update(ctx context.Context, configID uint, req *dto.ConfigUpdateRequest) (*models.Config, error)
	FindByKey(ctx context.Context, configKey string) (*models.Config, error)
}

type configService struct {
	configRepository repository.ConfigRepository
	logRepository    logRepository.LogRepository
}

// NewConfigService builds a ConfigService from its dependencies.
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
	configs, err := s.configRepository.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.configRepository.Count(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	return configs, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Update implements ConfigService.
func (s *configService) Update(ctx context.Context, configID uint, req *dto.ConfigUpdateRequest) (*models.Config, error) {
	config, err := s.configRepository.FindByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	config.Value = req.Value

	if err := s.configRepository.Update(ctx, config); err != nil {
		return nil, err
	}

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, config.ID, config.Key)

	return config, nil
}

// FindByKey implements ConfigService.
func (s *configService) FindByKey(ctx context.Context, configKey string) (*models.Config, error) {
	config, err := s.configRepository.FindByKey(ctx, configKey)
	if err != nil {
		return nil, err
	}

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
