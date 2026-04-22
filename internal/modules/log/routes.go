// Package log wires the log module.
package log

import (
	"github.com/PhantomX7/athleton/internal/modules/log/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type routeRegistrar struct {
	controller controller.LogController
}

// NewRoutes constructs the log route registrar.
func NewRoutes(controller controller.LogController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the audit-log endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	logRoute := ctx.Admin.Group("/log")
	logRoute.GET("", ctx.MW.RequirePermission(permissions.LogRead), r.controller.Index)
}
