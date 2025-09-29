package repository

import "gorm.io/gorm"

type IRepository[T any] interface {
	Create(entity *T) error
	Update(entity *T) error
	Delete(entity *T) error
	FindById(entity *T, id any) error
}

type Repository[T any] struct {
	DB *gorm.DB
}

func (r *Repository[T]) Create(entity *T) error {
	return r.DB.Create(entity).Error
}

func (r *Repository[T]) Update(entity *T) error {
	return r.DB.Save(entity).Error
}

func (r *Repository[T]) Delete(entity *T) error {
	return r.DB.Delete(entity).Error
}

func (r *Repository[T]) FindById(entity *T, id any) error {
	return r.DB.Where("id = ?", id).Take(entity).Error
}
