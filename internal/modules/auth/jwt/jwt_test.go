package authjwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PhantomX7/athleton/internal/models"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/pkg/config"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"
	"github.com/appleboy/gin-jwt/v3/core"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	findByIDFn       func(context.Context, uint, ...repository.Association) (*models.User, error)
	findByUsernameFn func(context.Context, string) (*models.User, error)
	findByEmailFn    func(context.Context, string) (*models.User, error)
}

func (m *mockUserRepository) Create(context.Context, *models.User) error {
	panic("unexpected Create call")
}
func (m *mockUserRepository) Update(context.Context, *models.User) error {
	panic("unexpected Update call")
}
func (m *mockUserRepository) Delete(context.Context, *models.User) error {
	panic("unexpected Delete call")
}
func (m *mockUserRepository) FindById(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}
func (m *mockUserRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.User, error) {
	panic("unexpected FindAll call")
}
func (m *mockUserRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	if m.findByUsernameFn == nil {
		panic("unexpected FindByUsername call")
	}
	return m.findByUsernameFn(ctx, username)
}
func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn == nil {
		panic("unexpected FindByEmail call")
	}
	return m.findByEmailFn(ctx, email)
}

var _ userrepository.UserRepository = (*mockUserRepository)(nil)

type mockRefreshTokenRepository struct {
	createFn                  func(context.Context, *models.RefreshToken) error
	findByTokenFn             func(context.Context, string) (*models.RefreshToken, error)
	getValidCountByUserIDFn   func(context.Context, uint) (int64, error)
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
func (m *mockRefreshTokenRepository) FindById(context.Context, uint, ...repository.Association) (*models.RefreshToken, error) {
	panic("unexpected FindById call")
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
func (m *mockRefreshTokenRepository) GetValidCountByUserID(ctx context.Context, userID uint) (int64, error) {
	if m.getValidCountByUserIDFn == nil {
		panic("unexpected GetValidCountByUserID call")
	}
	return m.getValidCountByUserIDFn(ctx, userID)
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
func (m *mockLogRepository) FindById(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindById call")
}
func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}
func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

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

func newAuthJWT(t *testing.T, userRepo userrepository.UserRepository, refreshRepo refreshtokenrepository.RefreshTokenRepository, logRepo logrepository.LogRepository) *AuthJWT {
	t.Helper()
	setupConfig(t)
	setupLogger(t)

	auth, err := NewAuthJWT(userRepo, refreshRepo, logRepo)
	require.NoError(t, err)
	return auth
}

func TestPayloadFuncIncludesIdentityRoleAndAdminRoleID(t *testing.T) {
	setupConfig(t)

	adminRoleID := uint(9)
	a := &AuthJWT{}
	claims := a.payloadFunc(&models.User{
		ID:          5,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &adminRoleID,
	})

	require.Equal(t, uint(5), claims[IdentityKey])
	require.Equal(t, models.UserRoleAdmin.ToString(), claims[RoleKey])
	require.Equal(t, adminRoleID, claims[AdminRoleIDKey])
	require.Equal(t, config.Get().JWT.Issuer, claims["iss"])
}

func TestIdentityHandlerBuildsUserFromClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)

	a := &AuthJWT{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("JWT_PAYLOAD", jwt.MapClaims{
		IdentityKey:    float64(7),
		RoleKey:        "admin",
		AdminRoleIDKey: float64(3),
	})

	user, ok := a.identityHandler(c).(*models.User)

	require.True(t, ok)
	require.Equal(t, uint(7), user.ID)
	require.Equal(t, models.UserRoleAdmin, user.Role)
	require.NotNil(t, user.AdminRoleID)
	require.Equal(t, uint(3), *user.AdminRoleID)
}

func TestValidateCredentialsUsesEmailAndReturnsUser(t *testing.T) {
	setupLogger(t)

	hashed, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &mockUserRepository{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			require.Equal(t, "alice@example.com", email)
			return &models.User{
				ID:       1,
				Email:    email,
				IsActive: true,
				Password: string(hashed),
			}, nil
		},
	}

	a := &AuthJWT{userRepo: repo}
	user, err := a.validateCredentials(context.Background(), " Alice@Example.com ", "secret123")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, uint(1), user.ID)
}

