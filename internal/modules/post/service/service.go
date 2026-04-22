// Package service contains the post business logic.
package service

import (
	"context"
	"errors"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/post/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
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
	log := logger.Ctx(ctx)
	log.Info("Fetching posts", zap.Int("page", pg.GetPage()), zap.Int("limit", pg.Limit), zap.Int("offset", pg.Offset))

	posts, err := s.postRepository.FindAll(ctx, pg)
	if err != nil {
		log.Error("Failed to fetch posts", zap.Error(err))
		return nil, response.Meta{}, err
	}

	count, err := s.postRepository.Count(ctx, pg)
	if err != nil {
		log.Error("Failed to count posts", zap.Error(err))
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
	log := logger.Ctx(ctx)
	entity := &models.Post{}

	if err := copier.Copy(entity, req); err != nil {
		log.Error("Failed to copy post payload", zap.Error(err))
		return nil, err
	}

	if err := s.postRepository.Create(ctx, entity); err != nil {
		log.Error("Failed to create post", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Update changes an existing post resource.
func (s *postService) Update(ctx context.Context, postID uint, req *dto.PostUpdateRequest) (*models.Post, error) {
	log := logger.Ctx(ctx, zap.Uint("post_id", postID))

	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		log.Error("Failed to find post for update", zap.Error(err))
		return nil, err
	}

	if err := copier.Copy(entity, req); err != nil {
		log.Error("Failed to copy post payload", zap.Error(err))
		return nil, err
	}

	if err := s.postRepository.Update(ctx, entity); err != nil {
		log.Error("Failed to update post", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Delete removes an existing post resource.
func (s *postService) Delete(ctx context.Context, postID uint) error {
	log := logger.Ctx(ctx, zap.Uint("post_id", postID))

	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		log.Error("Failed to find post for deletion", zap.Error(err))
		return err
	}

	if err := s.postRepository.Delete(ctx, entity); err != nil {
		log.Error("Failed to delete post", zap.Error(err))
		return err
	}

	return nil
}

// FindByID returns one post resource by ID.
func (s *postService) FindByID(ctx context.Context, postID uint) (*models.Post, error) {
	log := logger.Ctx(ctx, zap.Uint("post_id", postID))

	entity, err := s.postRepository.FindByID(ctx, postID)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			log.Error("Failed to find post", zap.Error(err))
		}
		return nil, err
	}

	return entity, nil
}
