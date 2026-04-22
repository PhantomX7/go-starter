// Package repository provides post persistence primitives.
package repository

import (
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// PostRepository defines the persistence operations for post resources.
type PostRepository interface {
	repository.Repository[models.Post]
}

type postRepository struct {
	repository.BaseRepository[models.Post]
}

// NewPostRepository constructs a PostRepository.
func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{
		BaseRepository: repository.NewBaseRepository[models.Post](db),
	}
}
