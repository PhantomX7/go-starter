package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/post/controller"
	postservice "github.com/PhantomX7/athleton/internal/modules/post/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

type mockPostService struct {
	indexFn    func(context.Context, *pagination.Pagination) ([]*models.Post, response.Meta, error)
	createFn   func(context.Context, *dto.PostCreateRequest) (*models.Post, error)
	updateFn   func(context.Context, uint, *dto.PostUpdateRequest) (*models.Post, error)
	deleteFn   func(context.Context, uint) error
	findByIDFn func(context.Context, uint) (*models.Post, error)
}

func (m *mockPostService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Post, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mockPostService) Create(ctx context.Context, req *dto.PostCreateRequest) (*models.Post, error) {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, req)
}

func (m *mockPostService) Update(ctx context.Context, postID uint, req *dto.PostUpdateRequest) (*models.Post, error) {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, postID, req)
}

func (m *mockPostService) Delete(ctx context.Context, postID uint) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, postID)
}

func (m *mockPostService) FindByID(ctx context.Context, postID uint) (*models.Post, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, postID)
}

var _ postservice.PostService = (*mockPostService)(nil)

func TestPostControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.Post, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 2, pg.Limit)
			require.Equal(t, 4, pg.Offset)
			return []*models.Post{
					{Name: "first"},
				}, response.Meta{
					Total:  7,
					Offset: 4,
					Limit:  2,
				}, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/post?limit=2&offset=4", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])
}

func TestPostControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockPostService{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.Post, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/post", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestPostControllerCreateReturnsCreatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		createFn: func(ctx context.Context, req *dto.PostCreateRequest) (*models.Post, error) {
			require.Equal(t, "New Post", req.Name)
			require.Equal(t, "content", req.Description)
			created := &models.Post{Name: req.Name, Description: req.Description}
			created.ID = 3
			return created, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/post",
		strings.NewReader(`{"name":"New Post","description":"content"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrl.Create(ctx)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Post created successfully", body["message"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), data["id"])
}

func TestPostControllerCreateRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		createFn: func(context.Context, *dto.PostCreateRequest) (*models.Post, error) {
			t.Fatal("Create should not be called for invalid payloads")
			return nil, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/post",
		strings.NewReader(`{"description":"missing required name"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrl.Create(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypeBind))
}

func TestPostControllerUpdateReturnsUpdatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		updateFn: func(ctx context.Context, postID uint, req *dto.PostUpdateRequest) (*models.Post, error) {
			require.Equal(t, uint(5), postID)
			require.Equal(t, "Renamed", req.Name)
			updated := &models.Post{Name: req.Name}
			updated.ID = postID
			return updated, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/post/5",
		strings.NewReader(`{"name":"Renamed"}`))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Post updated successfully", body["message"])
}

func TestPostControllerUpdateRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		updateFn: func(context.Context, uint, *dto.PostUpdateRequest) (*models.Post, error) {
			t.Fatal("Update should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/post/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}

func TestPostControllerDeleteReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		deleteFn: func(ctx context.Context, postID uint) error {
			require.Equal(t, uint(8), postID)
			return nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/post/8", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "8"}}

	ctrl.Delete(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Post deleted successfully", body["message"])
}

func TestPostControllerFindByIDReturnsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockPostService{
		findByIDFn: func(ctx context.Context, postID uint) (*models.Post, error) {
			require.Equal(t, uint(2), postID)
			found := &models.Post{Name: "Found"}
			found.ID = 2
			return found, nil
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/post/2", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "2"}}

	ctrl.FindByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Post found successfully", body["message"])
}

func TestPostControllerFindByIDPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("not found")
	svc := &mockPostService{
		findByIDFn: func(context.Context, uint) (*models.Post, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewPostController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/post/2", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "2"}}

	ctrl.FindByID(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
