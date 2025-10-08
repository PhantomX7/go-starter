package repository

import (
	"context"
	"fmt"

	"github.com/PhantomX7/go-starter/pkg/errors"
	"github.com/PhantomX7/go-starter/pkg/pagination"

	"gorm.io/gorm"
)

type IRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
	FindById(ctx context.Context, id uint) (T, error)
	FindAll(ctx context.Context, pg *pagination.Pagination) ([]T, error)
	Count(ctx context.Context, pg *pagination.Pagination) (int64, error)
}

type Repository[T any] struct {
	DB *gorm.DB
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	err := r.DB.WithContext(ctx).Create(entity).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to create %T record", *entity)
		return errors.NewInternalServerError(errMessage, err)
	}
	return nil
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	err := r.DB.WithContext(ctx).Save(entity).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to update %T record", *entity)
		return errors.NewInternalServerError(errMessage, err)
	}
	return nil
}

func (r *Repository[T]) Delete(ctx context.Context, entity *T) error {
	err := r.DB.WithContext(ctx).Delete(entity).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to delete %T record", *entity)
		return errors.NewInternalServerError(errMessage, err)
	}
	return nil
}

func (r *Repository[T]) FindAll(ctx context.Context, pg *pagination.Pagination) ([]T, error) {
	entities := make([]T, 0)

	err := r.DB.WithContext(ctx).
		Scopes(pg.Apply).
		Find(&entities).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to find %T records", *new(T))
		return nil, errors.NewInternalServerError(errMessage, err)
	}

	return entities, nil
}

func (r *Repository[T]) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	var count int64

	err := r.DB.WithContext(ctx).
		Scopes(pg.ApplyWithoutMeta).
		Model(new(T)).Count(&count).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to count %T records", *new(T))
		return 0, errors.NewInternalServerError(errMessage, err)
	}
	return count, nil
}

func (r *Repository[T]) FindById(ctx context.Context, id uint) (T, error) {
	var entity T
	err := r.DB.WithContext(ctx).Where("id = ?", id).Take(&entity).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to find %T record by id %v", *new(T), id)
		return entity, errors.NewInternalServerError(errMessage, err)
	}
	return entity, nil
}
