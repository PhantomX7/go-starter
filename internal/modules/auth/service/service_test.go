package service_test

import (
	"context"
	"testing"
	"time"

	casbinv2 "github.com/casbin/casbin/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/internal/modules/auth/service"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type mockUserRepository struct {
	createFn   func(context.Context, *models.User) error
	updateFn   func(context.Context, *models.User) error
	findByIDFn func(context.Context, uint, ...repository.Association) (*models.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, entity *models.User) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}
func (m *mockUserRepository) Update(ctx context.Context, entity *models.User) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}
func (m *mockUserRepository) Delete(context.Context, *models.User) error {
	panic("unexpected Delete call")
}
func (m *mockUserRepository) FindByID(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}
func (m *mockUserRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.User, error) {
	panic("unexpected FindAll call")
}
func (m *mockUserRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockUserRepository) FindByUsername(context.Context, string) (*models.User, error) {
	panic("unexpected FindByUsername call")
}
func (m *mockUserRepository) FindByEmail(context.Context, string) (*models.User, error) {
	panic("unexpected FindByEmail call")
}

var _ userrepository.UserRepository = (*mockUserRepository)(nil)

type mockRefreshTokenRepository struct {
	createFn                  func(context.Context, *models.RefreshToken) error
	findByTokenFn             func(context.Context, string) (*models.RefreshToken, error)
	revokeByTokenFn           func(context.Context, string) error
	revokeAllByUserIDExceptFn func(context.Context, uint, string) error
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, entity *models.RefreshToken) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}
func (m *mockRefreshTokenRepository) Update(context.Context, *models.RefreshToken) error {
	panic("unexpected Update call")
}
func (m *mockRefreshTokenRepository) Delete(context.Context, *models.RefreshToken) error {
	panic("unexpected Delete call")
}
func (m *mockRefreshTokenRepository) FindByID(context.Context, uint, ...repository.Association) (*models.RefreshToken, error) {
	panic("unexpected FindByID call")
}
func (m *mockRefreshTokenRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.RefreshToken, error) {
	panic("unexpected FindAll call")
}
func (m *mockRefreshTokenRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockRefreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	if m.findByTokenFn == nil {
		panic("unexpected FindByToken call")
	}
	return m.findByTokenFn(ctx, token)
}
func (m *mockRefreshTokenRepository) FindActiveByID(context.Context, uuid.UUID) (*models.RefreshToken, error) {
	panic("unexpected FindActiveByID call")
}
func (m *mockRefreshTokenRepository) GetValidCountByUserID(context.Context, uint) (int64, error) {
	panic("unexpected GetValidCountByUserID call")
}
func (m *mockRefreshTokenRepository) DeleteInvalidToken(context.Context) error {
	panic("unexpected DeleteInvalidToken call")
}
func (m *mockRefreshTokenRepository) RevokeAllByUserID(context.Context, uint) error {
	panic("unexpected RevokeAllByUserID call")
}
func (m *mockRefreshTokenRepository) RevokeAllByUserIDExcept(ctx context.Context, userID uint, exceptToken string) error {
	if m.revokeAllByUserIDExceptFn == nil {
		panic("unexpected RevokeAllByUserIDExcept call")
	}
	return m.revokeAllByUserIDExceptFn(ctx, userID, exceptToken)
}
func (m *mockRefreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	if m.revokeByTokenFn == nil {
		panic("unexpected RevokeByToken call")
	}
	return m.revokeByTokenFn(ctx, token)
}

var _ refreshtokenrepository.RefreshTokenRepository = (*mockRefreshTokenRepository)(nil)

type mockLogRepository struct {
	createFn func(context.Context, *models.Log) error
}

func (m *mockLogRepository) Create(ctx context.Context, entity *models.Log) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}
func (m *mockLogRepository) Update(context.Context, *models.Log) error {
	panic("unexpected Update call")
}
func (m *mockLogRepository) Delete(context.Context, *models.Log) error {
	panic("unexpected Delete call")
}
func (m *mockLogRepository) FindByID(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindByID call")
}
func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}
func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

type mockCasbinClient struct {
	getRolePermissionsFn func(uint) []string
}

func (m *mockCasbinClient) GetEnforcer() *casbinv2.Enforcer { return nil }
func (m *mockCasbinClient) AddRolePermissions(uint, []string) error {
	panic("unexpected AddRolePermissions call")
}
func (m *mockCasbinClient) RemoveRolePermissions(uint, []string) error {
	panic("unexpected RemoveRolePermissions call")
}
func (m *mockCasbinClient) SetRolePermissions(uint, []string) error {
	panic("unexpected SetRolePermissions call")
}
func (m *mockCasbinClient) GetRolePermissions(roleID uint) []string {
	if m.getRolePermissionsFn == nil {
		panic("unexpected GetRolePermissions call")
	}
	return m.getRolePermissionsFn(roleID)
}
func (m *mockCasbinClient) CheckPermission(uint, string) (bool, error) {
	panic("unexpected CheckPermission call")
}
func (m *mockCasbinClient) CheckPermissionWithRoot(string, *uint, string) (bool, error) {
	panic("unexpected CheckPermissionWithRoot call")
}
func (m *mockCasbinClient) DeleteRole(uint) error { panic("unexpected DeleteRole call") }

var _ casbin.Client = (*mockCasbinClient)(nil)

type mockTxManager struct {
	executeFn func(context.Context, func(context.Context) error) error
}

func (m *mockTxManager) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if m.executeFn == nil {
		panic("unexpected ExecuteInTransaction call")
	}
	return m.executeFn(ctx, fn)
}

