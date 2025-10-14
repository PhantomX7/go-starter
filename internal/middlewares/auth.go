package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/pkg/response"
	"github.com/PhantomX7/go-starter/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// AuthorizationHeader is the header key for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the expected prefix for bearer tokens
	BearerPrefix = "Bearer"
	// TokenTimeout defines the maximum time for token validation operations
	TokenTimeout = 5 * time.Second
)

// AuthHandle creates a JWT authentication middleware handler
// This middleware validates JWT tokens, checks user status, and sets user context
func (m *Middleware) AuthHandle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract and validate token from header
		token, err := m.extractTokenFromHeader(c)
		if err != nil {
			m.handleAuthError(c, http.StatusUnauthorized, err.Error())
			return
		}

		// Parse and validate JWT token
		claims, err := m.parseAndValidateToken(token)
		if err != nil {
			m.handleAuthError(c, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Create context with timeout for database operations
		ctx, cancel := context.WithTimeout(c.Request.Context(), TokenTimeout)
		defer cancel()

		// Validate user and session
		if err := m.validateUserAndSession(ctx, claims); err != nil {
			statusCode := http.StatusUnauthorized
			if err.Error() == "Internal server error" {
				statusCode = http.StatusInternalServerError
			}
			m.handleAuthError(c, statusCode, err.Error())
			return
		}

		// Set authenticated user context
		m.setUserContext(c, claims)

		c.Next()
	}
}

// extractTokenFromHeader extracts and validates the Bearer token from Authorization header
func (m *Middleware) extractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Validate Bearer token format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != BearerPrefix {
		return "", errors.New("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("token cannot be empty")
	}

	return token, nil
}

// parseAndValidateToken parses the JWT token and validates its claims
func (m *Middleware) parseAndValidateToken(tokenString string) (*models.AccessClaims, error) {
	claims := &models.AccessClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method - only allow HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.cfg.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Additional claims validation
	if claims.UserID == 0 {
		return nil, errors.New("invalid user ID in token")
	}

	if claims.Role == "" {
		return nil, errors.New("invalid role in token")
	}

	return claims, nil
}

// validateUserAndSession validates user existence, status, and session validity
func (m *Middleware) validateUserAndSession(ctx context.Context, claims *models.AccessClaims) error {
	// Check if user exists and is active
	user, err := m.userRepo.FindById(ctx, claims.UserID)
	if err != nil {
		return errors.New("invalid user")
	}

	if !user.IsActive {
		return errors.New("user is not active")
	}

	// Validate session by checking refresh token count
	count, err := m.refreshTokenRepo.GetValidCountByUserID(ctx, claims.UserID)
	if err != nil {
		return errors.New("internal server error")
	}

	if count == 0 {
		return errors.New("invalid session")
	}

	return nil
}

// setUserContext sets the authenticated user information in the request context
func (m *Middleware) setUserContext(c *gin.Context, claims *models.AccessClaims) {
	c.Request = c.Request.WithContext(utils.NewContextWithValues(
		c.Request.Context(),
		utils.ContextValues{
			UserID: claims.UserID,
			Role:   claims.Role,
		},
	))
}

// handleAuthError handles authentication errors with consistent response format
func (m *Middleware) handleAuthError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, response.BuildResponseFailed(message))
	c.Abort()
}
