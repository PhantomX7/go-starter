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
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	refreshtokenmocks "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository/mocks"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	usermocks "github.com/PhantomX7/athleton/internal/modules/user/repository/mocks"
	txmocks "github.com/PhantomX7/athleton/libs/transaction_manager/mocks"
	"github.com/PhantomX7/athleton/pkg/config"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
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

func newAuthJWT(t *testing.T, userRepo userrepository.UserRepository, refreshRepo refreshtokenrepository.RefreshTokenRepository, logRepo logrepository.LogRepository) *AuthJWT {
	t.Helper()
	cfg := setupConfig(t)
	setupLogger(t)

	auth, err := NewAuthJWT(cfg, userRepo, refreshRepo, logRepo, &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	})
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

	repo := &usermocks.UserRepositoryMock{
		FindByEmailFunc: func(ctx context.Context, email string) (*models.User, error) {
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

	repo := &usermocks.UserRepositoryMock{
		FindByUsernameFunc: func(context.Context, string) (*models.User, error) {
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
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
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
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindActiveByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
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
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindActiveByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
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
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: false, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindActiveByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
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
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 5, IsActive: true, Role: models.UserRoleUser}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindActiveByIDFunc: func(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
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
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		CreateFunc: func(ctx context.Context, entity *models.RefreshToken) error {
			require.NotNil(t, ctx)
			require.Equal(t, uint(7), entity.UserID)
			require.NotEmpty(t, entity.Token)
			require.True(t, entity.ExpiresAt.After(time.Now()))
			return nil
		},
	}

	a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})

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
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		CreateFunc: func(context.Context, *models.RefreshToken) error { return nil },
	}

	a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})
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
			refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
				GetValidCountByUserIDFunc: func(ctx context.Context, userID uint) (int64, error) {
					require.Equal(t, uint(7), userID)
					return tc.activeCount, nil
				},
				RevokeOldestActiveByUserIDFunc: func(ctx context.Context, userID uint, n int) error {
					require.Equal(t, uint(7), userID)
					require.False(t, createCalled, "oldest sessions must be revoked before the new one is created")
					revokeCalled = true
					revokedN = n
					return nil
				},
				CreateFunc: func(ctx context.Context, entity *models.RefreshToken) error {
					createCalled = true
					return nil
				},
			}

			a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})
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
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		GetValidCountByUserIDFunc: func(context.Context, uint) (int64, error) {
			return 0, cerrors.NewInternalServerError("count boom", errors.New("boom"))
		},
	}

	a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})
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
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, uint(11), id)
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	var rotatedTo string
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			require.Equal(t, "old-token", token)
			return &models.RefreshToken{ID: sessionID, UserID: 11, Token: token}, nil
		},
		UpdateTokenHashIfActiveFunc: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			require.Equal(t, "old-token", oldToken)
			require.NotEmpty(t, newToken)
			rotatedTo = newToken
			return true, nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &logmocks.LogRepositoryMock{})
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
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		UpdateTokenHashIfActiveFunc: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, cerrors.NewInternalServerError("db boom", errors.New("boom"))
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &logmocks.LogRepositoryMock{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res, "must not hand out new tokens when the hash swap fails")
}

// When a token passes the active-lookup but the hash swap matches 0 rows, it
// is being reused (already rotated by a concurrent request). Rotation must
// refuse AND revoke every session for the user so a replayed stolen token
// cannot survive.
func TestValidateAndRotateRefreshTokenDetectsReuse(t *testing.T) {
	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	revokeAllUser := uint(0)
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		UpdateTokenHashIfActiveFunc: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, nil // lost the race: hash already rotated away
		},
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			revokeAllUser = userID
			return nil
		},
	}

	a := newAuthJWT(t, userRepo, refreshRepo, &logmocks.LogRepositoryMock{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res, "must not hand out new tokens on reuse")
	require.Equal(t, uint(11), revokeAllUser, "must revoke all sessions for the user on reuse")
}

