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
func (m *mockUserRepository) FindByIDForUpdate(context.Context, uint) (*models.User, error) {
	panic("unexpected FindByIDForUpdate call")
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
	createFn                     func(context.Context, *models.RefreshToken) error
	findByTokenFn                func(context.Context, string) (*models.RefreshToken, error)
	findActiveByIDFn             func(context.Context, uuid.UUID) (*models.RefreshToken, error)
	getValidCountByUserIDFn      func(context.Context, uint) (int64, error)
	revokeByTokenFn              func(context.Context, string) error
	revokeByTokenIfActiveFn      func(context.Context, string) (bool, error)
	revokeAllByUserIDFn          func(context.Context, uint) error
	revokeAllByUserIDExceptFn    func(context.Context, uint, string) error
	revokeOldestActiveByUserIDFn func(context.Context, uint, int) error
	updateTokenHashIfActiveFn    func(context.Context, string, string) (bool, error)
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
func (m *mockRefreshTokenRepository) RevokeOldestActiveByUserID(ctx context.Context, userID uint, n int) error {
	if m.revokeOldestActiveByUserIDFn == nil {
		panic("unexpected RevokeOldestActiveByUserID call")
	}
	return m.revokeOldestActiveByUserIDFn(ctx, userID, n)
}
func (m *mockRefreshTokenRepository) UpdateTokenHashIfActive(ctx context.Context, oldToken, newToken string) (bool, error) {
	if m.updateTokenHashIfActiveFn == nil {
		panic("unexpected UpdateTokenHashIfActive call")
	}
	return m.updateTokenHashIfActiveFn(ctx, oldToken, newToken)
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
	// The production default is a nonzero cap; disable it here so tests that
	// are not about the session cap need no count/revoke mocks. Cap tests set
	// cfg.JWT.MaxActiveSessions explicitly.
	t.Setenv("JWT_MAX_ACTIVE_SESSIONS", "0")
	t.Setenv("APP_NAME", "Athleton Test")
	t.Setenv("APP_ENVIRONMENT", "development")
	// Config validation requires an explicit, non-weak admin default password.
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "test-default-password-123")

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

// Regression test for the prior panic: a forged or malformed admin-role claim
// of the wrong JSON type hit an unchecked val.(float64) assertion. It must be
// skipped instead — never trusted, never a crash.
func TestIdentityHandlerSkipsAdminRoleIDWithWrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	a := &AuthJWT{}

	cases := map[string]any{
		"string": "3",
		"bool":   true,
		"nil":    nil,
	}

	for name, val := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Set("JWT_PAYLOAD", jwt.MapClaims{
				IdentityKey:    float64(7),
				RoleKey:        "admin",
				AdminRoleIDKey: val,
			})

			var subj *authSubject
			var ok bool
			require.NotPanics(t, func() {
				subj, ok = a.identityHandler(c).(*authSubject)
			})

			require.True(t, ok)
			require.Nil(t, subj.User.AdminRoleID)
		})
	}
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

// With JWT_MAX_ACTIVE_SESSIONS unset (0) the cap is disabled: token creation
// must neither count sessions nor revoke anything — the mock panics on any
// unexpected GetValidCountByUserID / RevokeOldestActiveByUserID call, so the
// happy path above already proves this; this test makes the contract explicit.
func TestSessionCapDisabledSkipsCountAndRevocation(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		createFn: func(context.Context, *models.RefreshToken) error { return nil },
	}

	a := newAuthJWT(t, &mockUserRepository{}, refreshRepo, &mockLogRepository{})
	a.cfg.JWT.MaxActiveSessions = 0

	_, err := a.GenerateTokensForUser(context.Background(), &models.User{ID: 7, Role: models.UserRoleUser})

	require.NoError(t, err)
}

