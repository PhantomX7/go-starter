// internal/modules/config/repository/config_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// ConfigRepository defines the interface for config repository operations
type ConfigRepository interface {
	repository.IRepository[models.Config]
	FindByKey(ctx context.Context, key string) (*models.Config, error)
}

// configRepository implements the ConfigRepository interface
type configRepository struct {
	repository.Repository[models.Config]
}

// NewConfigRepository creates a new instance of ConfigRepository
func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{
		Repository: repository.Repository[models.Config]{
			DB: db,
		},
	}
}

// FindByKey finds a config by its key with optional preloads
func (r *configRepository) FindByKey(ctx context.Context, key string) (*models.Config, error) {
	start := time.Now()

	var config models.Config

	db := r.GetDB(ctx).WithContext(ctx)

	err := db.Where("key = ?", key).Take(&config).Error
	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("config with key %s not found", key))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find config by key %s", key), err)
	}

	r.LogSlowQuery(ctx, "FindByKey", duration, 500*time.Millisecond)

	return &config, nil
}