var _ transaction_manager.TransactionManager = (*mockTxManager)(nil)

func setupConfig(t *testing.T) {
	t.Helper()

	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_EXPIRATION", "10m")
	t.Setenv("JWT_REFRESH_EXPIRATION", "72h")
	t.Setenv("JWT_ISSUER", "athleton-test")
	t.Setenv("APP_NAME", "Athleton Test")
	t.Setenv("APP_ENVIRONMENT", "development")

	_, err := config.Load()
	require.NoError(t, err)
}

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func newAuthJWT(t *testing.T, userRepo userrepository.UserRepository, refreshRepo refreshtokenrepository.RefreshTokenRepository, logRepo logrepository.LogRepository) *authjwt.AuthJWT {
	t.Helper()
	setupConfig(t)
	setupLogger(t)

	auth, err := authjwt.NewAuthJWT(userRepo, refreshRepo, logRepo)
	require.NoError(t, err)
	return auth
}

func TestAuthServiceGetMeReturnsUserWithPermissions(t *testing.T) {
	setupLogger(t)

	adminRoleID := uint(7)
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(5), id)
			require.NotEmpty(t, preloads)
			return &models.User{
				ID:          5,
				Username:    "admin",
				Name:        "Admin User",
				Email:       "admin@example.com",
				Phone:       "081",
				IsActive:    true,
				Role:        models.UserRoleAdmin,
				AdminRoleID: &adminRoleID,
				AdminRole:   &models.AdminRole{ID: adminRoleID, Name: "Manager"},
			}, nil
		},
	}
	casbinClient := &mockCasbinClient{
		getRolePermissionsFn: func(roleID uint) []string {
			require.Equal(t, adminRoleID, roleID)
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAuthService(userRepo, &mockLogRepository{}, nil, casbinClient, &mockTxManager{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 5})

	me, err := svc.GetMe(ctx)

	require.NoError(t, err)
	require.Equal(t, uint(5), me.ID)
	require.NotNil(t, me.AdminRole)
	require.Equal(t, []string{permissions.UserRead.String()}, me.AdminRole.Permissions)
}

func TestAuthServiceRegisterCreatesUserAndTokens(t *testing.T) {
	logRepo := &mockLogRepository{}
	userRepo := &mockUserRepository{
		createFn: func(ctx context.Context, user *models.User) error {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, "user@example.com", user.Email)
			require.Equal(t, "user@example.com", user.Username)
			require.Equal(t, "08123", user.Phone)
			require.Equal(t, models.UserRoleUser, user.Role)
			require.True(t, user.IsActive)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("secret123")))
			user.ID = 9
			return nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(9), token.UserID)
			require.NotEmpty(t, token.Token)
			require.NotEqual(t, uuid.Nil, token.ID)
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, logRepo)
	txManager := &mockTxManager{
		executeFn: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := service.NewAuthService(userRepo, logRepo, auth, &mockCasbinClient{}, txManager)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	res, err := svc.Register(ctx, &dto.RegisterRequest{
		Name:         "User",
		BusinessName: "Biz",
		Email:        " User@Example.com ",
		Phone:        " 08123 ",
		Password:     "secret123",
	})

	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
	require.Equal(t, "Bearer", res.TokenType)
}

func TestAuthServiceRefreshRotatesToken(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(3), id)
			return &models.User{ID: 3, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "old-token", token)
			return &models.RefreshToken{UserID: 3, Token: token}, nil
		},
		revokeByTokenFn: func(ctx context.Context, token string) error {
			require.Equal(t, "old-token", token)
			return nil
		},
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			require.Equal(t, uint(3), entity.UserID)
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})

	svc := service.NewAuthService(userRepo, &mockLogRepository{}, auth, &mockCasbinClient{}, &mockTxManager{})

	res, err := svc.Refresh(context.Background(), &dto.RefreshRequest{RefreshToken: "old-token"})

	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
}

func TestAuthServiceChangePasswordUpdatesHashRevokesTokensAndLogsAdmin(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	require.NoError(t, err)

	user := &models.User{
		ID:       4,
		Name:     "Admin User",
		Role:     models.UserRoleAdmin,
		Password: string(oldHash),
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(4), id)
			return user, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Same(t, user, entity)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("new-password")))
			return nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		revokeAllByUserIDExceptFn: func(ctx context.Context, userID uint, exceptToken string) error {
			require.Equal(t, uint(4), userID)
			require.Equal(t, "keep-token", exceptToken)
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, logRepo)
	txManager := &mockTxManager{
		executeFn: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := service.NewAuthService(userRepo, logRepo, auth, &mockCasbinClient{}, txManager)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 4, UserName: "Root"})

	err = svc.ChangePassword(ctx, &dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
		ExceptToken: "keep-token",
	})

	require.NoError(t, err)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionChangePassword, entry.Action)
		require.Equal(t, "Root changed password for: Admin User", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAuthServiceLogoutRevokesRefreshToken(t *testing.T) {
	userRepo := &mockUserRepository{}
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "refresh-token", token)
			return &models.RefreshToken{UserID: 6, Token: token}, nil
		},
		revokeByTokenFn: func(ctx context.Context, token string) error {
			require.Equal(t, "refresh-token", token)
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})

	svc := service.NewAuthService(userRepo, &mockLogRepository{}, auth, &mockCasbinClient{}, &mockTxManager{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 6})

	err := svc.Logout(ctx, &dto.LogoutRequest{RefreshToken: "refresh-token"})

	require.NoError(t, err)
}
