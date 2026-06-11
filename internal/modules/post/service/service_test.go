package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	postrepository "github.com/PhantomX7/athleton/internal/modules/post/repository"
	"github.com/PhantomX7/athleton/internal/modules/post/service"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
)

type mockPostRepository struct {
	createFn   func(context.Context, *models.Post) error
	updateFn   func(context.Context, *models.Post) error
	deleteFn   func(context.Context, *models.Post) error
	findByIDFn func(context.Context, uint, ...repository.Association) (*models.Post, error)
	findAllFn  func(context.Context, *pagination.Pagination) ([]*models.Post, error)
	countFn    func(context.Context, *pagination.Pagination) (int64, error)
}

func (m *mockPostRepository) Create(ctx context.Context, entity *models.Post) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}

func (m *mockPostRepository) Update(ctx context.Context, entity *models.Post) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}

func (m *mockPostRepository) Delete(ctx context.Context, entity *models.Post) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, entity)
}

func (m *mockPostRepository) FindByID(ctx context.Context, id uint, preloads ...repository.Association) (*models.Post, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mockPostRepository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.Post, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mockPostRepository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
}

var _ postrepository.PostRepository = (*mockPostRepository)(nil)

func strPtr(s string) *string {
	return &s
}

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func TestPostServiceIndexReturnsPostsAndMeta(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(map[string][]string{"limit": {"2"}}, nil, pagination.PaginationOptions{})
	repo := &mockPostRepository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.Post, error) {
			require.Same(t, pg, gotPg)
			return []*models.Post{
				{Name: "first"},
				{Name: "second"},
			}, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Same(t, pg, gotPg)
			return 5, nil
		},
	}

	svc := service.NewPostService(repo)

	posts, meta, err := svc.Index(context.Background(), pg)

	require.NoError(t, err)
	require.Len(t, posts, 2)
	require.Equal(t, int64(5), meta.Total)
	require.Equal(t, pg.Limit, meta.Limit)
	require.Equal(t, pg.Offset, meta.Offset)
}

func TestPostServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mockPostRepository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.Post, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewPostService(repo)

	posts, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, posts)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}

func TestPostServiceCreateCopiesRequestAndPersists(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		createFn: func(ctx context.Context, post *models.Post) error {
			require.Equal(t, "New Post", post.Name)
			require.Equal(t, "fresh content", post.Description)
			post.ID = 9
			return nil
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.Create(context.Background(), &dto.PostCreateRequest{
		Name:        "New Post",
		Description: "fresh content",
	})

	require.NoError(t, err)
	require.NotNil(t, post)
	require.Equal(t, uint(9), post.ID)
}

func TestPostServiceCreateReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("insert failed")
	repo := &mockPostRepository{
		createFn: func(context.Context, *models.Post) error {
			return expectedErr
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.Create(context.Background(), &dto.PostCreateRequest{Name: "New Post"})

	require.Nil(t, post)
	require.ErrorIs(t, err, expectedErr)
}

func TestPostServiceUpdateAppliesChanges(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Post, error) {
			require.Equal(t, uint(4), id)
			existing := &models.Post{Name: "Old", Description: "old text"}
			existing.ID = 4
			return existing, nil
		},
		updateFn: func(ctx context.Context, post *models.Post) error {
			require.Equal(t, uint(4), post.ID)
			require.Equal(t, "Updated", post.Name)
			// Omitted fields (nil pointers) must keep their current value.
			require.Equal(t, "old text", post.Description)
			return nil
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.Update(context.Background(), 4, &dto.PostUpdateRequest{Name: strPtr("Updated")})

	require.NoError(t, err)
	require.NotNil(t, post)
	require.Equal(t, "Updated", post.Name)
	require.Equal(t, "old text", post.Description)
}

func TestPostServiceUpdatePropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.Post, error) {
			return nil, cerrors.NewNotFoundError("post not found")
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.Update(context.Background(), 4, &dto.PostUpdateRequest{Name: strPtr("Updated")})

	require.Nil(t, post)
	require.ErrorIs(t, err, cerrors.ErrNotFound)
}

func TestPostServiceDeleteRemovesPost(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Post, error) {
			existing := &models.Post{Name: "Doomed"}
			existing.ID = id
			return existing, nil
		},
		deleteFn: func(ctx context.Context, post *models.Post) error {
			require.Equal(t, uint(6), post.ID)
			return nil
		},
	}

	svc := service.NewPostService(repo)

	require.NoError(t, svc.Delete(context.Background(), 6))
}

func TestPostServiceDeletePropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.Post, error) {
			return nil, cerrors.NewNotFoundError("post not found")
		},
	}

	svc := service.NewPostService(repo)

	err := svc.Delete(context.Background(), 6)

	require.ErrorIs(t, err, cerrors.ErrNotFound)
}

func TestPostServiceFindByIDReturnsPost(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.Post, error) {
			require.Equal(t, uint(2), id)
			existing := &models.Post{Name: "Found"}
			existing.ID = 2
			return existing, nil
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.FindByID(context.Background(), 2)

	require.NoError(t, err)
	require.Equal(t, "Found", post.Name)
}

func TestPostServiceFindByIDPropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mockPostRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.Post, error) {
			return nil, cerrors.NewNotFoundError("post not found")
		},
	}

	svc := service.NewPostService(repo)

	post, err := svc.FindByID(context.Background(), 2)

	require.Nil(t, post)
	require.ErrorIs(t, err, cerrors.ErrNotFound)
}