// A stolen token replayed AFTER it was rotated away no longer matches any
// active row (FindByToken fails), but its hash was recorded as the current
// row's predecessor. Rotation must recognize the replay as reuse via the
// previous-token lookup and revoke every session for the owning user — this is
// the sequential steal-then-rotate case the in-place rotation could not catch
// before, where the attacker who rotated first would otherwise keep a live
// session and the victim would just see a generic invalid-token error.
func TestValidateAndRotateRefreshTokenDetectsSupersededReuse(t *testing.T) {
	revokeAllUser := uint(0)
	prevLookedUp := ""
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return nil, cerrors.NewNotFoundError("invalid refresh token")
		},
		FindByPreviousTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			prevLookedUp = token
			return &models.RefreshToken{ID: uuid.New(), UserID: 11}, nil
		},
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			revokeAllUser = userID
			return nil
		},
	}

	a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "stolen-old-token")

	require.Error(t, err)
	require.Nil(t, res, "must not hand out new tokens on reuse")
	require.Equal(t, "stolen-old-token", prevLookedUp)
	require.Equal(t, uint(11), revokeAllUser, "a superseded-token replay must revoke all of the user's sessions")
}

// A token that is neither active nor a recorded predecessor is just an unknown
// value (expired-and-purged, garbage, wrong user). It must be rejected WITHOUT
// revoking anyone's sessions, so a stranger poking random tokens can't force a
// legitimate user to be logged out.
func TestValidateAndRotateRefreshTokenIgnoresUnknownToken(t *testing.T) {
	revokeCalled := false
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return nil, cerrors.NewNotFoundError("invalid refresh token")
		},
		FindByPreviousTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return nil, cerrors.NewNotFoundError("refresh token not found")
		},
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			revokeCalled = true
			return nil
		},
	}

	a := newAuthJWT(t, &usermocks.UserRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{})
	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "random-garbage")

	require.Error(t, err)
	require.Nil(t, res)
	require.False(t, revokeCalled, "an unknown token must not trigger a family-wide revocation")
}

// Rotation must wrap the hash-swap + access-token mint in a single transaction
// so the two commit or roll back together (a mint failure after a committed
// swap would leave the client with a dead old token and no replacement). This
// asserts the work goes through the transaction manager and that a failure
// inside it surfaces as an error rather than a half-rotated session.
func TestValidateAndRotateRefreshTokenIsTransactional(t *testing.T) {
	cfg := setupConfig(t)
	setupLogger(t)

	userRepo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			return &models.User{ID: 11, Role: models.UserRoleUser, IsActive: true}, nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(ctx context.Context, token string) (*models.RefreshToken, error) {
			return &models.RefreshToken{ID: uuid.New(), UserID: 11, Token: token}, nil
		},
		UpdateTokenHashIfActiveFunc: func(ctx context.Context, oldToken, newToken string) (bool, error) {
			return false, errors.New("swap failed") // forces the transaction to unwind
		},
	}

	txCalled := false
	tx := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			txCalled = true
			return fn(ctx)
		},
	}
	a, err := NewAuthJWT(cfg, userRepo, refreshRepo, &logmocks.LogRepositoryMock{}, tx)
	require.NoError(t, err)

	res, err := a.ValidateAndRotateRefreshToken(context.Background(), "old-token")

	require.Error(t, err)
	require.Nil(t, res)
	require.True(t, txCalled, "hash swap + mint must run inside a transaction")
}

func TestRevokeRefreshTokenIsIdempotentWhenTokenMissing(t *testing.T) {
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(context.Context, string) (*models.RefreshToken, error) {
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

	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		FindByTokenFunc: func(context.Context, string) (*models.RefreshToken, error) {
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
			a := &AuthJWT{logRepository: &logmocks.LogRepositoryMock{
				CreateFunc: func(_ context.Context, entry *models.Log) error {
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
