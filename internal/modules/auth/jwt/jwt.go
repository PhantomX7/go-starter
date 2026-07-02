// Package authjwt contains the JWT authentication integration for the API.
package authjwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	rtokenrepo "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepo "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/config"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	ginjwt "github.com/appleboy/gin-jwt/v3"
	"github.com/appleboy/gin-jwt/v3/core"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// IdentityKey stores the authenticated user ID in claims and Gin context.
	IdentityKey = "user_id"
	// RoleKey stores the authenticated role in claims and Gin context.
	RoleKey = "role"
	// AdminRoleIDKey stores the authenticated admin-role ID in claims and Gin context.
	AdminRoleIDKey = "admin_role_id"
	// SessionIDKey stores the refresh-token session identifier in claims.
	SessionIDKey = "jti"

	// AuthUserKey is the gin-context key under which the authorizer stores the
	// freshly loaded *models.User for downstream middleware (e.g. the
	// must-change-default-password gate) — saving them a second DB lookup.
	AuthUserKey = "auth_user"

	authRefreshTokenKey = "auth_refresh_token" // #nosec G101 -- identifier name, not a credential

	// dummyBcryptCost matches the production bcrypt cost (see service.BcryptCost)
	// so the timing-equalization path costs the same as a real comparison.
	dummyBcryptCost = 12
)

// dummyHash is a real bcrypt hash generated lazily on first use. It is fed to
// bcrypt.CompareHashAndPassword on the username-not-found / inactive-user
// paths so the response time matches a real wrong-password comparison and
// account existence is not leaked via timing. The previous string literal was
// not a valid bcrypt hash, so CompareHashAndPassword returned ErrHashTooShort
// immediately and did no work.
var dummyHash = sync.OnceValue(func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("athleton-timing-dummy-password"), dummyBcryptCost)
	if err != nil {
		panic(fmt.Sprintf("authjwt: failed to generate dummy bcrypt hash: %v", err))
	}
	return h
})

// authSubject is the value passed between Authenticator → payloadFunc and
// identityHandler → authorizer. It binds the access JWT to a specific
// refresh-token session: revoking that refresh token also kills the access
// tokens minted alongside it.
type authSubject struct {
	User      *models.User
	SessionID uuid.UUID
}

// errRefreshTokenReuse is an internal sentinel used to unwind the rotation
// transaction when a reused (already-revoked) token is detected. It never
// escapes ValidateAndRotateRefreshToken — the caller sees a generic bad-request
// error instead.
var errRefreshTokenReuse = errors.New("refresh token reuse detected")

// AuthJWT bundles the gin-jwt middleware with the repositories it depends on.
type AuthJWT struct {
	Middleware       *ginjwt.GinJWTMiddleware
	cfg              *config.Config
	userRepo         userrepo.UserRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
	logRepository    logRepository.LogRepository
	txManager        transaction_manager.TransactionManager
}

// NewAuthJWT constructs the JWT authentication middleware and its helpers.
func NewAuthJWT(
	cfg *config.Config,
	userRepo userrepo.UserRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
	logRepository logRepository.LogRepository,
	txManager transaction_manager.TransactionManager,
) (*AuthJWT, error) {
	// Force dummy-hash generation now so a bcrypt failure surfaces as a boot
	// error instead of a panic on the first login attempt.
	_ = dummyHash()

	a := &AuthJWT{
		cfg:              cfg,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		logRepository:    logRepository,
		txManager:        txManager,
	}

	middleware, err := ginjwt.New(&ginjwt.GinJWTMiddleware{
		Realm:            cfg.App.Name,
		Key:              []byte(cfg.JWT.Secret),
		Timeout:          cfg.JWT.Expiration,
		MaxRefresh:       cfg.JWT.RefreshExpiration,
		IdentityKey:      IdentityKey,
		SigningAlgorithm: "HS256",
		TokenLookup:      "header: Authorization",
		TokenHeadName:    "Bearer",
		TimeFunc:         time.Now,
		SendCookie:       false,
		SecureCookie:     cfg.IsProduction(),
		CookieHTTPOnly:   true,
		CookieSameSite:   http.SameSiteStrictMode,

		PayloadFunc:     a.payloadFunc,
		IdentityHandler: a.identityHandler,
		Authenticator:   a.authenticator,
		Authorizer:      a.authorizer,
		Unauthorized:    a.unauthorized,
		LoginResponse:   a.loginResponse,
		LogoutResponse:  a.logoutResponse,
	})

	if err != nil {
		return nil, err
	}

	if err := middleware.MiddlewareInit(); err != nil {
		return nil, err
	}

	a.Middleware = middleware
	return a, nil
}

// --- Middleware Callbacks ---

