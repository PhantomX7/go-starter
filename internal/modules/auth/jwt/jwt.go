// internal/modules/auth/jwt/jwt.go
package authjwt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	IdentityKey    = "user_id"
	RoleKey        = "role"
	AdminRoleIDKey = "admin_role_id"
	authUserKey    = "auth_user"
)

type AuthJWT struct {
	Middleware       *ginjwt.GinJWTMiddleware
	userRepo         userrepo.UserRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
	logRepository    logRepository.LogRepository
}

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
	if user, ok := data.(*models.User); ok {
		claims := jwt.MapClaims{
			IdentityKey: user.ID,
			RoleKey:     user.Role.ToString(),
			"sub":       user.ID,
			"iss":       config.Get().JWT.Issuer,
		}

		// Include admin_role_id if present
		if user.AdminRoleID != nil {
			claims[AdminRoleIDKey] = *user.AdminRoleID
		}

		return claims
	}
	return jwt.MapClaims{}
}

func (a *AuthJWT) identityHandler(c *gin.Context) any {
	claims := ginjwt.ExtractClaims(c)
	userID, _ := claims[IdentityKey].(float64)
	role, _ := claims[RoleKey].(string)

	// Extract admin_role_id (optional)
	var adminRoleID *uint
	if val, ok := claims[AdminRoleIDKey]; ok && val != nil {
		id := uint(val.(float64))
		adminRoleID = &id
	}

	return &models.User{
		ID:          uint(userID),
		Role:        models.UserRole(role),
		AdminRoleID: adminRoleID,
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

	c.Set(authUserKey, user)
	logger.Info("Login successful", zap.Uint("user_id", user.ID))
	return user, nil
}

func (a *AuthJWT) authorizer(c *gin.Context, data any) bool {
	user, ok := data.(*models.User)
	if !ok || user.ID == 0 {
		return false
	}

	ctx := c.Request.Context()

	// Validate user status and session
	dbUser, err := a.userRepo.FindById(ctx, user.ID)
	if err != nil || !dbUser.IsActive {
		return false
	}

	count, err := a.refreshTokenRepo.GetValidCountByUserID(ctx, user.ID)
	if err != nil || count == 0 {
		return false
	}

	// Set context values before c.Next() is called
	// Use dbUser to get the latest admin_role_id from database
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

	refreshToken, err := a.createRefreshToken(c.Request.Context(), user.ID)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Uint("user_id", user.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("failed to generate token"))
		return
	}

	// Log admin login
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

func (a *AuthJWT) GenerateTokensForUser(ctx context.Context, user *models.User) (*dto.AuthResponse, error) {
	token, err := a.Middleware.TokenGenerator(ctx, user)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to generate access token", err)
	}

	refreshToken, err := a.createRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to generate refresh token", err)
	}

	return &dto.AuthResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}, nil
}

func (a *AuthJWT) ValidateAndRotateRefreshToken(ctx context.Context, oldToken string) (*dto.AuthResponse, error) {
	tokenRecord, err := a.refreshTokenRepo.FindByToken(ctx, oldToken)
	if err != nil {
		return nil, cerrors.NewBadRequestError("invalid or expired refresh token")
	}

	user, err := a.userRepo.FindById(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, cerrors.NewBadRequestError("user account is inactive")
	}

	// Revoke old token (ignore error, continue with new token)
	_ = a.refreshTokenRepo.RevokeByToken(ctx, oldToken)

	return a.GenerateTokensForUser(ctx, user)
}

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

func (a *AuthJWT) RevokeAllUserTokensExcept(ctx context.Context, userID uint, exceptToken string) error {
	return a.refreshTokenRepo.RevokeAllByUserIDExcept(ctx, userID, exceptToken)
}

// --- Private Helpers ---

func (a *AuthJWT) validateCredentials(ctx context.Context, username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	dummyHash := []byte("$2a$12$dummy.hash.to.prevent.timing.attacks")

	var user *models.User
	var err error

	if strings.Contains(username, "@") {
		user, err = a.userRepo.FindByEmail(ctx, strings.ToLower(username))
	} else {
		user, err = a.userRepo.FindByUsername(ctx, strings.ToLower(username))
	}

	if err != nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, err
	}

	if !user.IsActive {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, errors.New("inactive account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	return user, nil
}

func (a *AuthJWT) createRefreshToken(ctx context.Context, userID uint) (string, error) {
	token := uuid.New().String() + "-" + uuid.New().String()

	err := a.refreshTokenRepo.Create(ctx, &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(config.Get().JWT.RefreshExpiration),
	})

	return token, err
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
