// Package user wires the user module.
package user

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/user/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.UserController
}

// NewRoutes constructs the user route registrar.
func NewRoutes(controller controller.UserController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the user-management endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	userRoute := adminAPI.Group("/user")
	userRoute.GET("", middleware.RequirePermission(permissions.UserRead), r.controller.Index)
	userRoute.GET("/:id", middleware.RequirePermission(permissions.UserRead), r.controller.FindByID)
	userRoute.PATCH("/:id", middleware.RequirePermission(permissions.UserUpdate), r.controller.Update)
	userRoute.POST("/:id/admin-role", middleware.RequirePermission(permissions.UserAssignRole), r.controller.AssignAdminRole)
	userRoute.POST("/:id/change-password", middleware.RequirePermission(permissions.AdminUserChangePassword), r.controller.ChangePassword)
}
