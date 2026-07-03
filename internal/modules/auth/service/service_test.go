package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/internal/modules/auth/service"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	refreshtokenmocks "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository/mocks"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	usermocks "github.com/PhantomX7/athleton/internal/modules/user/repository/mocks"
	casbinmocks "github.com/PhantomX7/athleton/libs/casbin/mocks"
	txmocks "github.com/PhantomX7/athleton/libs/transaction_manager/mocks"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func setupConfig(t *testing.T) *config.Config {
	t.Helper()

	t.Setenv("JWT_SECRET", "test-secret-of-at-least-32-characters")
	t.Setenv("JWT_EXPIRATION", "10m")
	t.Setenv("JWT_REFRESH_EXPIRATION", "72h")
	t.Setenv("JWT_ISSUER", "athleton-test")
	t.Setenv("APP_NAME", "Athleton Test")
	t.Setenv("APP_ENVIRONMENT", "development")
	// Config validation requires an admin default password (no default is
	// provided on purpose); any strong non-weak value satisfies it in tests.
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "test-admin-password-123")

	cfg, err := config.Load()
	require.NoError(t, err)
	return cfg
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
	cfg := setupConfig(t)
	setupLogger(t)

	auth, err := authjwt.NewAuthJWT(cfg, userRepo, refreshRepo, logRepo, &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	})
	require.NoError(t, err)
	return auth
}

func TestAuthServiceGetMeReturnsUserWithPermissions(t *testing.T) {
	setupLogger(t)

	adminRoleID := uint(7)
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
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
	casbinClient := &casbinmocks.ClientMock{
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, adminRoleID, roleID)
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAuthService(userRepo, &logmocks.LogRepositoryMock{}, nil, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 5})

	me, err := svc.GetMe(ctx)

	require.NoError(t, err)
	require.Equal(t, uint(5), me.ID)
	require.NotNil(t, me.AdminRole)
	require.Equal(t, []string{permissions.UserRead.String()}, me.AdminRole.Permissions)
}

func TestAuthServiceGetMeToleratesMissingPreloadedRole(t *testing.T) {
	setupLogger(t)

	// AdminRoleID can point at a role whose row failed to preload (soft-deleted
	// role, seed drift). GetMe must degrade gracefully instead of panicking.
	adminRoleID := uint(7)
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(context.Context, uint, ...repository.Association) (*models.User, error) {
			return &models.User{
				ID:          5,
				Username:    "admin",
				IsActive:    true,
				Role:        models.UserRoleAdmin,
				AdminRoleID: &adminRoleID,
				AdminRole:   nil,
			}, nil
		},
	}

	svc := service.NewAuthService(userRepo, &logmocks.LogRepositoryMock{}, nil, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 5})

	me, err := svc.GetMe(ctx)

	require.NoError(t, err)
	require.Equal(t, uint(5), me.ID)
	require.Nil(t, me.AdminRole)
}

func TestAuthServiceRegisterCreatesUserAndTokens(t *testing.T) {
	logRepo := &logmocks.LogRepositoryMock{}
	userRepo := &usermocks.UserRepositoryMock{
		CreateFunc: func(ctx context.Context, user *models.User) error {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, "user@example.com", user.Email)
			require.Equal(t, "user@example.com", user.Username)
			require.Equal(t, "08123", user.Phone)
			require.Equal(t, models.UserRoleUser, user.Role)
			require.True(t, user.IsActive)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("secret123")))
			require.NotNil(t, user.PasswordChangedAt, "a self-chosen password at registration counts as changed")
			user.ID = 9
			return nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		CreateFunc: func(ctx context.Context, token *models.RefreshToken) error {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(9), token.UserID)
			require.NotEmpty(t, token.Token)
			require.NotEqual(t, uuid.Nil, token.ID)
			return nil
		},
		// Session-cap check runs at token creation; no active sessions yet.
		GetValidCountByUserIDFunc: func(context.Context, uint) (int64, error) {
			return 0, nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, logRepo)
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := service.NewAuthService(userRepo, logRepo, auth, &casbinmocks.ClientMock{}, txManager)
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
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(3), id)
			return &models.User{ID: 3, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "old-token", token)
			return &models.RefreshToken{ID: uuid.New(), UserID: 3, Token: token}, nil
		},
		// Rotation swaps the stored hash in place on the existing session row.
		UpdateTokenHashIfActiveFunc: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			require.Equal(t, "old-token", oldToken)
			require.NotEmpty(t, newToken)
			require.NotEqual(t, oldToken, newToken)
			return true, nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, &logmocks.LogRepositoryMock{})

	svc := service.NewAuthService(userRepo, &logmocks.LogRepositoryMock{}, auth, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})

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
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(4), id)
			return user, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.User) error {
			require.Same(t, user, entity)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("new-password")))
			require.NotNil(t, entity.PasswordChangedAt, "ChangePassword must clear the must-change-default-password gate")
			return nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDExceptFunc: func(ctx context.Context, userID uint, exceptToken string) error {
			require.Equal(t, uint(4), userID)
			require.Equal(t, "keep-token", exceptToken)
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, logRepo)
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := service.NewAuthService(userRepo, logRepo, auth, &casbinmocks.ClientMock{}, txManager)
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

func TestAuthServiceChangePasswordAuditsRootUsers(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	require.NoError(t, err)

	// Root accounts are the most privileged; their password rotations must be
	// audited just like admin ones.
	user := &models.User{
		ID:       4,
		Name:     "Root User",
		Role:     models.UserRoleRoot,
		Password: string(oldHash),
	}
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return user, nil
		},
		UpdateFunc: func(context.Context, *models.User) error { return nil },
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDExceptFunc: func(context.Context, uint, string) error { return nil },
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, logRepo)
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := service.NewAuthService(userRepo, logRepo, auth, &casbinmocks.ClientMock{}, txManager)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 4, UserName: "Root User"})

	err = svc.ChangePassword(ctx, &dto.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
		ExceptToken: "keep-token",
	})

	require.NoError(t, err)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionChangePassword, entry.Action)
	case <-time.After(2 * time.Second):
		t.Fatal("root password change must produce an audit log")
	}
}

func TestAuthServiceLogoutRevokesRefreshToken(t *testing.T) {
	userRepo := &usermocks.UserRepositoryMock{}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "refresh-token", token)
			return &models.RefreshToken{UserID: 6, Token: token}, nil
		},
		RevokeByTokenFunc: func(ctx context.Context, token string) error {
			require.Equal(t, "refresh-token", token)
			return nil
		},
	}
	auth := newAuthJWT(t, userRepo, refreshRepo, &logmocks.LogRepositoryMock{})

	svc := service.NewAuthService(userRepo, &logmocks.LogRepositoryMock{}, auth, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 6})

	err := svc.Logout(ctx, &dto.LogoutRequest{RefreshToken: "refresh-token"})

	require.NoError(t, err)
}
