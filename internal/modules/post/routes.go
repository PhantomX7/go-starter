// Package post wires the post module.
package post

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/post/controller"
	"github.com/PhantomX7/athleton/internal/routes"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.PostController
}

// NewRoutes constructs the post route registrar.
func NewRoutes(controller controller.PostController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the post endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	postRoute := adminAPI.Group("/post")
	postRoute.GET("", r.controller.Index)
	postRoute.GET("/:id", r.controller.FindByID)
	postRoute.POST("", r.controller.Create)
	postRoute.PATCH("/:id", r.controller.Update)
	postRoute.DELETE("/:id", r.controller.Delete)
}
