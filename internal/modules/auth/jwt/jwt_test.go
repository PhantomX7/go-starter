package authjwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/appleboy/gin-jwt/v3/core"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

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
	findActiveByIDFn          func(context.Context, uuid.UUID) (*models.RefreshToken, error)
	getValidCountByUserIDFn   func(context.Context, uint) (int64, error)
	revokeByTokenFn           func(context.Context, string) error
	revokeByTokenIfActiveFn   func(context.Context, string) (bool, error)
	revokeAllByUserIDFn       func(context.Context, uint) error
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
func (m *mockRefreshTokenRepository) FindActiveByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
	if m.findActiveByIDFn == nil {
		panic("unexpected FindActiveByID call")
	}
	return m.findActiveByIDFn(ctx, id)
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
func (m *mockRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	if m.revokeAllByUserIDFn == nil {
		panic("unexpected RevokeAllByUserID call")
	}
	return m.revokeAllByUserIDFn(ctx, userID)
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
func (m *mockRefreshTokenRepository) RevokeByTokenIfActive(ctx context.Context, token string) (bool, error) {
	if m.revokeByTokenIfActiveFn == nil {
		panic("unexpected RevokeByTokenIfActive call")
	}
	return m.revokeByTokenIfActiveFn(ctx, token)
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

func setupConfig(t *testing.T) *config.Config {
	t.Helper()

	t.Setenv("JWT_SECRET", "test-secret-of-at-least-32-characters")
	t.Setenv("JWT_EXPIRATION", "10m")
	t.Setenv("JWT_REFRESH_EXPIRATION", "72h")
	t.Setenv("JWT_ISSUER", "athleton-test")
	t.Setenv("APP_NAME", "Athleton Test")
	t.Setenv("APP_ENVIRONMENT", "development")

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

// passthroughTxManager runs the closure directly without a real transaction —
// adequate for unit tests whose repositories are mocks (there is nothing to
// commit or roll back).
type passthroughTxManager struct{}

func (passthroughTxManager) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// recordingTxManager reports whether the rotation wrapped its work in a
// transaction, while still executing the closure.
type recordingTxManager struct {
	called bool
}

func (r *recordingTxManager) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	r.called = true
	return fn(ctx)
}

func newAuthJWT(t *testing.T, userRepo userrepository.UserRepository, refreshRepo refreshtokenrepository.RefreshTokenRepository, logRepo logrepository.LogRepository) *AuthJWT {
	t.Helper()
	cfg := setupConfig(t)
	setupLogger(t)

	auth, err := NewAuthJWT(cfg, userRepo, refreshRepo, logRepo, passthroughTxManager{})
	require.NoError(t, err)
	return auth
}

func TestPayloadFuncIncludesIdentityRoleSessionAndAdminRoleID(t *testing.T) {
	cfg := setupConfig(t)

	adminRoleID := uint(9)
	sessionID := uuid.New()
	a := &AuthJWT{cfg: cfg}
	claims := a.payloadFunc(&authSubject{
		User: &models.User{
			ID:          5,
			Role:        models.UserRoleAdmin,
			AdminRoleID: &adminRoleID,
		},
		SessionID: sessionID,
	})

	require.Equal(t, uint(5), claims[IdentityKey])
	require.Equal(t, models.UserRoleAdmin.ToString(), claims[RoleKey])
	require.Equal(t, adminRoleID, claims[AdminRoleIDKey])
	require.Equal(t, sessionID.String(), claims[SessionIDKey])
	require.Equal(t, cfg.JWT.Issuer, claims["iss"])
}

func TestPayloadFuncReturnsEmptyForUnknownData(t *testing.T) {
	setupConfig(t)

	a := &AuthJWT{}
	require.Empty(t, a.payloadFunc(&models.User{ID: 1}))
}

func TestIdentityHandlerBuildsSubjectFromClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionID := uuid.New()
	a := &AuthJWT{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("JWT_PAYLOAD", jwt.MapClaims{
		IdentityKey:    float64(7),
		RoleKey:        "admin",
		AdminRoleIDKey: float64(3),
		SessionIDKey:   sessionID.String(),
	})

	subj, ok := a.identityHandler(c).(*authSubject)

	require.True(t, ok)
	require.NotNil(t, subj.User)
	require.Equal(t, uint(7), subj.User.ID)
	require.Equal(t, models.UserRoleAdmin, subj.User.Role)
	require.NotNil(t, subj.User.AdminRoleID)
	require.Equal(t, uint(3), *subj.User.AdminRoleID)
	require.Equal(t, sessionID, subj.SessionID)
}

func TestIdentityHandlerLeavesSessionZeroWhenJTIMissingOrInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	a := &AuthJWT{}

	cases := map[string]jwt.MapClaims{
		"missing": {IdentityKey: float64(1), RoleKey: "user"},
		"invalid": {IdentityKey: float64(1), RoleKey: "user", SessionIDKey: "not-a-uuid"},
	}

	for name, claims := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Set("JWT_PAYLOAD", claims)

			subj, ok := a.identityHandler(c).(*authSubject)

			require.True(t, ok)
			require.Equal(t, uuid.Nil, subj.SessionID)
		})
	}
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

