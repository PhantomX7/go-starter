package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
	FindById(ctx context.Context, id uint, preloads ...string) (*T, error)
	FindAll(ctx context.Context, pg *pagination.Pagination) ([]*T, error)
	Count(ctx context.Context, pg *pagination.Pagination) (int64, error)
}

type Repository[T any] struct {
	DB *gorm.DB
}

// GetDB checks context for transaction, falls back to default DB
func (r *Repository[T]) GetDB(ctx context.Context) *gorm.DB {
	if tx := utils.GetTxFromContext(ctx); tx != nil {
		return tx
	}
	return r.DB
}

// getEntityType returns the type name for logging
func (r *Repository[T]) getEntityType() string {
	return fmt.Sprintf("%T", *new(T))
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	start := time.Now()
	err := r.GetDB(ctx).WithContext(ctx).Create(entity).Error
	duration := time.Since(start)

	if err != nil {
		errMessage := fmt.Sprintf("failed to create %s record", r.getEntityType())
		return cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "Create", duration, 500*time.Millisecond)
	return nil
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	start := time.Now()
	err := r.GetDB(ctx).WithContext(ctx).Save(entity).Error
	duration := time.Since(start)

	if err != nil {
		errMessage := fmt.Sprintf("failed to update %s record", r.getEntityType())
		return cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "Update", duration, 500*time.Millisecond)
	return nil
}

func (r *Repository[T]) Delete(ctx context.Context, entity *T) error {
	start := time.Now()
	err := r.GetDB(ctx).WithContext(ctx).Delete(entity).Error
	duration := time.Since(start)

	if err != nil {
		errMessage := fmt.Sprintf("failed to delete %s record", r.getEntityType())
		return cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "Delete", duration, 500*time.Millisecond)
	return nil
}

func (r *Repository[T]) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*T, error) {
	entities := make([]*T, 0)
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Scopes(pg.Apply).
		Find(&entities).Error

	duration := time.Since(start)

	if err != nil {
		errMessage := fmt.Sprintf("failed to find %s records", r.getEntityType())
		return nil, cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "FindAll", duration, 1*time.Second)
	return entities, nil
}

func (r *Repository[T]) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	var count int64
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Scopes(pg.ApplyWithoutMeta).
		Model(new(T)).Count(&count).Error

	duration := time.Since(start)

	if err != nil {
		errMessage := fmt.Sprintf("failed to count %s records", r.getEntityType())
		return 0, cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "Count", duration, 1*time.Second)
	return count, nil
}

func (r *Repository[T]) FindById(ctx context.Context, id uint, preloads ...string) (*T, error) {
	var entity T
	start := time.Now()

	db := r.GetDB(ctx)
	db = r.ApplyPreloads(db, preloads...)
	err := db.WithContext(ctx).Where("id = ?", id).Take(&entity).Error

	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMessage := fmt.Sprintf("%s record with id %v not found", r.getEntityType(), id)
			return &entity, cerrors.NewNotFoundError(errMessage)
		}
		errMessage := fmt.Sprintf("failed to find %s record by id %v", r.getEntityType(), id)
		return &entity, cerrors.NewInternalServerError(errMessage, err)
	}

	r.LogSlowQuery(ctx, "FindById", duration, 500*time.Millisecond)
	return &entity, nil
}

func (r Repository[T]) ApplyPreloads(db *gorm.DB, preloads ...string) *gorm.DB {
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	return db
}

// LogSlowQuery logs queries that exceed threshold
func (r *Repository[T]) LogSlowQuery(ctx context.Context, operation string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		logger.Warn("Slow query detected",
			zap.String("request_id", utils.GetRequestIDFromContext(ctx)),
			zap.String("entity_type", r.getEntityType()),
			zap.String("operation", operation),
			zap.Duration("duration", duration),
			zap.Duration("threshold", threshold),
		)
	}
}
