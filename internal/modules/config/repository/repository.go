// Package repository contains data-access code for the config module.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// ConfigRepository defines the interface for config repository operations.
type ConfigRepository interface {
	repository.Repository[models.Config]
	FindByKey(ctx context.Context, key string) (*models.Config, error)
}

type configRepository struct {
	repository.BaseRepository[models.Config]
}

// NewConfigRepository builds a ConfigRepository backed by GORM.
func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{
		BaseRepository: repository.NewBaseRepository[models.Config](db),
	}
}

// FindByKey returns the config entry for the given key.
func (r *configRepository) FindByKey(ctx context.Context, key string) (*models.Config, error) {
	start := time.Now()

	config, err := gorm.G[models.Config](r.GetDB(ctx)).
		Where(generated.Config.Key.Eq(key)).
		First(ctx)

	r.LogSlowRead(ctx, "FindByKey", time.Since(start))

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("config with key %s not found", key))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find config by key %s", key), err)
	}

	return &config, nil
}
