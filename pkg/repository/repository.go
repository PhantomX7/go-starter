package repository

import (
	"context"

	"gorm.io/gorm"
)

type IRepository[T any] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, entity *T) error
	FindById(ctx context.Context, entity *T, id any) error
}

type Repository[T any] struct {
	DB *gorm.DB
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Save(entity).Error
}

func (r *Repository[T]) Delete(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Delete(entity).Error
}

func (r *Repository[T]) FindById(ctx context.Context, entity *T, id any) error {
	return r.DB.WithContext(ctx).Where("id = ?", id).Take(entity).Error
}
