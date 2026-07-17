package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/user/controller"
	userservicemocks "github.com/PhantomX7/athleton/internal/modules/user/service/mocks"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

// The DTOs carry DB-backed custom tags (exist=, unique=) registered only in
// bootstrap. Register no-op versions on the binding engine so valid payloads
// bind without a database — these tests exercise controller plumbing, not the
// DB-backed validators (covered in pkg/validator and the integration suite).
func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("unique", func(validator.FieldLevel) bool { return true })
		_ = v.RegisterValidation("exist", func(validator.FieldLevel) bool { return true })
	}
}

func TestUserControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		IndexFunc: func(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 2, pg.Limit)
			require.Equal(t, 3, pg.Offset)
			require.Equal(t, "username asc", pg.Order)
			return []*models.User{
					{ID: 1, Username: "alice", Name: "Alice", Email: "alice@example.com", Phone: "081", Role: models.UserRoleUser},
				}, response.Meta{
					Total:  5,
					Offset: 3,
					Limit:  2,
				}, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/user?limit=2&offset=3&sort=username+asc", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])
}

func TestUserControllerCreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		CreateFunc: func(ctx context.Context, req *dto.AdminUserCreateRequest) (*models.User, error) {
			require.NotNil(t, ctx)
			require.Equal(t, "new-admin", req.Username)
			require.Equal(t, uint(3), req.AdminRoleID)
			return &models.User{
				ID:       9,
				Username: "new-admin",
				Name:     "New Admin",
				Email:    "new.admin@test.local",
				Phone:    "083",
				Role:     models.UserRoleAdmin,
			}, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/user", bytes.NewBufferString(
		`{"username":"new-admin","name":"New Admin","email":"new.admin@test.local","phone":"083","password":"initial-pass-123","admin_role_id":3}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Create(ctx)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Admin user created successfully", body["message"])
}

func TestUserControllerCreateRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		CreateFunc: func(context.Context, *dto.AdminUserCreateRequest) (*models.User, error) {
			t.Fatal("Create should not be called when binding fails")
			return nil, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/user", bytes.NewBufferString(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Create(ctx)

	require.NotEmpty(t, ctx.Errors, "an empty body must record a binding error")
}

func TestUserControllerUpdateReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		UpdateFunc: func(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(5), userID)
			require.NotNil(t, req.Name)
			require.Equal(t, "Alice Updated", *req.Name)
			return &models.User{
				ID:       5,
				Username: "alice",
				Name:     "Alice Updated",
				Email:    "alice@example.com",
				Phone:    "081",
				Role:     models.UserRoleUser,
			}, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/user/5", bytes.NewBufferString(`{"name":"Alice Updated"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "User updated successfully", body["message"])
}

func TestUserControllerFindByIDReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		FindByIDFunc: func(ctx context.Context, userID uint) (*models.User, error) {
			require.NotNil(t, ctx)
			require.Equal(t, uint(7), userID)
			return &models.User{
				ID:       7,
				Username: "bob",
				Name:     "Bob",
				Email:    "bob@example.com",
				Phone:    "082",
				Role:     models.UserRoleAdmin,
			}, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/user/7", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}

	ctrl.FindByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "User found successfully", body["message"])
}

func TestUserControllerAssignAdminRoleRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		AssignAdminRoleFunc: func(context.Context, uint, *dto.UserAssignAdminRoleRequest) (*models.User, error) {
			t.Fatal("AssignAdminRole should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/user/bad/admin-role", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.AssignAdminRole(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, cerrors.ErrInvalidInput)
}

func TestUserControllerDeleteReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		DeleteFunc: func(ctx context.Context, userID uint) error {
			require.NotNil(t, ctx)
			require.Equal(t, uint(6), userID)
			return nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/admin/user/6", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "6"}}

	ctrl.Delete(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "User deleted successfully", body["message"])
}

func TestUserControllerDeleteRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		DeleteFunc: func(context.Context, uint) error {
			t.Fatal("Delete should not be called for invalid ids")
			return nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/admin/user/bad", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "bad"}}

	ctrl.Delete(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, cerrors.ErrInvalidInput)
}

func TestUserControllerDeletePropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("delete failed")
	svc := &userservicemocks.UserServiceMock{
		DeleteFunc: func(context.Context, uint) error {
			return expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/admin/user/6", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "6"}}

	ctrl.Delete(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestUserControllerChangePasswordReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &userservicemocks.UserServiceMock{
		ChangePasswordFunc: func(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error {
			require.NotNil(t, ctx)
			require.Equal(t, uint(9), userID)
			require.Equal(t, "new-password", req.NewPassword)
			return nil
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/user/9/change-password", bytes.NewBufferString(`{"new_password":"new-password"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "9"}}

	ctrl.ChangePassword(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Password changed successfully", body["message"])
}

func TestUserControllerUpdatePropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &userservicemocks.UserServiceMock{
		UpdateFunc: func(context.Context, uint, *dto.UserUpdateRequest) (*models.User, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/user/5", bytes.NewBufferString(`{"name":"Alice"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestUserControllerFindByIDPropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &userservicemocks.UserServiceMock{
		FindByIDFunc: func(context.Context, uint) (*models.User, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/user/7", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "7"}}

	ctrl.FindByID(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestUserControllerAssignAdminRolePropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &userservicemocks.UserServiceMock{
		AssignAdminRoleFunc: func(context.Context, uint, *dto.UserAssignAdminRoleRequest) (*models.User, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/user/5/admin-role", bytes.NewBufferString(`{"admin_role_id":3}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	ctrl.AssignAdminRole(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestUserControllerChangePasswordPropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &userservicemocks.UserServiceMock{
		ChangePasswordFunc: func(context.Context, uint, *dto.ChangeAdminPasswordRequest) error {
			return expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/admin/user/9/change-password", bytes.NewBufferString(`{"new_password":"new-password"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "id", Value: "9"}}

	ctrl.ChangePassword(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestUserControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &userservicemocks.UserServiceMock{
		IndexFunc: func(context.Context, *pagination.Pagination) ([]*models.User, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.NewUserController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/user", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
