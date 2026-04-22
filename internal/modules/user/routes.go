// Package user wires the user module.
package user

import (
	"github.com/PhantomX7/athleton/internal/modules/user/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type routeRegistrar struct {
	controller controller.UserController
}

// NewRoutes constructs the user route registrar.
func NewRoutes(controller controller.UserController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the user-management endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	userRoute := ctx.Admin.Group("/user")
	userRoute.GET("", ctx.MW.RequirePermission(permissions.UserRead), r.controller.Index)
	userRoute.GET("/:id", ctx.MW.RequirePermission(permissions.UserRead), r.controller.FindByID)
	userRoute.PATCH("/:id", ctx.MW.RequirePermission(permissions.UserUpdate), r.controller.Update)
	userRoute.POST("/:id/admin-role", ctx.MW.RequirePermission(permissions.UserAssignRole), r.controller.AssignAdminRole)
	userRoute.POST("/:id/change-password", ctx.MW.RequirePermission(permissions.AdminUserChangePassword), r.controller.ChangePassword)
}
