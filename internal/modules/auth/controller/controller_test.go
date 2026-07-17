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
	"github.com/PhantomX7/athleton/internal/modules/auth/controller"
	authservicemocks "github.com/PhantomX7/athleton/internal/modules/auth/service/mocks"
)

// RegisterRequest carries a DB-backed unique= tag registered only in bootstrap.
// Register a no-op so a valid register payload binds without a database; these
// tests exercise controller plumbing, not the uniqueness rule.
func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("unique", func(validator.FieldLevel) bool { return true })
		_ = v.RegisterValidation("exist", func(validator.FieldLevel) bool { return true })
	}
}

func TestAuthControllerGetMeReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &authservicemocks.AuthServiceMock{
		GetMeFunc: func(ctx context.Context) (*dto.MeResponse, error) {
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
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", nil)

	ctrl.GetMe(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "get me success", body["message"])
}

func TestAuthControllerRefreshReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &authservicemocks.AuthServiceMock{
		RefreshFunc: func(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
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
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Refresh(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "refresh success", body["message"])
}

func TestAuthControllerChangePasswordReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &authservicemocks.AuthServiceMock{
		ChangePasswordFunc: func(ctx context.Context, req *dto.ChangePasswordRequest) error {
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
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/change-password", bytes.NewBufferString(`{"old_password":"old-password","new_password":"new-password","except_token":"keep-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.ChangePassword(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "password changed successfully", body["message"])
}

func TestAuthControllerLogoutReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &authservicemocks.AuthServiceMock{
		LogoutFunc: func(ctx context.Context, req *dto.LogoutRequest) error {
			require.NotNil(t, ctx)
			require.Equal(t, "refresh-token", req.RefreshToken)
			return nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", bytes.NewBufferString(`{"refresh_token":"refresh-token"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Logout(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "logout successful", body["message"])
}

func TestAuthControllerRegisterRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &authservicemocks.AuthServiceMock{
		RegisterFunc: func(context.Context, *dto.RegisterRequest) (*dto.AuthResponse, error) {
			t.Fatal("Register should not be called for invalid payloads")
			return nil, nil
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/register", bytes.NewBufferString(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Register(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypeBind))
}

func TestAuthControllerRegisterReturnsTokensOnSuccess(t *testing.T) {
	svc := &authservicemocks.AuthServiceMock{
		RegisterFunc: func(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
			require.NotNil(t, ctx)
			require.Equal(t, "alice@example.com", req.Email)
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
	body := `{"name":"Alice","business_name":"Acme","email":"alice@example.com","phone":"081","password":"supersecret"}`
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Register(ctx)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "register success", resp["message"])
	data, _ := resp["data"].(map[string]any)
	require.Equal(t, "access", data["access_token"])
	require.Equal(t, "refresh", data["refresh_token"])
}

func TestAuthControllerRefreshPropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &authservicemocks.AuthServiceMock{
		RefreshFunc: func(context.Context, *dto.RefreshRequest) (*dto.AuthResponse, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/refresh", bytes.NewBufferString(`{"refresh_token":"t"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Refresh(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestAuthControllerChangePasswordPropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &authservicemocks.AuthServiceMock{
		ChangePasswordFunc: func(context.Context, *dto.ChangePasswordRequest) error {
			return expectedErr
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	body := `{"old_password":"old-password","new_password":"new-password","except_token":"keep"}`
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/change-password", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.ChangePassword(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestAuthControllerLogoutPropagatesServiceError(t *testing.T) {
	expectedErr := errors.New("service failed")
	svc := &authservicemocks.AuthServiceMock{
		LogoutFunc: func(context.Context, *dto.LogoutRequest) error {
			return expectedErr
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/logout", bytes.NewBufferString(`{"refresh_token":"t"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ctrl.Logout(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func TestAuthControllerGetMePropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &authservicemocks.AuthServiceMock{
		GetMeFunc: func(context.Context) (*dto.MeResponse, error) {
			return nil, expectedErr
		},
	}

	ctrl := controller.NewAuthController(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/me", nil)

	ctrl.GetMe(ctx)

	require.Len(t, ctx.Errors, 1)
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}
