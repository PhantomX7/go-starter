package service

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/post/dto"
	"github.com/PhantomX7/go-starter/internal/modules/post/repository"
	"github.com/PhantomX7/go-starter/libs/transaction_manager"
	"github.com/PhantomX7/go-starter/pkg/errors"
	"github.com/PhantomX7/go-starter/pkg/pagination"
	"github.com/PhantomX7/go-starter/pkg/response"

	"github.com/jinzhu/copier"
)

type PostService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]models.Post, response.Meta, error)
	Create(ctx context.Context, req *dto.PostCreateRequest) (models.Post, error)
	Update(ctx context.Context, postId uint, req *dto.PostUpdateRequest) (models.Post, error)
	Delete(ctx context.Context, postId uint) error
	FindById(ctx context.Context, postId uint) (models.Post, error)
}

type postService struct {
	postRepository repository.PostRepository
	transactionManager transaction_manager.TransactionManager
}

func NewPostService(Repository repository.PostRepository, tm transaction_manager.TransactionManager) PostService {
	return &postService{
		postRepository: Repository,
		transactionManager: tm,
	}
}

// Index implements PostService.
func (p *postService) Index(ctx context.Context, pg *pagination.Pagination) ([]models.Post, response.Meta, error) {
	posts, err := p.postRepository.FindAll(ctx, pg)
	if err != nil {
		return posts, response.Meta{}, err
	}

	count, err := p.postRepository.Count(ctx, pg)
	if err != nil {
		return posts, response.Meta{}, err
	}

	return posts, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements PostService.
func (p *postService) Create(ctx context.Context, req *dto.PostCreateRequest) (models.Post, error) {
	var post models.Post

	err := copier.Copy(&post, &req)
	if err != nil {
		return post, err
	}

	err = p.transactionManager.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		err = p.postRepository.Create(ctx, &post)
		if err != nil {
			return err
		}
		
		post.Description = "aaaaa"
		err = p.postRepository.Update(ctx, &post)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return post, errors.NewInternalServerError("failed to create post", err)
	}

	return post, nil
}

// Update implements PostService.
func (p *postService) Update(ctx context.Context, postId uint, req *dto.PostUpdateRequest) (models.Post, error) {
	post, err := p.postRepository.FindById(ctx, postId)
	if err != nil {
		return post, err
	}

	err = copier.Copy(&post, &req)
	if err != nil {
		return post, err
	}

	err = p.postRepository.Update(ctx, &post)
	if err != nil {
		return post, err
	}

	return post, nil
}

// Delete implements PostService.
func (p *postService) Delete(ctx context.Context, postId uint) error {
	var post models.Post

	post.ID = postId

	return p.postRepository.Delete(ctx, &post)
}

// FindById implements PostService.
func (p *postService) FindById(ctx context.Context, postId uint) (models.Post, error) {
	post, err := p.postRepository.FindById(ctx, postId)
	if err != nil {
		return post, err
	}
	return post, nil
}
