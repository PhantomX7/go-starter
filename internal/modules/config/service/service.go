// Package service contains the config module business logic.
package service

import (
	"context"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/config/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

// ConfigService exposes the config use cases used by handlers. The *Public
// variants back the unauthenticated surface and only see rows explicitly
// marked is_public.
//
//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . ConfigService
type ConfigService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.Config, response.Meta, error)
	PublicIndex(ctx context.Context, req *pagination.Pagination) ([]*models.Config, response.Meta, error)
	Update(ctx context.Context, configID uint, req *dto.ConfigUpdateRequest) (*models.Config, error)
	FindByKey(ctx context.Context, configKey string) (*models.Config, error)
	FindPublicByKey(ctx context.Context, configKey string) (*models.Config, error)
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

// PublicIndex implements ConfigService for the unauthenticated listing.
func (s *configService) PublicIndex(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, response.Meta, error) {
	configs, err := s.configRepository.FindAllPublic(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.configRepository.CountPublic(ctx, pg)
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
	// nil pointer = field omitted: keep the current visibility.
	if req.IsPublic != nil {
		config.IsPublic = *req.IsPublic
	}

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

// FindPublicByKey implements ConfigService: private keys are indistinguishable
// from missing ones (both not-found).
func (s *configService) FindPublicByKey(ctx context.Context, configKey string) (*models.Config, error) {
	return s.configRepository.FindPublicByKey(ctx, configKey)
}

// createLog creates an audit log entry for config operations
func (s *configService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	audit.RecordAction(ctx, s.logRepository, action, models.LogEntityTypeConfig, entityID, "config", entityName)
}
