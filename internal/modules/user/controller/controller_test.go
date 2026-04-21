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
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/user/controller"
	userservice "github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

type mockUserService struct {
	indexFn           func(context.Context, *pagination.Pagination) ([]*models.User, response.Meta, error)
	updateFn          func(context.Context, uint, *dto.UserUpdateRequest) (*models.User, error)
	findByIDFn        func(context.Context, uint) (*models.User, error)
	assignAdminRoleFn func(context.Context, uint, *dto.UserAssignAdminRoleRequest) (*models.User, error)
	changePasswordFn  func(context.Context, uint, *dto.ChangeAdminPasswordRequest) error
}

func (m *mockUserService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mockUserService) Update(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error) {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, userID, req)
}

func (m *mockUserService) FindByID(ctx context.Context, userID uint) (*models.User, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, userID)
}

func (m *mockUserService) AssignAdminRole(ctx context.Context, userID uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error) {
	if m.assignAdminRoleFn == nil {
		panic("unexpected AssignAdminRole call")
	}
	return m.assignAdminRoleFn(ctx, userID, req)
}

func (m *mockUserService) ChangePassword(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error {
	if m.changePasswordFn == nil {
		panic("unexpected ChangePassword call")
	}
	return m.changePasswordFn(ctx, userID, req)
}

var _ userservice.UserService = (*mockUserService)(nil)

func TestUserControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockUserService{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
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

func TestUserControllerUpdateReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockUserService{
		updateFn: func(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error) {
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

	svc := &mockUserService{
		findByIDFn: func(ctx context.Context, userID uint) (*models.User, error) {
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

	svc := &mockUserService{
		assignAdminRoleFn: func(context.Context, uint, *dto.UserAssignAdminRoleRequest) (*models.User, error) {
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
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}

func TestUserControllerChangePasswordReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockUserService{
		changePasswordFn: func(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error {
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

func TestUserControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockUserService{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.User, response.Meta, error) {
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
