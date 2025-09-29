package repository

import (
	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/pkg/repository"

	"gorm.io/gorm"
)

// PostRepository defines the interface for post repository operations
type PostRepository interface {
	repository.IRepository[models.Post]
}

// postRepository implements the PostRepository interface
type postRepository struct {
	repository.Repository[models.Post]
}

// NewPostRepository creates a new instance of PostRepository
func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{
		Repository: repository.Repository[models.Post]{
			DB: db,
		},
	}
}