func (a *AuthJWT) payloadFunc(data any) jwt.MapClaims {
	subj, ok := data.(*authSubject)
	if !ok || subj.User == nil {
		return jwt.MapClaims{}
	}

	user := subj.User
	claims := jwt.MapClaims{
		IdentityKey:  user.ID,
		RoleKey:      user.Role.ToString(),
		SessionIDKey: subj.SessionID.String(),
		"sub":        user.ID,
		"iss":        a.cfg.JWT.Issuer,
	}

	if user.AdminRoleID != nil {
		claims[AdminRoleIDKey] = *user.AdminRoleID
	}

	return claims
}

func (a *AuthJWT) identityHandler(c *gin.Context) any {
	claims := ginjwt.ExtractClaims(c)
	userID, _ := claims[IdentityKey].(float64)
	role, _ := claims[RoleKey].(string)

	var adminRoleID *uint
	// Comma-ok assertion: JSON numbers decode as float64, but a forged or
	// malformed claim could carry any type — skip it rather than panic.
	if val, ok := claims[AdminRoleIDKey].(float64); ok {
		id := uint(val)
		adminRoleID = &id
	}

	var sessionID uuid.UUID
	if jtiStr, ok := claims[SessionIDKey].(string); ok {
		if parsed, err := uuid.Parse(jtiStr); err == nil {
			sessionID = parsed
		}
	}

	return &authSubject{
		User: &models.User{
			ID:          uint(userID),
			Role:        models.UserRole(role),
			AdminRoleID: adminRoleID,
		},
		SessionID: sessionID,
	}
}

func (a *AuthJWT) authenticator(c *gin.Context) (any, error) {
	var req dto.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		return nil, ginjwt.ErrMissingLoginValues
	}

	user, err := a.validateCredentials(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		return nil, ginjwt.ErrFailedAuthentication
	}

	// Pre-create the refresh-token session so the access JWT can carry its
	// ID as the jti claim. The authorizer will look this session up on every
	// request, so revoking it kills the matching access tokens too.
	refreshTokenStr, sessionID, err := a.createRefreshToken(c.Request.Context(), user.ID)
	if err != nil {
		logger.Error("Failed to create refresh token at login", zap.Uint("user_id", user.ID), zap.Error(err))
		return nil, ginjwt.ErrFailedAuthentication
	}

	subj := &authSubject{User: user, SessionID: sessionID}
	c.Set(AuthUserKey, user)
	c.Set(authRefreshTokenKey, refreshTokenStr)
	logger.Info("Login successful", zap.Uint("user_id", user.ID))
	return subj, nil
}

func (a *AuthJWT) authorizer(c *gin.Context, data any) bool {
	subj, ok := data.(*authSubject)
	if !ok || subj.User == nil || subj.User.ID == 0 || subj.SessionID == uuid.Nil {
		return false
	}

	ctx := c.Request.Context()

	dbUser, err := a.userRepo.FindByID(ctx, subj.User.ID)
	if err != nil || !dbUser.IsActive {
		return false
	}

	// Per-session check: the refresh-token row whose ID equals the access
	// token's jti claim must still be active and belong to this user. Once
	// that row is revoked (logout, change-password, admin action), every
	// access token minted for that session stops working immediately.
	session, err := a.refreshTokenRepo.FindActiveByID(ctx, subj.SessionID)
	if err != nil || session.UserID != subj.User.ID {
		return false
	}

	a.setContextValues(c, dbUser.ID, dbUser.Name, string(dbUser.Role), dbUser.AdminRoleID)
	// Expose the loaded user so later middleware (e.g. RequirePasswordChanged)
	// can inspect fields like PasswordChangedAt without another DB query.
	c.Set(AuthUserKey, dbUser)
	return true
}

func (a *AuthJWT) unauthorized(c *gin.Context, code int, message string) {
	c.JSON(code, response.BuildResponseFailed(message))
}

func (a *AuthJWT) loginResponse(c *gin.Context, token *core.Token) {
	userData, _ := c.Get(AuthUserKey)
	user, ok := userData.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("internal error"))
		return
	}

	refreshTokenData, _ := c.Get(authRefreshTokenKey)
	refreshToken, ok := refreshTokenData.(string)
	if !ok || refreshToken == "" {
		c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("internal error"))
		return
	}

	if user.Role == models.UserRoleAdmin {
		a.createLoginLog(user)
	}

	c.JSON(http.StatusOK, response.BuildResponseSuccess("login success", dto.AuthResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}))
}

func (a *AuthJWT) logoutResponse(c *gin.Context) {
	c.JSON(http.StatusOK, response.BuildResponseSuccess("logout successful", nil))
}

// --- Public Methods ---

