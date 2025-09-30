package service

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/post/dto"
	"github.com/PhantomX7/go-starter/internal/modules/post/repository"

	"github.com/jinzhu/copier"
)

type PostService interface {
	Create(ctx context.Context, req *dto.PostCreateRequest) (models.Post, error)
	Update(ctx context.Context, postId uint, req *dto.PostUpdateRequest) (models.Post, error)
	Delete(ctx context.Context, postId uint) error
	FindById(ctx context.Context, postId uint) (models.Post, error)
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
func (p *postService) Create(ctx context.Context, req *dto.PostCreateRequest) (models.Post, error) {
	var post models.Post

	err := copier.Copy(&post, &req)
	if err != nil {
		return post, err
	}

	err = p.Repository.Create(ctx, &post)
	if err != nil {
		return post, err
	}

	return post, nil
}

// Update implements PostService.
func (p *postService) Update(ctx context.Context, postId uint, req *dto.PostUpdateRequest) (models.Post, error) {
	var post models.Post
	err := p.Repository.FindById(ctx, &post, postId)
	if err != nil {
		return post, err
	}

	err = copier.Copy(&post, &req)
	if err != nil {
		return post, err
	}

	err = p.Repository.Update(ctx, &post)
	if err != nil {
		return post, err
	}

	return post, nil
}

// Delete implements PostService.
func (p *postService) Delete(ctx context.Context, postId uint) error {
	var post models.Post

	post.ID = postId

	return p.Repository.Delete(ctx, &post)
}

// FindById implements PostService.
func (p *postService) FindById(ctx context.Context, postId uint) (models.Post, error) {
	var post models.Post
	err := p.Repository.FindById(ctx, &post, postId)
	if err != nil {
		return post, err
	}
	return post, nil
}
