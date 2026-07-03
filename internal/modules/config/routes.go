// Package config wires the configuration module.
package config

import (
	"github.com/PhantomX7/athleton/internal/modules/config/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type adminRoutes struct {
	controller controller.ConfigController
}

// NewAdminRoutes constructs the admin-scoped config route registrar.
func NewAdminRoutes(controller controller.ConfigController) routes.Registrar {
	return &adminRoutes{controller: controller}
}

// RegisterRoutes mounts the admin configuration endpoints. Reads and updates
// are permission-guarded like every other admin module; root bypasses the
// checks, and admins need an explicit config:* grant.
func (r *adminRoutes) RegisterRoutes(ctx *routes.Context) {
	cfg := ctx.Admin.Group("/config")
	cfg.GET("", ctx.MW.RequirePermission(permissions.ConfigRead), r.controller.Index)
	cfg.GET("/key/:key", ctx.MW.RequirePermission(permissions.ConfigRead), r.controller.FindByKey)
	cfg.PATCH("/:id", ctx.MW.RequirePermission(permissions.ConfigUpdate), r.controller.Update)
}

type publicRoutes struct {
	controller controller.ConfigController
}

// NewPublicRoutes constructs the public-scoped config route registrar.
func NewPublicRoutes(controller controller.ConfigController) routes.Registrar {
	return &publicRoutes{controller: controller}
}

// RegisterRoutes mounts the public read-only configuration endpoints.
func (r *publicRoutes) RegisterRoutes(ctx *routes.Context) {
	cfg := ctx.Public.Group("/config")
	cfg.GET("", r.controller.Index)
	cfg.GET("/key/:key", r.controller.FindByKey)
}
