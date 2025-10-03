package repository

import (
	"context"
	"fmt"

	"github.com/PhantomX7/go-starter/pkg/errors"

	"gorm.io/gorm"
)

type IRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
	FindById(ctx context.Context, entity *T, id uint) error
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

func (r *Repository[T]) FindById(ctx context.Context, entity *T, id uint) error {
	err := r.DB.WithContext(ctx).Where("id = ?", id).Take(entity).Error
	if err != nil {
		errMessage := fmt.Sprintf("failed to find %T record by id %v", *entity, id)
		return errors.NewInternalServerError(errMessage, err)
	}
	return nil
}
