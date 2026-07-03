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
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// ConfigRepository defines the interface for config repository operations.
// The *Public variants back the unauthenticated /public/config surface and
// only ever see rows explicitly marked is_public.
//
//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . ConfigRepository
type ConfigRepository interface {
	repository.Repository[models.Config]
	FindByKey(ctx context.Context, key string) (*models.Config, error)
	FindAllPublic(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, error)
	CountPublic(ctx context.Context, pg *pagination.Pagination) (int64, error)
	FindPublicByKey(ctx context.Context, key string) (*models.Config, error)
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

// FindAllPublic returns the public config rows for the unauthenticated
// listing, honoring pagination like BaseRepository.FindAll.
func (r *configRepository) FindAllPublic(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, error) {
	entities := make([]*models.Config, 0)
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Where("is_public = ?", true).
		Scopes(pg.Apply).
		Find(&entities).Error

	r.LogSlowRead(ctx, "FindAllPublic", time.Since(start))

	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to find public config records", err)
	}
	return entities, nil
}

// CountPublic returns the total public row count after the filter portion of
// pagination, mirroring BaseRepository.Count.
func (r *configRepository) CountPublic(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	var count int64
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Where("is_public = ?", true).
		Scopes(pg.ApplyWithoutMeta).
		Model(&models.Config{}).Count(&count).Error

	r.LogSlowRead(ctx, "CountPublic", time.Since(start))

	if err != nil {
		return 0, cerrors.NewInternalServerError("failed to count public config records", err)
	}
	return count, nil
}

// FindPublicByKey returns the config entry for key only when it is public.
// A private key yields the same not-found error as a missing one, so the
// public surface cannot be used to probe which keys exist.
func (r *configRepository) FindPublicByKey(ctx context.Context, key string) (*models.Config, error) {
	start := time.Now()

	config, err := gorm.G[models.Config](r.GetDB(ctx)).
		Where(generated.Config.Key.Eq(key)).
		Where(generated.Config.IsPublic.Eq(true)).
		First(ctx)

	r.LogSlowRead(ctx, "FindPublicByKey", time.Since(start))

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError(fmt.Sprintf("config with key %s not found", key))
		}
		return nil, cerrors.NewInternalServerError(fmt.Sprintf("failed to find public config by key %s", key), err)
	}

	return &config, nil
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
