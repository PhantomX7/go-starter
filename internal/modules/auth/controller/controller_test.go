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
	"github.com/PhantomX7/athleton/internal/modules/auth/controller"
	authservice "github.com/PhantomX7/athleton/internal/modules/auth/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct {
	getMeFn          func(context.Context) (*dto.MeResponse, error)
	registerFn       func(context.Context, *dto.RegisterRequest) (*dto.AuthResponse, error)
	refreshFn        func(context.Context, *dto.RefreshRequest) (*dto.AuthResponse, error)
	changePasswordFn func(context.Context, *dto.ChangePasswordRequest) error
	logoutFn         func(context.Context, *dto.LogoutRequest) error
}

func (m *mockAuthService) GetMe(ctx context.Context) (*dto.MeResponse, error) {
	if m.getMeFn == nil {
		panic("unexpected GetMe call")
	}
	return m.getMeFn(ctx)
}

func (m *mockAuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	if m.registerFn == nil {
		panic("unexpected Register call")
	}
	return m.registerFn(ctx, req)
}

func (m *mockAuthService) Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
	if m.refreshFn == nil {
		panic("unexpected Refresh call")
	}
	return m.refreshFn(ctx, req)
}

func (m *mockAuthService) ChangePassword(ctx context.Context, req *dto.ChangePasswordRequest) error {
	if m.changePasswordFn == nil {
		panic("unexpected ChangePassword call")
	}
	return m.changePasswordFn(ctx, req)
}

func (m *mockAuthService) Logout(ctx context.Context, req *dto.LogoutRequest) error {
	if m.logoutFn == nil {
		panic("unexpected Logout call")
	}
	return m.logoutFn(ctx, req)
}

var _ authservice.AuthService = (*mockAuthService)(nil)

func TestAuthControllerGetMeReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		getMeFn: func(ctx context.Context) (*dto.MeResponse, error) {
			require.NotNil(t, ctx)
			return &dto.MeResponse{
				UserResponse: dto.UserResponse{
					ID:       1,
					Username: "alice",
					Name:     "Alice",
					Email:    "alice@example.com",
					Phone:    "081",
					Role:     "user",
				},
			}, nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/auth/me", nil)

	ctrl.GetMe(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "get me success", body["message"])
}

func TestAuthControllerRefreshReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		refreshFn: func(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
			require.NotNil(t, ctx)
			require.Equal(t, "refresh-token", req.RefreshToken)
			return &dto.AuthResponse{
				AccessToken:  "access",
				RefreshToken: "refresh",
				TokenType:    "Bearer",
			}, nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Refresh(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "refresh success", body["message"])
}

func TestAuthControllerChangePasswordReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		changePasswordFn: func(ctx context.Context, req *dto.ChangePasswordRequest) error {
			require.NotNil(t, ctx)
			require.Equal(t, "old-password", req.OldPassword)
			require.Equal(t, "new-password", req.NewPassword)
			require.Equal(t, "keep-token", req.ExceptToken)
			return nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewBufferString(`{"old_password":"old-password","new_password":"new-password","except_token":"keep-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.ChangePassword(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "password changed successfully", body["message"])
}

func TestAuthControllerLogoutReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		logoutFn: func(ctx context.Context, req *dto.LogoutRequest) error {
			require.NotNil(t, ctx)
			require.Equal(t, "refresh-token", req.RefreshToken)
			return nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Logout(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "logout successful", body["message"])
}

func TestAuthControllerRegisterRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		registerFn: func(context.Context, *dto.RegisterRequest) (*dto.AuthResponse, error) {
			t.Fatal("Register should not be called for invalid payloads")
			return nil, nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Register(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypeBind))
}

func TestAuthControllerGetMePropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mockAuthService{
		getMeFn: func(context.Context) (*dto.MeResponse, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/auth/me", nil)

	ctrl.GetMe(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
