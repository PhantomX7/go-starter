// Package log wires the log module.
package log

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/log/controller"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.LogController
}

// NewRoutes constructs the log route registrar.
func NewRoutes(controller controller.LogController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the audit-log endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	logRoute := adminAPI.Group("/log")
	logRoute.GET("", middleware.RequirePermission(permissions.LogRead), r.controller.Index)
}