// GenerateTokensForUser mints a new access/refresh token pair for user.
func (a *AuthJWT) GenerateTokensForUser(ctx context.Context, user *models.User) (*dto.AuthResponse, error) {
	// Refresh token first so the access token can carry its session ID as jti.
	refreshTokenStr, sessionID, err := a.createRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to generate refresh token", err)
	}

	token, err := a.Middleware.TokenGenerator(ctx, &authSubject{User: user, SessionID: sessionID})
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to generate access token", err)
	}

	return &dto.AuthResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshTokenStr,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAndRotateRefreshToken rotates oldToken in place and returns a fresh
// token pair bound to the SAME session. Rotation swaps only the stored token
// hash on the existing row: the row ID is preserved so access tokens carrying
// it as their jti claim keep working across a refresh, and expires_at is
// preserved so the session has an absolute lifetime of JWT_REFRESH_EXPIRATION
// from login instead of sliding forever one refresh at a time.
func (a *AuthJWT) ValidateAndRotateRefreshToken(ctx context.Context, oldToken string) (*dto.AuthResponse, error) {
	tokenRecord, err := a.refreshTokenRepo.FindByToken(ctx, oldToken)
	if err != nil {
		return nil, cerrors.NewBadRequestError("invalid or expired refresh token")
	}

	user, err := a.userRepo.FindByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, cerrors.NewBadRequestError("user account is inactive")
	}

	newToken := newRefreshTokenValue()

	// Rotate atomically: swapping the hash and minting the access token must
	// either both commit or both roll back. Outside a transaction, a mint
	// failure after the hash swap committed would leave the user with a dead
	// old token and an undelivered replacement — a silent logout that forces
	// re-authentication.
	var resp *dto.AuthResponse
	reuseDetected := false
	err = a.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Swap the hash before minting anything. If the swap fails we must NOT
		// hand out new tokens: the old token would remain presentable and
		// rotation would silently degrade to "issue extra tokens", letting an
		// attacker who captured a refresh token replay it indefinitely while
		// the store is unhealthy.
		updated, err := a.refreshTokenRepo.UpdateTokenHashIfActive(txCtx, oldToken, newToken)
		if err != nil {
			logger.Error("Failed to rotate refresh token hash",
				zap.Uint("user_id", user.ID), zap.Error(err))
			return cerrors.NewInternalServerError("failed to rotate refresh token", err)
		}

		// The token was active at FindByToken but its hash no longer matched by
		// the time we tried to swap it: another request already rotated it (or a
		// revocation landed in between), i.e. this token is being reused. This is
		// the same detection window the previous revoke-and-replace rotation had —
		// there, too, a long-rotated token failed the active FindByToken lookup
		// with a generic error, and only race-window reuse tripped the alarm.
		// Flag it and unwind the transaction — the session-wide revocation below
		// MUST commit, so it cannot run inside this rolled-back tx. (The swap
		// above was a no-op since no row matched, so there is nothing of value to
		// roll back here.)
		if !updated {
			reuseDetected = true
			return errRefreshTokenReuse
		}

		// Mint the access token with the EXISTING session ID as jti so tokens
		// issued before this refresh stay valid: the authorizer resolves jti to
		// this same still-active row.
		token, err := a.Middleware.TokenGenerator(txCtx, &authSubject{User: user, SessionID: tokenRecord.ID})
		if err != nil {
			return cerrors.NewInternalServerError("failed to generate access token", err)
		}

		resp = &dto.AuthResponse{
			AccessToken:  token.AccessToken,
			RefreshToken: newToken,
			TokenType:    "Bearer",
		}
		return nil
	})

	// Reuse is a breach: revoke every session for the user so a stolen-then-
	// replayed token cannot outlive detection, then refuse. Run outside the
	// (rolled-back) transaction so the revocation persists.
	if reuseDetected {
		logger.Warn("Refresh-token reuse detected during rotation; revoking all sessions",
			zap.Uint("user_id", user.ID))
		if err := a.refreshTokenRepo.RevokeAllByUserID(ctx, user.ID); err != nil {
			logger.Error("Failed to revoke sessions after refresh-token reuse",
				zap.Uint("user_id", user.ID), zap.Error(err))
		}
		return nil, cerrors.NewBadRequestError("invalid or expired refresh token")
	}

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// RevokeRefreshToken revokes token when it belongs to userID.
func (a *AuthJWT) RevokeRefreshToken(ctx context.Context, token string, userID uint) error {
	tokenRecord, err := a.refreshTokenRepo.FindByToken(ctx, token)
	if errors.Is(err, cerrors.ErrNotFound) {
		return nil // Idempotent
	}
	if err != nil {
		return err
	}

	// Ownership mismatch must be indistinguishable from not-found: returning a
	// distinct error would let a caller probe whether an arbitrary token value
	// is currently active for SOME user (a validity oracle). Log server-side
	// instead — a mismatch here means someone is submitting tokens that are not
	// theirs — and leave the token untouched.
	if tokenRecord.UserID != userID {
		logger.Warn("Refresh token revocation attempted by non-owner",
			zap.Uint("acting_user_id", userID),
			zap.Uint("owner_user_id", tokenRecord.UserID))
		return nil
	}

	return a.refreshTokenRepo.RevokeByToken(ctx, token)
}

