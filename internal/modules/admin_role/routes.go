// Package admin_role wires the admin-role module into the application container.
package admin_role

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.AdminRoleController
}

// NewRoutes constructs the admin-role route registrar.
func NewRoutes(controller controller.AdminRoleController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the admin-role endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	adminRoleRoute := adminAPI.Group("/admin-role")
	adminRoleRoute.GET("", middleware.RequirePermission(permissions.AdminRoleRead), r.controller.Index)
	adminRoleRoute.GET("/permissions", middleware.RequirePermission(permissions.AdminRoleRead), r.controller.GetAllPermissions)
	adminRoleRoute.GET("/:id", middleware.RequirePermission(permissions.AdminRoleRead), r.controller.FindByID)
	adminRoleRoute.POST("", middleware.RequirePermission(permissions.AdminRoleCreate), r.controller.Create)
	adminRoleRoute.PATCH("/:id", middleware.RequirePermission(permissions.AdminRoleUpdate), r.controller.Update)
	adminRoleRoute.DELETE("/:id", middleware.RequirePermission(permissions.AdminRoleDelete), r.controller.Delete)
}
