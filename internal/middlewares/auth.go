// Package middlewares provides shared Gin middleware for the API.
package middlewares

import (
	"net/http"
	"slices"

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

// RequirePermission validates that the authenticated user holds permission.
// The authorization rule itself — root bypasses, non-admin is denied, admin is
// checked against its admin_role via Casbin — lives in
// casbin.CheckPermissionWithRoot so every guard (here and the Any/All variants)
// shares one decision instead of re-deriving it. Denials return a generic 403
// rather than naming the reason, to avoid leaking authorization internals.
func (m *Middleware) RequirePermission(permission permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		values, err := utils.ValuesFromContext(ctx)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		allowed, err := m.casbinClient.CheckPermissionWithRoot(values.Role, values.AdminRoleID, permission.String())
		if err != nil {
			logger.Ctx(ctx).Error("Failed to verify permission",
				zap.String("permission", permission.String()), zap.Error(err))
			c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("failed to verify permissions"))
			c.Abort()
			return
		}
		if !allowed {
			logger.Ctx(ctx).Warn("Access denied",
				zap.String("permission", permission.String()), zap.String("role", values.Role))
			c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission allows the request when the user holds at least one of perms.
func (m *Middleware) RequireAnyPermission(perms ...permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		values, err := utils.ValuesFromContext(ctx)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		for _, perm := range perms {
			allowed, err := m.casbinClient.CheckPermissionWithRoot(values.Role, values.AdminRoleID, perm.String())
			if err != nil {
				logger.Ctx(ctx).Error("Failed to verify permission",
					zap.String("permission", perm.String()), zap.Error(err))
				continue
			}
			if allowed {
				c.Next()
				return
			}
		}

		logger.Ctx(ctx).Warn("Access denied", zap.String("role", values.Role))
		c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
		c.Abort()
	}
}

// RequireAllPermissions allows the request only when the user holds every perm.
func (m *Middleware) RequireAllPermissions(perms ...permissions.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		values, err := utils.ValuesFromContext(ctx)
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		for _, perm := range perms {
			allowed, err := m.casbinClient.CheckPermissionWithRoot(values.Role, values.AdminRoleID, perm.String())
			if err != nil {
				logger.Ctx(ctx).Error("Failed to verify permission",
					zap.String("permission", perm.String()), zap.Error(err))
				c.JSON(http.StatusInternalServerError, response.BuildResponseFailed("failed to verify permissions"))
				c.Abort()
				return
			}
			if !allowed {
				logger.Ctx(ctx).Warn("Access denied", zap.String("missing_permission", perm.String()))
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
