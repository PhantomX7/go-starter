// internal/middlewares/auth.go
package middlewares

import (
	"net/http"
	"slices"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequireAuth returns the standard authentication middleware
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return m.authJWT.Middleware.MiddlewareFunc()
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
