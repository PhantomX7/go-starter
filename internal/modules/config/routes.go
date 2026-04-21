// Package config wires the configuration module.
package config

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/config/controller"
	"github.com/PhantomX7/athleton/internal/routes"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.ConfigController
}

// NewRoutes constructs the config route registrar.
func NewRoutes(controller controller.ConfigController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the configuration endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	adminRoute := adminAPI.Group("/config")
	adminRoute.GET("", r.controller.Index)
	adminRoute.GET("/key/:key", r.controller.FindByKey)
	adminRoute.PATCH("/:id", middleware.RequireRole("root"), r.controller.Update)

	publicAPI := api.Group("/public")
	publicRoute := publicAPI.Group("/config")
	publicRoute.GET("", r.controller.Index)
	publicRoute.GET("/key/:key", r.controller.FindByKey)
}
