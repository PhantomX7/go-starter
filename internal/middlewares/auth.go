// internal/middlewares/auth.go
package middlewares

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// RequireAuth returns the standard authentication middleware
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return m.authJWT.Middleware.MiddlewareFunc()
}

// OptionalAuth allows requests to proceed with or without valid authentication
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, role, adminRoleID, ok := parseTokenFromHeader(c); ok {
			// Note: userName is empty here as it's not stored in the JWT token
			// For audit logging, the user name will be fetched from the database
			setContextValues(c, userID, "", role, adminRoleID)
		}
		c.Next()
	}
}

// RequireRole validates if the authenticated user has one of the allowed roles
func (m *Middleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		values, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		if !slices.Contains(allowedRoles, values.Role) {
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission validates if the user has the required permission
// - Root users bypass all permission checks
// - Admin users are checked against their admin_role permissions via Casbin
// - Other roles are denied access
func (m *Middleware) RequirePermission(permission permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := utils.GetRequestIDFromContext(c.Request.Context())

		values, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			logger.Warn("Failed to get context values in RequirePermission",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		// Root bypasses all permission checks
		if values.Role == models.UserRoleRoot.ToString() {
			logger.Debug("Root user - permission check bypassed",
				zap.String("request_id", requestID),
				zap.Uint("user_id", values.UserID),
				zap.String("permission", permission.String()),
			)
			c.Next()
			return
		}

		// Only admin role can have granular permissions
		if values.Role != models.UserRoleAdmin.ToString() {
			logger.Warn("Access denied - not an admin user",
				zap.String("request_id", requestID),
				zap.Uint("user_id", values.UserID),
				zap.String("user_role", values.Role),
				zap.String("permission", permission.String()),
			)
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		// Admin must have an admin_role assigned
		if values.AdminRoleID == nil {
			logger.Warn("Access denied - admin user has no role assigned",
				zap.String("request_id", requestID),
				zap.Uint("user_id", values.UserID),
				zap.String("permission", permission.String()),
			)
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("no admin role assigned"))
			c.Abort()
			return
		}

		// Check permission via Casbin
		allowed, err := m.casbinClient.CheckPermission(*values.AdminRoleID, permission.String())
		if err != nil {
			logger.Error("Failed to check permission",
				zap.String("request_id", requestID),
				zap.Uint("user_id", values.UserID),
				zap.Uint("admin_role_id", *values.AdminRoleID),
				zap.String("permission", permission.String()),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("failed to verify permissions"))
			c.Abort()
			return
		}

		if !allowed {
			logger.Warn("Access denied - permission not granted",
				zap.String("request_id", requestID),
				zap.Uint("user_id", values.UserID),
				zap.Uint("admin_role_id", *values.AdminRoleID),
				zap.String("permission", permission.String()),
			)
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		logger.Debug("Permission check passed",
			zap.String("request_id", requestID),
			zap.Uint("user_id", values.UserID),
			zap.Uint("admin_role_id", *values.AdminRoleID),
			zap.String("permission", permission.String()),
		)

		c.Next()
	}
}

// RequireAnyPermission validates if the user has at least one of the required permissions
func (m *Middleware) RequireAnyPermission(perms ...permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := utils.GetRequestIDFromContext(c.Request.Context())

		values, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			logger.Warn("Failed to get context values in RequireAnyPermission",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		// Root bypasses all permission checks
		if values.Role == models.UserRoleRoot.ToString() {
			c.Next()
			return
		}

		// Only admin role can have granular permissions
		if values.Role != models.UserRoleAdmin.ToString() || values.AdminRoleID == nil {
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		for _, perm := range perms {
			allowed, err := m.casbinClient.CheckPermission(*values.AdminRoleID, perm.String())
			if err != nil {
				logger.Error("Failed to check permission",
					zap.String("request_id", requestID),
					zap.Uint("user_id", values.UserID),
					zap.String("permission", perm.String()),
					zap.Error(err),
				)
				continue
			}
			if allowed {
				c.Next()
				return
			}
		}

		permStrings := make([]string, len(perms))
		for i, p := range perms {
			permStrings[i] = p.String()
		}

		logger.Warn("Access denied - none of the required permissions granted",
			zap.String("request_id", requestID),
			zap.Uint("user_id", values.UserID),
			zap.Uint("admin_role_id", *values.AdminRoleID),
			zap.Strings("required_permissions", permStrings),
		)

		c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
		c.Abort()
	}
}

// RequireAllPermissions validates if the user has all of the required permissions
func (m *Middleware) RequireAllPermissions(perms ...permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := utils.GetRequestIDFromContext(c.Request.Context())

		values, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			logger.Warn("Failed to get context values in RequireAllPermissions",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		// Root bypasses all permission checks
		if values.Role == models.UserRoleRoot.ToString() {
			c.Next()
			return
		}

		// Only admin role can have granular permissions
		if values.Role != models.UserRoleAdmin.ToString() || values.AdminRoleID == nil {
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		// Check if user has all required permissions
		for _, perm := range perms {
			allowed, err := m.casbinClient.CheckPermission(*values.AdminRoleID, perm.String())
			if err != nil {
				logger.Error("Failed to check permission",
					zap.String("request_id", requestID),
					zap.Uint("user_id", values.UserID),
					zap.String("permission", perm.String()),
					zap.Error(err),
				)
				c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("failed to verify permissions"))
				c.Abort()
				return
			}
			if !allowed {
				logger.Warn("Access denied - missing required permission",
					zap.String("request_id", requestID),
					zap.Uint("user_id", values.UserID),
					zap.Uint("admin_role_id", *values.AdminRoleID),
					zap.String("missing_permission", perm.String()),
				)
				c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// LoginHandler returns gin-jwt's login handler
func (m *Middleware) LoginHandler() gin.HandlerFunc {
	return m.authJWT.Middleware.LoginHandler
}

// parseTokenFromHeader extracts and validates token, returns userID, role, adminRoleID, and success
func parseTokenFromHeader(c *gin.Context) (uint, string, *uint, bool) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, "", nil, false
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Get().JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return 0, "", nil, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", nil, false
	}

	userIDFloat, ok := claims[authjwt.IdentityKey].(float64)
	if !ok {
		return 0, "", nil, false
	}

	role, _ := claims[authjwt.RoleKey].(string)

	// Extract admin_role_id (optional)
	var adminRoleID *uint
	if val, ok := claims[authjwt.AdminRoleIDKey]; ok && val != nil {
		id := uint(val.(float64))
		adminRoleID = &id
	}

	return uint(userIDFloat), role, adminRoleID, true
}

// setContextValues sets user context values
func setContextValues(c *gin.Context, userID uint, userName string, role string, adminRoleID *uint) {
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
