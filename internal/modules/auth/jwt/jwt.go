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

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	rtokenrepo "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepo "github.com/PhantomX7/athleton/internal/modules/user/repository"
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

	authUserKey         = "auth_user"
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

// AuthJWT bundles the gin-jwt middleware with the repositories it depends on.
type AuthJWT struct {
	Middleware       *ginjwt.GinJWTMiddleware
	userRepo         userrepo.UserRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
	logRepository    logRepository.LogRepository
}

// NewAuthJWT constructs the JWT authentication middleware and its helpers.
func NewAuthJWT(
	userRepo userrepo.UserRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
	logRepository logRepository.LogRepository,
) (*AuthJWT, error) {
	cfg := config.Get()

	a := &AuthJWT{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		logRepository:    logRepository,
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
		"iss":        config.Get().JWT.Issuer,
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
	if val, ok := claims[AdminRoleIDKey]; ok && val != nil {
		id := uint(val.(float64))
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
	c.Set(authUserKey, user)
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
	return true
}

func (a *AuthJWT) unauthorized(c *gin.Context, code int, message string) {
	c.JSON(code, response.BuildResponseFailed(message))
}

func (a *AuthJWT) loginResponse(c *gin.Context, token *core.Token) {
	userData, _ := c.Get(authUserKey)
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

// ValidateAndRotateRefreshToken rotates oldToken and returns a fresh token pair.
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

	// Revoke old token before issuing the new one. If revocation fails we
	// must NOT mint a replacement: the old token would remain reusable and
	// rotation would silently degrade to "issue extra tokens", letting an
	// attacker who captured a refresh token replay it indefinitely while the
	// store is unhealthy.
	if err := a.refreshTokenRepo.RevokeByToken(ctx, oldToken); err != nil {
		logger.Error("Failed to revoke refresh token during rotation",
			zap.Uint("user_id", user.ID), zap.Error(err))
		return nil, cerrors.NewInternalServerError("failed to rotate refresh token", err)
	}

	return a.GenerateTokensForUser(ctx, user)
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

	if tokenRecord.UserID != userID {
		return cerrors.NewForbiddenError("token does not belong to user")
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

// createRefreshToken inserts a new refresh-token row and returns both the
// opaque token string (handed to the client) and the row's UUID, which the
// caller embeds in the access JWT as the jti claim to bind it to this session.
func (a *AuthJWT) createRefreshToken(ctx context.Context, userID uint) (string, uuid.UUID, error) {
	sessionID := uuid.New()
	token := uuid.New().String() + "-" + uuid.New().String()

	err := a.refreshTokenRepo.Create(ctx, &models.RefreshToken{
		ID:        sessionID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(config.Get().JWT.RefreshExpiration),
	})
	if err != nil {
		return "", uuid.Nil, err
	}

	return token, sessionID, nil
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

	go func() {
		if err := a.logRepository.Create(context.Background(), log); err != nil {
			logger.Error("Failed to create login audit log",
				zap.String("entity_type", models.LogEntityTypeUser),
				zap.Uint("entity_id", user.ID),
				zap.String("action", string(models.LogActionLogin)),
				zap.Error(err),
			)
		}
	}()
}