// RevokeAllUserTokensExcept revokes every active token for userID except exceptToken.
func (a *AuthJWT) RevokeAllUserTokensExcept(ctx context.Context, userID uint, exceptToken string) error {
	return a.refreshTokenRepo.RevokeAllByUserIDExcept(ctx, userID, exceptToken)
}

// --- Private Helpers ---

func (a *AuthJWT) validateCredentials(ctx context.Context, username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)

	var user *models.User
	var err error

	if strings.Contains(username, "@") {
		user, err = a.userRepo.FindByEmail(ctx, strings.ToLower(username))
	} else {
		user, err = a.userRepo.FindByUsername(ctx, strings.ToLower(username))
	}

	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return nil, err
	}

	if !user.IsActive {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return nil, errors.New("inactive account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	return user, nil
}

// newRefreshTokenValue returns a fresh opaque refresh-token wire value. Two
// concatenated UUIDv4s give ~244 bits of entropy — far beyond brute-force
// reach — and only the SHA-256 hash of this value is ever stored at rest.
func newRefreshTokenValue() string {
	return uuid.New().String() + "-" + uuid.New().String()
}

// createRefreshToken inserts a new refresh-token row and returns both the
// opaque token string (handed to the client) and the row's UUID, which the
// caller embeds in the access JWT as the jti claim to bind it to this session.
func (a *AuthJWT) createRefreshToken(ctx context.Context, userID uint) (string, uuid.UUID, error) {
	if err := a.enforceSessionCap(ctx, userID); err != nil {
		return "", uuid.Nil, err
	}

	sessionID := uuid.New()
	token := newRefreshTokenValue()

	err := a.refreshTokenRepo.Create(ctx, &models.RefreshToken{
		ID:        sessionID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(a.cfg.JWT.RefreshExpiration),
	})
	if err != nil {
		return "", uuid.Nil, err
	}

	return token, sessionID, nil
}

// enforceSessionCap keeps a user's concurrent active sessions at or below
// JWT_MAX_ACTIVE_SESSIONS by revoking the oldest active session(s) so the
// session about to be created fits. A cap of 0 disables the check. Enforcing
// the cap at creation time (login/register) — rather than on refresh — means
// rotation can never be blocked by it, and an attacker with stolen
// credentials cannot mint unbounded parallel sessions.
func (a *AuthJWT) enforceSessionCap(ctx context.Context, userID uint) error {
	maxSessions := a.cfg.JWT.MaxActiveSessions
	if maxSessions <= 0 {
		return nil
	}

	count, err := a.refreshTokenRepo.GetValidCountByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Revoke enough of the oldest sessions that count+1 (this new session)
	// stays within the cap.
	overflow := int(count) - maxSessions + 1
	if overflow <= 0 {
		return nil
	}

	logger.Info("Active session cap reached; revoking oldest session(s)",
		zap.Uint("user_id", userID),
		zap.Int64("active_sessions", count),
		zap.Int("max_active_sessions", maxSessions),
		zap.Int("revoking", overflow))
	return a.refreshTokenRepo.RevokeOldestActiveByUserID(ctx, userID, overflow)
}

func (a *AuthJWT) setContextValues(c *gin.Context, userID uint, userName string, role string, adminRoleID *uint) {
	ctx := utils.NewContextWithValues(c.Request.Context(), utils.ContextValues{
		UserID:      userID,
		UserName:    userName,
		Role:        role,
		AdminRoleID: adminRoleID,
		RequestID:   utils.GetRequestIDFromContext(c.Request.Context()),
	})
	c.Request = c.Request.WithContext(ctx)
	c.Set("user_id", userID)
	c.Set("role", role)
	if adminRoleID != nil {
		c.Set("admin_role_id", *adminRoleID)
	}
}

// createLoginLog creates an audit log entry for admin login
func (a *AuthJWT) createLoginLog(user *models.User) {
	message := fmt.Sprintf("%s logged in", user.Name)

	log := &models.Log{
		UserID:     &user.ID,
		Action:     models.LogActionLogin,
		EntityType: models.LogEntityTypeUser,
		EntityID:   user.ID,
		Message:    message,
	}

	// Tracked by audit.Drain so graceful shutdown waits for the write.
	audit.Go(func() {
		if err := a.logRepository.Create(context.Background(), log); err != nil {
			logger.Error("Failed to create login audit log",
				zap.String("entity_type", models.LogEntityTypeUser),
				zap.Uint("entity_id", user.ID),
				zap.String("action", string(models.LogActionLogin)),
				zap.Error(err),
			)
		}
	})
}
