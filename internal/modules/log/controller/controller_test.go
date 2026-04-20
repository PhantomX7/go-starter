package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/log/controller"
	logservice "github.com/PhantomX7/athleton/internal/modules/log/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockLogService struct {
	indexFn    func(context.Context, *pagination.Pagination) ([]*models.Log, response.Meta, error)
	findByIDFn func(context.Context, uint) (*models.Log, error)
}

func (m *mockLogService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Log, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mockLogService) FindById(ctx context.Context, logID uint) (*models.Log, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, logID)
}

var _ logservice.LogService = (*mockLogService)(nil)

func TestLogControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	svc := &mockLogService{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.Log, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 5, pg.Limit)
			require.Equal(t, 10, pg.Offset)
			require.Equal(t, "created_at asc", pg.Order)

			return []*models.Log{
					{
						ID:         1,
						Action:     models.LogActionCreate,
						EntityType: models.LogEntityTypeUser,
						EntityID:   11,
						Message:    "created user",
						Timestamp:  models.Timestamp{CreatedAt: createdAt},
					},
				}, response.Meta{
					Total:  23,
					Offset: 10,
					Limit:  5,
				}, nil
		},
	}

	ctrl := controller.NewLogController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/logs?limit=5&offset=10&sort=created_at+asc", nil)
	ctx.Request = req

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])

	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	item, ok := data[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), item["id"])
	require.Equal(t, "create", item["action"])
	require.Equal(t, "user", item["entity_type"])
	require.Equal(t, "created user", item["message"])

	meta, ok := body["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(23), meta["total"])
	require.Equal(t, float64(10), meta["offset"])
	require.Equal(t, float64(5), meta["limit"])
}

func TestLogControllerFindByIDReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLogService{
		findByIDFn: func(ctx context.Context, logID uint) (*models.Log, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(7), logID)
			return &models.Log{
				ID:         7,
				Action:     models.LogActionDelete,
				EntityType: models.LogEntityTypeConfig,
				EntityID:   3,
				Message:    "deleted config",
			}, nil
		},
	}

	ctrl := controller.NewLogController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/log/7", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}

	ctrl.FindById(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Log found successfully", body["message"])

	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(7), data["id"])
	require.Equal(t, "delete", data["action"])
	require.Equal(t, "config", data["entity_type"])
}

func TestLogControllerFindByIDRejectsInvalidParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLogService{
		findByIDFn: func(context.Context, uint) (*models.Log, error) {
			t.Fatal("FindById should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.NewLogController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/log/not-a-number", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "not-a-number"}}

	ctrl.FindById(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.Error(t, ctx.Errors[0].Err)
	require.Equal(t, 0, rec.Body.Len())
}

func TestLogControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockLogService{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.Log, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewLogController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	ctx.Request = req

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
	require.Equal(t, 0, rec.Body.Len())
}
