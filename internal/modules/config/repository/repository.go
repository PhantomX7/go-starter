// internal/modules/config/repository/config_repository.go
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
	repository.IRepository[models.Config]
	FindByKey(ctx context.Context, key string) (*models.Config, error)
}

type configRepository struct {
	repository.Repository[models.Config]
}

func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{
		Repository: repository.Repository[models.Config]{DB: db},
	}
}

// FindByKey returns the config entry for the given key.
func (r *configRepository) FindByKey(ctx context.Context, key string) (*models.Config, error) {
	start := time.Now()

	config, err := gorm.G[models.Config](r.GetDB(ctx)).
		Where(generated.Config.Key.Eq(key)).
		First(ctx)

	r.LogSlowQuery(ctx, "FindByKey", time.Since(start), 500*time.Millisecond)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("config with key %s not found", key))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find config by key %s", key), err)
	}

	return &config, nil
}