func TestValidateCredentialsRejectsInactiveUser(t *testing.T) {
	setupLogger(t)

	hashed, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &mockUserRepository{
		findByUsernameFn: func(context.Context, string) (*models.User, error) {
			return &models.User{
				ID:       2,
				Username: "alice",
				IsActive: false,
				Password: string(hashed),
			}, nil
		},
	}

	a := &AuthJWT{userRepo: repo}
	user, err := a.validateCredentials(context.Background(), "alice", "secret123")

	require.Nil(t, user)
	require.Error(t, err)
}

func TestAuthorizerSetsContextValuesForActiveUserWithSession(t *testing.T) {
	setupLogger(t)

	adminRoleID := uint(4)
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(5), id)
			return &models.User{
				ID:          5,
				Name:        "Alice",
				Role:        models.UserRoleAdmin,
				IsActive:    true,
				AdminRoleID: &adminRoleID,
			}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		getValidCountByUserIDFn: func(ctx context.Context, userID uint) (int64, error) {
			require.Equal(t, uint(5), userID)
			return 1, nil
		},
	}

	a := &AuthJWT{userRepo: repo, refreshTokenRepo: refreshRepo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	c.Request = req.WithContext(context.Background())

	allowed := a.authorizer(c, &models.User{ID: 5})

	require.True(t, allowed)
	values, err := utils.ValuesFromContext(c.Request.Context())
	require.NoError(t, err)
	require.Equal(t, uint(5), values.UserID)
	require.Equal(t, "Alice", values.UserName)
	require.Equal(t, "admin", values.Role)
	require.NotNil(t, values.AdminRoleID)
	require.Equal(t, uint(4), *values.AdminRoleID)
}

func TestGenerateTokensForUserCreatesRefreshToken(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			require.NotNil(t, ctx)
			require.Equal(t, uint(7), entity.UserID)
			require.NotEmpty(t, entity.Token)
			require.True(t, entity.ExpiresAt.After(time.Now()))
			return nil
		},
	}

	a := newAuthJWT(t, &mockUserRepository{}, refreshRepo, &mockLogRepository{})

	res, err := a.GenerateTokensForUser(context.Background(), &models.User{ID: 7, Role: models.UserRoleUser})

	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
	require.Equal(t, "Bearer", res.TokenType)
}

func TestValidateAndRotateRefreshTokenReturnsNewTokens(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(11), id)
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	revokeCalled := false
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "old-token", token)
			return &models.RefreshToken{UserID: 11, Token: token}, nil
		},
		revokeByTokenFn: func(ctx context.Context, token string) error {
			require.Equal(t, "old-token", token)
			revokeCalled = true
			return nil
		},
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			require.Equal(t, uint(11), entity.UserID)
			return nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.NoError(t, err)
	require.True(t, revokeCalled)
	require.NotEmpty(t, res.AccessToken)
	require.NotEmpty(t, res.RefreshToken)
}

func TestRevokeRefreshTokenIsIdempotentWhenTokenMissing(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(context.Context, string) (*models.RefreshToken, error) {
			return nil, cerrors.NewNotFoundError("missing")
		},
	}

	a := &AuthJWT{refreshTokenRepo: refreshRepo}
	err := a.RevokeRefreshToken(context.Background(), "missing-token", 1)

	require.NoError(t, err)
}

func TestRevokeRefreshTokenRejectsOtherUsersToken(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(context.Context, string) (*models.RefreshToken, error) {
			return &models.RefreshToken{UserID: 99}, nil
		},
	}

	a := &AuthJWT{refreshTokenRepo: refreshRepo}
	err := a.RevokeRefreshToken(context.Background(), "token", 1)

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestLoginResponseReturnsInternalErrorWithoutAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	a := &AuthJWT{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", nil)

	a.loginResponse(c, &core.Token{AccessToken: "access"})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
	require.Equal(t, "internal error", body["message"])
}
