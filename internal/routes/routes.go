// Package routes registers the application's HTTP routes.
package routes

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"

	adminrole "github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	auth "github.com/PhantomX7/athleton/internal/modules/auth/controller"
	config "github.com/PhantomX7/athleton/internal/modules/config/controller"
	log "github.com/PhantomX7/athleton/internal/modules/log/controller"
	user "github.com/PhantomX7/athleton/internal/modules/user/controller"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts every API route on the provided Gin engine.
func RegisterRoutes(
	route *gin.Engine,
	middleware *middlewares.Middleware,
	adminRoleController adminrole.AdminRoleController,
	authController auth.AuthController,
	configController config.ConfigController,
	logController log.LogController,
	userController user.UserController,
) {
	api := route.Group("/api/v1")
	{
		// ============================================================
		// AUTH ROUTES (Public + Authenticated)
		// ============================================================
		authRoute := api.Group("/auth")
		{
			authRoute.POST("/register", authController.Register)
			authRoute.POST("/login", middleware.LoginHandler())
			authRoute.POST("/refresh", authController.Refresh)

			authenticatedAuthRoute := authRoute.Group("", middleware.RequireAuth())
			{
				authenticatedAuthRoute.GET("/me", authController.GetMe)
				authenticatedAuthRoute.POST("/change-password", authController.ChangePassword)
				authenticatedAuthRoute.POST("/logout", authController.Logout)
			}
		}

		// ============================================================
		// ADMIN ROUTES (Requires Auth + Permission)
		// ============================================================
		adminAPI := api.Group("/admin", middleware.RequireAuth())
		{
			// ---------------------------------------------------------
			// Admin Role Management (Root only - no permission check)
			// ---------------------------------------------------------
			adminRoleRoute := adminAPI.Group("/admin-role")
			{
				adminRoleRoute.GET("", middleware.RequirePermission(permissions.AdminRoleRead), adminRoleController.Index)
				adminRoleRoute.GET("/permissions", middleware.RequirePermission(permissions.AdminRoleRead), adminRoleController.GetAllPermissions)
				adminRoleRoute.GET("/:id", middleware.RequirePermission(permissions.AdminRoleRead), adminRoleController.FindByID)
				adminRoleRoute.POST("", middleware.RequirePermission(permissions.AdminRoleCreate), adminRoleController.Create)
				adminRoleRoute.PATCH("/:id", middleware.RequirePermission(permissions.AdminRoleUpdate), adminRoleController.Update)
				adminRoleRoute.DELETE("/:id", middleware.RequirePermission(permissions.AdminRoleDelete), adminRoleController.Delete)
			}

			// ---------------------------------------------------------
			// Config Management
			// ---------------------------------------------------------
			configRoute := adminAPI.Group("/config")
			{
				configRoute.GET("", configController.Index)
				configRoute.GET("/key/:key", configController.FindByKey)
				configRoute.PATCH("/:id", middleware.RequireRole("root"), configController.Update)
			}

			// ---------------------------------------------------------
			// Log Management (Audit Logs)
			// ---------------------------------------------------------
			logRoute := adminAPI.Group("/log")
			{
				logRoute.GET("", middleware.RequirePermission(permissions.LogRead), logController.Index)
			}

			// ---------------------------------------------------------
			// User Management
			// ---------------------------------------------------------
			userRoute := adminAPI.Group("/user")
			{
				userRoute.GET("", middleware.RequirePermission(permissions.UserRead), userController.Index)
				userRoute.GET("/:id", middleware.RequirePermission(permissions.UserRead), userController.FindByID)
				userRoute.PATCH("/:id", middleware.RequirePermission(permissions.UserUpdate), userController.Update)
				userRoute.POST("/:id/admin-role", middleware.RequirePermission(permissions.UserAssignRole), userController.AssignAdminRole)
				userRoute.POST("/:id/change-password", middleware.RequirePermission(permissions.AdminUserChangePassword), userController.ChangePassword)
			}
		}

		// ============================================================
		// PUBLIC ROUTES
		// ============================================================
		publicAPI := api.Group("/public")
		{
			configRoute := publicAPI.Group("/config")
			{
				configRoute.GET("", configController.Index)
				configRoute.GET("/key/:key", configController.FindByKey)
			}
		}
	}
}
