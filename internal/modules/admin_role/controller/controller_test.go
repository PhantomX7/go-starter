package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	adminroleservice "github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockAdminRoleService struct {
	indexFn             func(context.Context, *pagination.Pagination) ([]*models.AdminRole, response.Meta, error)
	deleteFn            func(context.Context, uint) error
	findByIDFn          func(context.Context, uint) (*models.AdminRole, error)
	getAllPermissionsFn func(context.Context) map[string][]map[string]string
}

func (m *mockAdminRoleService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mockAdminRoleService) Create(context.Context, *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
	panic("unexpected Create call")
}

func (m *mockAdminRoleService) Update(context.Context, uint, *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
	panic("unexpected Update call")
}

func (m *mockAdminRoleService) Delete(ctx context.Context, roleID uint) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, roleID)
}

func (m *mockAdminRoleService) FindById(ctx context.Context, roleID uint) (*models.AdminRole, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, roleID)
}

func (m *mockAdminRoleService) GetAllPermissions(ctx context.Context) map[string][]map[string]string {
	if m.getAllPermissionsFn == nil {
		panic("unexpected GetAllPermissions call")
	}
	return m.getAllPermissionsFn(ctx)
}

var _ adminroleservice.AdminRoleService = (*mockAdminRoleService)(nil)

func TestAdminRoleControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAdminRoleService{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 2, pg.Limit)
			require.Equal(t, 4, pg.Offset)
			require.Equal(t, "name asc", pg.Order)
			return []*models.AdminRole{
					{ID: 1, Name: "Manager", IsActive: true},
				}, response.Meta{
					Total:  7,
					Offset: 4,
					Limit:  2,
				}, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/admin-role?limit=2&offset=4&sort=name+asc", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])
}

func TestAdminRoleControllerDeleteRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAdminRoleService{
		deleteFn: func(context.Context, uint) error {
			t.Fatal("Delete should not be called for invalid ids")
			return nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/admin/admin-role/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.Delete(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}

func TestAdminRoleControllerFindByIDReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAdminRoleService{
		findByIDFn: func(ctx context.Context, roleID uint) (*models.AdminRole, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(5), roleID)
			return &models.AdminRole{ID: 5, Name: "Support", IsActive: true}, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/admin-role/5", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.FindById(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Admin role found successfully", body["message"])
}

func TestAdminRoleControllerGetAllPermissionsReturnsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAdminRoleService{
		getAllPermissionsFn: func(ctx context.Context) map[string][]map[string]string {
			require.NotNil(t, ctx)
			return map[string][]map[string]string{
				"admin_role": {
					{"permission": "admin_role:read", "action": "read", "description": "View admin roles"},
				},
			}
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/admin-role/permissions", nil)

	ctrl.GetAllPermissions(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Permissions retrieved successfully", body["message"])
}

func TestAdminRoleControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockAdminRoleService{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/admin/admin-role", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
