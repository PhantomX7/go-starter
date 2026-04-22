// Package post wires the post module.
package post

import (
	"github.com/PhantomX7/athleton/internal/modules/post/controller"
	"github.com/PhantomX7/athleton/internal/routes"
)

type routeRegistrar struct {
	controller controller.PostController
}

// NewRoutes constructs the post route registrar.
func NewRoutes(controller controller.PostController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the post endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	postRoute := ctx.Admin.Group("/post")
	postRoute.GET("", r.controller.Index)
	postRoute.GET("/:id", r.controller.FindByID)
	postRoute.POST("", r.controller.Create)
	postRoute.PATCH("/:id", r.controller.Update)
	postRoute.DELETE("/:id", r.controller.Delete)
}
