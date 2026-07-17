package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	adminroleservicemocks "github.com/PhantomX7/athleton/internal/modules/admin_role/service/mocks"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

// The DTOs carry DB-backed custom tags (unique=, exist=) that are only
// registered in bootstrap. Register no-op versions on the binding engine so
// valid payloads bind here without a database — these controller tests exercise
// the bind→service→response plumbing, not the uniqueness rules (those are
// covered in pkg/validator and the integration suite).
func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("unique", func(validator.FieldLevel) bool { return true })
		_ = v.RegisterValidation("exist", func(validator.FieldLevel) bool { return true })
	}
}

func TestAdminRoleControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &adminroleservicemocks.AdminRoleServiceMock{
		IndexFunc: func(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
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
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/admin-role?limit=2&offset=4&sort=name+asc", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])
}

func TestAdminRoleControllerDeleteRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &adminroleservicemocks.AdminRoleServiceMock{
		DeleteFunc: func(context.Context, uint) error {
			t.Fatal("Delete should not be called for invalid ids")
			return nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/admin/admin-role/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.Delete(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, cerrors.ErrInvalidInput)
}

func TestAdminRoleControllerFindByIDReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &adminroleservicemocks.AdminRoleServiceMock{
		FindByIDFunc: func(ctx context.Context, roleID uint) (*models.AdminRole, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(5), roleID)
			return &models.AdminRole{ID: 5, Name: "Support", IsActive: true}, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/admin-role/5", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.FindByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Admin role found successfully", body["message"])

	// Pin the response contract to the AdminRoleResponse DTO: all admin-role
	// endpoints serialize through it, so a new model field can never leak into
	// the API without an explicit DTO change.
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.ElementsMatch(t,
		[]string{"id", "name", "description", "is_active", "permissions", "created_at", "updated_at"},
		slices.Collect(maps.Keys(data)),
	)
	require.Equal(t, float64(5), data["id"])
	require.Equal(t, "Support", data["name"])
}

func TestAdminRoleControllerGetAllPermissionsReturnsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &adminroleservicemocks.AdminRoleServiceMock{
		GetAllPermissionsFunc: func(ctx context.Context) map[string][]map[string]string {
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
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/admin-role/permissions", nil)

	ctrl.GetAllPermissions(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Permissions retrieved successfully", body["message"])
}

func TestAdminRoleControllerCreateReturnsCreatedResponse(t *testing.T) {
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		CreateFunc: func(ctx context.Context, req *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
			require.NotNil(t, ctx)
			require.Equal(t, "Manager", req.Name)
			require.Equal(t, []string{"admin_role:read"}, req.Permissions)
			return &models.AdminRole{ID: 9, Name: req.Name, IsActive: true}, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := `{"name":"Manager","description":"manages","permissions":["admin_role:read"]}`
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/admin-role", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Create(ctx)

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Admin role created successfully", resp["message"])
}

func TestAdminRoleControllerCreateRejectsInvalidPayload(t *testing.T) {
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		CreateFunc: func(context.Context, *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
			t.Fatal("Create should not be called for invalid payloads")
			return nil, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	// Missing required name and permissions.
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/admin-role", bytes.NewBufferString(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Create(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypeBind))
}

func TestAdminRoleControllerCreatePropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		CreateFunc: func(context.Context, *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := `{"name":"Manager","permissions":["admin_role:read"]}`
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/admin-role", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Create(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestAdminRoleControllerUpdateReturnsSuccessResponse(t *testing.T) {
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		UpdateFunc: func(ctx context.Context, roleID uint, req *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(5), roleID)
			require.Equal(t, []string{"admin_role:read"}, req.Permissions)
			return &models.AdminRole{ID: roleID, Name: "Support", IsActive: true}, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := `{"permissions":["admin_role:read"]}`
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/admin/admin-role/5", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Admin role updated successfully", resp["message"])
}

func TestAdminRoleControllerUpdateRejectsInvalidID(t *testing.T) {
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		UpdateFunc: func(context.Context, uint, *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
			t.Fatal("Update should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/admin/admin-role/bad", bytes.NewBufferString(`{"permissions":["x"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, cerrors.ErrInvalidInput)
}

func TestAdminRoleControllerUpdatePropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		UpdateFunc: func(context.Context, uint, *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/admin/admin-role/5", bytes.NewBufferString(`{"permissions":["x"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestAdminRoleControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &adminroleservicemocks.AdminRoleServiceMock{
		IndexFunc: func(context.Context, *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewAdminRoleController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/admin-role", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