// When the user is at or over the cap, enough of the oldest active sessions
// must be revoked BEFORE the new session is created that the new login fits.
func TestSessionCapRevokesOldestSessionsToFitNewLogin(t *testing.T) {
	cases := map[string]struct {
		maxSessions  int
		activeCount  int64
		wantRevoked  int
		expectRevoke bool
	}{
		"under cap leaves sessions alone":  {maxSessions: 2, activeCount: 1, expectRevoke: false},
		"at cap revokes the oldest":        {maxSessions: 2, activeCount: 2, wantRevoked: 1, expectRevoke: true},
		"over cap revokes enough to fit":   {maxSessions: 2, activeCount: 5, wantRevoked: 4, expectRevoke: true},
		"cap of one keeps a single device": {maxSessions: 1, activeCount: 1, wantRevoked: 1, expectRevoke: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			revokedN := 0
			revokeCalled := false
			createCalled := false
			refreshRepo := &mockRefreshTokenRepository{
				getValidCountByUserIDFn: func(ctx context.Context, userID uint) (int64, error) {
					require.Equal(t, uint(7), userID)
					return tc.activeCount, nil
				},
				revokeOldestActiveByUserIDFn: func(ctx context.Context, userID uint, n int) error {
					require.Equal(t, uint(7), userID)
					require.False(t, createCalled, "oldest sessions must be revoked before the new one is created")
					revokeCalled = true
					revokedN = n
					return nil
				},
				createFn: func(ctx context.Context, entity *models.RefreshToken) error {
					createCalled = true
					return nil
				},
			}

			a := newAuthJWT(t, &mockUserRepository{}, refreshRepo, &mockLogRepository{})
			a.cfg.JWT.MaxActiveSessions = tc.maxSessions

			res, err := a.GenerateTokensForUser(context.Background(), &models.User{ID: 7, Role: models.UserRoleUser})

			require.NoError(t, err)
			require.NotEmpty(t, res.RefreshToken)
			require.True(t, createCalled)
			require.Equal(t, tc.expectRevoke, revokeCalled)
			if tc.expectRevoke {
				require.Equal(t, tc.wantRevoked, revokedN)
			}
		})
	}
}

// If the cap cannot be enforced (count or revoke fails), token creation must
// fail rather than silently minting an uncapped session.
func TestSessionCapFailureBlocksTokenCreation(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		getValidCountByUserIDFn: func(context.Context, uint) (int64, error) {
			return 0, cerrors.NewInternalServerError("count boom", errors.New("boom"))
		},
	}

	a := newAuthJWT(t, &mockUserRepository{}, refreshRepo, &mockLogRepository{})
	a.cfg.JWT.MaxActiveSessions = 2

	res, err := a.GenerateTokensForUser(context.Background(), &models.User{ID: 7, Role: models.UserRoleUser})

	require.Error(t, err)
	require.Nil(t, res)
}

// Rotation is in-place: the SAME row is rewritten with a fresh token hash, so
// the session ID (jti) survives the refresh and in-flight access tokens stay
// valid. This asserts (a) a new refresh token distinct from the old one is
// returned, (b) the access token carries the ORIGINAL session ID as jti, and
// (c) no new row is created (the mock panics on an unexpected Create).
func TestValidateAndRotateRefreshTokenRotatesInPlace(t *testing.T) {
	sessionID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(11), id)
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	var rotatedTo string
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "old-token", token)
			return &models.RefreshToken{ID: sessionID, UserID: 11, Token: token}, nil
		},
		updateTokenHashIfActiveFn: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			require.Equal(t, "old-token", oldToken)
			require.NotEmpty(t, newToken)
			rotatedTo = newToken
			return true, nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.NoError(t, err)
	require.NotEmpty(t, res.AccessToken)
	require.Equal(t, rotatedTo, res.RefreshToken, "the returned refresh token must be the value the row was rotated to")
	require.NotEqual(t, "old-token", res.RefreshToken)

	// The new access token must be bound to the ORIGINAL session: its jti claim
	// carries the untouched row ID, keeping older access tokens for this
	// session valid too.
	parsed, err := jwt.Parse(res.AccessToken, func(*jwt.Token) (any, error) {
		return []byte("test-secret-of-at-least-32-characters"), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	require.Equal(t, sessionID.String(), claims[SessionIDKey])
}

// Regression test for the prior bug where rotation-store errors were ignored
// and a fresh token pair was minted anyway, leaving the old refresh token
// reusable. (Mint = TokenGenerator; the mock panics on an unexpected Create.)
func TestValidateAndRotateRefreshTokenFailsWhenHashSwapFails(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		updateTokenHashIfActiveFn: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, cerrors.NewInternalServerError("db boom", errors.New("boom"))
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res, "must not hand out new tokens when the hash swap fails")
}