// Regression test for the prior bug where dummyHash was a string literal that
// bcrypt rejected as ErrHashTooShort, doing zero work and leaking account
// existence via timing. The replacement must be a real bcrypt hash at the
// production cost so the timing-equalization path actually pays the cost.
func TestDummyHashIsRealBcryptHashAtProductionCost(t *testing.T) {
	cost, err := bcrypt.Cost(dummyHash())

	require.NoError(t, err, "dummyHash must be parseable by bcrypt")
	require.Equal(t, dummyBcryptCost, cost, "dummyHash cost must match production")
}

func TestAuthorizerSetsContextValuesForActiveUserWithBoundSession(t *testing.T) {
	setupLogger(t)

	adminRoleID := uint(4)
	sessionID := uuid.New()
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
		findActiveByIDFn: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
			require.Equal(t, sessionID, id)
			return &models.RefreshToken{ID: id, UserID: 5}, nil
		},
	}

	a := &AuthJWT{userRepo: repo, refreshTokenRepo: refreshRepo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)
	c.Request = req.WithContext(context.Background())

	allowed := a.authorizer(c, &authSubject{
		User:      &models.User{ID: 5},
		SessionID: sessionID,
	})

	require.True(t, allowed)
	values, err := utils.ValuesFromContext(c.Request.Context())
	require.NoError(t, err)
	require.Equal(t, uint(5), values.UserID)
	require.Equal(t, "Alice", values.UserName)
	require.Equal(t, "admin", values.Role)
	require.NotNil(t, values.AdminRoleID)
	require.Equal(t, uint(4), *values.AdminRoleID)
}

func TestAuthorizerRejectsAccessTokenWithoutSessionClaim(t *testing.T) {
	setupLogger(t)

	a := &AuthJWT{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)

	allowed := a.authorizer(c, &authSubject{
		User:      &models.User{ID: 5},
		SessionID: uuid.Nil,
	})

	require.False(t, allowed)
}

func TestAuthorizerRejectsRevokedSession(t *testing.T) {
	setupLogger(t)

	sessionID := uuid.New()
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findActiveByIDFn: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
			return nil, cerrors.NewNotFoundError("invalid refresh token session")
		},
	}

	a := &AuthJWT{userRepo: repo, refreshTokenRepo: refreshRepo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)

	allowed := a.authorizer(c, &authSubject{
		User:      &models.User{ID: 5},
		SessionID: sessionID,
	})

	require.False(t, allowed)
}

