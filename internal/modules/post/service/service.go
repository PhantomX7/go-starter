// Package service contains the post business logic.
package service

import (
	"context"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/post/repository"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

// PostService defines the business operations for post resources.
type PostService interface {
	Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Post, response.Meta, error)
	Create(ctx context.Context, req *dto.PostCreateRequest) (*models.Post, error)
	Update(ctx context.Context, postID uint, req *dto.PostUpdateRequest) (*models.Post, error)
	Delete(ctx context.Context, postID uint) error
	FindByID(ctx context.Context, postID uint) (*models.Post, error)
}

type postService struct {
	postRepository repository.PostRepository
}

// NewPostService constructs a PostService.
func NewPostService(postRepository repository.PostRepository) PostService {
	return &postService{
		postRepository: postRepository,
	}
}

// Index returns a paginated collection of post resources.
func (s *postService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Post, response.Meta, error) {
	posts, err := s.postRepository.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.postRepository.Count(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	return posts, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create persists a new post resource.
func (s *postService) Create(ctx context.Context, req *dto.PostCreateRequest) (*models.Post, error) {
	entity := &models.Post{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.postRepository.Create(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Update changes an existing post resource. Fields are pointers in the request
// so an omitted field (nil) keeps its current value — PATCH semantics.
func (s *postService) Update(ctx context.Context, postID uint, req *dto.PostUpdateRequest) (*models.Post, error) {
	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		entity.Name = *req.Name
	}
	if req.Description != nil {
		entity.Description = *req.Description
	}

	if err := s.postRepository.Update(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes an existing post resource.
func (s *postService) Delete(ctx context.Context, postID uint) error {
	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		return err
	}

	return s.postRepository.Delete(ctx, entity)
}

// FindByID returns one post resource by ID.
func (s *postService) FindByID(ctx context.Context, postID uint) (*models.Post, error) {
	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	return entity, nil
}