// When a token passes the active-lookup but the hash swap matches 0 rows, it
// is being reused (already rotated by a concurrent request). Rotation must
// refuse AND revoke every session for the user so a replayed stolen token
// cannot survive.
func TestValidateAndRotateRefreshTokenDetectsReuse(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	revokeAllUser := uint(0)
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		updateTokenHashIfActiveFn: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, nil // lost the race: hash already rotated away
		},
		revokeAllByUserIDFn: func(ctx context.Context, userID uint) error {
			revokeAllUser = userID
			return nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &mockLogRepository{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res, "must not hand out new tokens on reuse")
	require.Equal(t, uint(11), revokeAllUser, "must revoke all sessions for the user on reuse")
}

// Rotation must wrap the hash-swap + access-token mint in a single transaction
// so the two commit or roll back together (a mint failure after a committed
// swap would leave the client with a dead old token and no replacement). This
// asserts the work goes through the transaction manager and that a failure
// inside it surfaces as an error rather than a half-rotated session.
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
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		updateTokenHashIfActiveFn: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, errors.New("swap failed") // forces the transaction to unwind
		},
	}

	tx := &recordingTxManager{}
	a, err := NewAuthJWT(cfg, userRepo, refreshRepo, &mockLogRepository{}, tx)
	require.NoError(t, err)

	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res)
	require.True(t, tx.called, "hash swap + mint must run inside a transaction")
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

// An ownership mismatch must be indistinguishable from a nonexistent token:
// a distinct error would let a caller probe whether an arbitrary token value
// is active for SOME user (a validity oracle). The token must be left alone —
// the mock panics if RevokeByToken were called.
func TestRevokeRefreshTokenSilentlyIgnoresOtherUsersToken(t *testing.T) {
	setupLogger(t)

	refreshRepo := &mockRefreshTokenRepository{
		findByTokenFn: func(context.Context, string) (*models.RefreshToken, error) {
			return &models.RefreshToken{UserID: 99}, nil
		},
	}

	a := &AuthJWT{refreshTokenRepo: refreshRepo}
	err := a.RevokeRefreshToken(context.Background(), "token", 1)

	require.NoError(t, err, "ownership mismatch must look exactly like not-found")
}

func TestLoginResponseCreatesAuditLogForPrivilegedRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupLogger(t)

	// Root is the most privileged account; its logins must be audited exactly
	// like admin logins.
	for _, role := range []models.UserRole{models.UserRoleAdmin, models.UserRoleRoot} {
		t.Run(string(role), func(t *testing.T) {
			created := make(chan *models.Log, 1)
			a := &AuthJWT{logRepository: &mockLogRepository{
				createFn: func(_ context.Context, entry *models.Log) error {
					created <- entry
					return nil
				},
			}}

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)
			c.Set(AuthUserKey, &models.User{ID: 3, Name: "Privileged", Role: role})
			c.Set(authRefreshTokenKey, "refresh-token")

			a.loginResponse(c, &core.Token{AccessToken: "access"})
			require.Equal(t, http.StatusOK, rec.Code)

			select {
			case entry := <-created:
				require.Equal(t, models.LogActionLogin, entry.Action)
				require.Equal(t, models.LogEntityTypeUser, entry.EntityType)
				require.Equal(t, uint(3), entry.EntityID)
			case <-time.After(2 * time.Second):
				t.Fatalf("no login audit log written for role %s", role)
			}
		})
	}
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