// A user deactivated AFTER login still holds a validly-signed access token.
// The authorizer re-reads the user every request, so a now-inactive account
// must be rejected even though its token and session are otherwise intact.
func TestAuthorizerRejectsUserDeactivatedAfterLogin(t *testing.T) {
	setupLogger(t)

	sessionID := uuid.New()
	sessionLookedUp := false
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: false, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findActiveByIDFn: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
			sessionLookedUp = true
			return &models.RefreshToken{ID: id, UserID: 5}, nil
		},
	}

	a := &AuthJWT{userRepo: repo, refreshTokenRepo: refreshRepo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)

	allowed := a.authorizer(c, &authSubject{
		User:      &models.User{ID: 5},
		SessionID: sessionID,
	})

	require.False(t, allowed)
	require.False(t, sessionLookedUp, "inactive user must be rejected before the session lookup")
}

func TestAuthorizerRejectsSessionBelongingToDifferentUser(t *testing.T) {
	setupLogger(t)

	sessionID := uuid.New()
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findActiveByIDFn: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: id, UserID: 99}, nil
		},
	}

	a := &AuthJWT{userRepo: repo, refreshTokenRepo: refreshRepo}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/protected", nil)

	allowed := a.authorizer(c, &authSubject{
		User:      &models.User{ID: 5},
		SessionID: sessionID,
	})

	require.False(t, allowed)
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
		revokeByTokenIfActiveFn: func(ctx context.Context, token string) (bool, error) {
			require.Equal(t, "old-token", token)
			revokeCalled = true
			return true, nil
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

// Regression test for the prior bug where revoke errors were ignored and a
// fresh token pair was minted anyway, leaving the old refresh token reusable.
func TestValidateAndRotateRefreshTokenFailsWhenRevokeFails(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	createCalled := false
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{UserID: 11, Token: token}, nil
		},
		revokeByTokenIfActiveFn: func(ctx context.Context, token string) (bool, error) {
			return false, cerrors.NewInternalServerError("db boom", errors.New("boom"))
		},
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			createCalled = true
			return nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res)
	require.False(t, createCalled, "must not mint a replacement refresh token when revoke fails")
}

// When a token passes the active-lookup but revoke reports 0 rows, it is being
// reused (already rotated by a concurrent request). Rotation must refuse AND
// revoke every session for the user so a replayed stolen token cannot survive.
func TestValidateAndRotateRefreshTokenDetectsReuse(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	createCalled := false
	revokeAllUser := uint(0)
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{UserID: 11, Token: token}, nil
		},
		revokeByTokenIfActiveFn: func(ctx context.Context, token string) (bool, error) {
			return false, nil // lost the race: already revoked
		},
		revokeAllByUserIDFn: func(ctx context.Context, userID uint) error {
			revokeAllUser = userID
			return nil
		},
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			createCalled = true
			return nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res)
	require.False(t, createCalled, "must not mint a replacement token on reuse")
	require.Equal(t, uint(11), revokeAllUser, "must revoke all sessions for the user on reuse")
}

// Rotation must wrap the revoke-old + mint-new pair in a single transaction so
// the two commit or roll back together. This asserts (a) the work goes through
// the transaction manager and (b) a mint failure surfaces as an error rather
// than a half-rotated session (old token dead, no replacement).
func TestValidateAndRotateRefreshTokenIsTransactional(t *testing.T) {
	cfg := setupConfig(t)
	setupLogger(t)

	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{UserID: 11, Token: token}, nil
		},
		revokeByTokenIfActiveFn: func(ctx context.Context, token string) (bool, error) {
			return true, nil
		},
		createFn: func(ctx context.Context, entity *models.RefreshToken) error {
			return errors.New("mint failed") // forces the transaction to unwind
		},
	}

	tx := &recordingTxManager{}
	a, err := NewAuthJWT(cfg, userRepo, refreshRepo, &mockLogRepository{}, tx)
	require.NoError(t, err)

	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res)
	require.True(t, tx.called, "revoke + mint must run inside a transaction")
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
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)

	a.loginResponse(c, &core.Token{AccessToken: "access"})

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
	require.Equal(t, "internal error", body["message"])
}
