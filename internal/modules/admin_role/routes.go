// Package admin_role wires the admin-role module into the application container.
package admin_role

import (
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type routeRegistrar struct {
	controller controller.AdminRoleController
}

// NewRoutes constructs the admin-role route registrar.
func NewRoutes(controller controller.AdminRoleController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the admin-role endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	adminRoleRoute := ctx.Admin.Group("/admin-role")
	adminRoleRoute.GET("", ctx.MW.RequirePermission(permissions.AdminRoleRead), r.controller.Index)
	adminRoleRoute.GET("/permissions", ctx.MW.RequirePermission(permissions.AdminRoleRead), r.controller.GetAllPermissions)
	adminRoleRoute.GET("/:id", ctx.MW.RequirePermission(permissions.AdminRoleRead), r.controller.FindByID)
	adminRoleRoute.POST("", ctx.MW.RequirePermission(permissions.AdminRoleCreate), r.controller.Create)
	adminRoleRoute.PATCH("/:id", ctx.MW.RequirePermission(permissions.AdminRoleUpdate), r.controller.Update)
	adminRoleRoute.DELETE("/:id", ctx.MW.RequirePermission(permissions.AdminRoleDelete), r.controller.Delete)
}
