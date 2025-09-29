package service

import (
	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/post/repository"
)

type PostService interface {
	Create(post *models.Post) error
	Update(post *models.Post) error
	Delete(post *models.Post) error
	FindById(post *models.Post, id any) error
}

type postService struct {
	Repository repository.PostRepository
}

func NewPostService(Repository repository.PostRepository) PostService {
	return &postService{
		Repository: Repository,
	}
}

// Create implements PostService.
func (p *postService) Create(post *models.Post) error {
	return p.Repository.Create(post)
}

// Delete implements PostService.
func (p *postService) Delete(post *models.Post) error {
	return p.Repository.Delete(post)
}

// FindById implements PostService.
func (p *postService) FindById(post *models.Post, id any) error {
	return p.Repository.FindById(post, id)
}

// Update implements PostService.
func (p *postService) Update(post *models.Post) error {
	return p.Repository.Update(post)
}
