package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/config/controller"
	configservice "github.com/PhantomX7/athleton/internal/modules/config/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockConfigService struct {
	indexFn     func(context.Context, *pagination.Pagination) ([]*models.Config, response.Meta, error)
	updateFn    func(context.Context, uint, *dto.ConfigUpdateRequest) (*models.Config, error)
	findByKeyFn func(context.Context, string) (*models.Config, error)
}

func (m *mockConfigService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mockConfigService) Update(ctx context.Context, configID uint, req *dto.ConfigUpdateRequest) (*models.Config, error) {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, configID, req)
}

func (m *mockConfigService) FindByKey(ctx context.Context, key string) (*models.Config, error) {
	if m.findByKeyFn == nil {
		panic("unexpected FindByKey call")
	}
	return m.findByKeyFn(ctx, key)
}

var _ configservice.ConfigService = (*mockConfigService)(nil)

func TestConfigControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockConfigService{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.Config, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 3, pg.Limit)
			require.Equal(t, 6, pg.Offset)
			require.Equal(t, "key asc", pg.Order)

			return []*models.Config{
					{Key: "site_name", Value: "Athleton"},
				}, response.Meta{
					Total:  8,
					Offset: 6,
					Limit:  3,
				}, nil
		},
	}

	ctrl := controller.NewConfigController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/config?limit=3&offset=6&sort=key+asc", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])

	meta := body["meta"].(map[string]any)
	require.Equal(t, float64(8), meta["total"])
	require.Equal(t, float64(6), meta["offset"])
	require.Equal(t, float64(3), meta["limit"])
}

func TestConfigControllerUpdateReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockConfigService{
		updateFn: func(ctx context.Context, configID uint, req *dto.ConfigUpdateRequest) (*models.Config, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(5), configID)
			require.Equal(t, "New Value", req.Value)
			return &models.Config{
				Key:   "site_name",
				Value: "New Value",
			}, nil
		},
	}

	ctrl := controller.NewConfigController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/config/5",
		bytes.NewBufferString(`{"value":"New Value"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Config updated successfully", body["message"])

	data := body["data"].(map[string]any)
	require.Equal(t, "site_name", data["key"])
	require.Equal(t, "New Value", data["value"])
}

func TestConfigControllerUpdateRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockConfigService{
		updateFn: func(context.Context, uint, *dto.ConfigUpdateRequest) (*models.Config, error) {
			t.Fatal("Update should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.NewConfigController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/config/not-a-number", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "not-a-number"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}

func TestConfigControllerFindByKeyReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockConfigService{
		findByKeyFn: func(ctx context.Context, key string) (*models.Config, error) {
			require.NotNil(t, ctx)
			require.Equal(t, "timezone", key)
			return &models.Config{
				Key:   "timezone",
				Value: "Asia/Jakarta",
			}, nil
		},
	}

	ctrl := controller.NewConfigController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/config/key/timezone", nil)
	ctx.Params = gin.Params{{Key: "key", Value: "timezone"}}

	ctrl.FindByKey(ctx)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Config found successfully", body["message"])
}

func TestConfigControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockConfigService{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.Config, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewConfigController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/config", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
