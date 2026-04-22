// Package routes registers the application's HTTP routes.
package routes

import (
	"github.com/PhantomX7/athleton/internal/middlewares"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// Context bundles the root groups and middleware available to module registrars.
// Shared groups are built once so every module applies the same middleware stack
// (e.g. a single RequireAuth handler chain for everything under /admin).
type Context struct {
	// Root is /api/v1 and carries no extra middleware.
	Root *gin.RouterGroup
	// Public is /api/v1/public for unauthenticated read endpoints.
	Public *gin.RouterGroup
	// Admin is /api/v1/admin with RequireAuth already attached.
	Admin *gin.RouterGroup
	// MW exposes the middleware bundle for per-route guards (permissions, roles).
	MW *middlewares.Middleware
}

// Registrar lets a feature module mount its own routes.
type Registrar interface {
	RegisterRoutes(ctx *Context)
}

type registerParams struct {
	fx.In

	Registrars []Registrar `group:"routes"`
}

// RegisterRoutes mounts every API route on the provided Gin engine.
func RegisterRoutes(engine *gin.Engine, middleware *middlewares.Middleware, params registerParams) {
	root := engine.Group("/api/v1")
	ctx := &Context{
		Root:   root,
		Public: root.Group("/public"),
		MW:     middleware,
	}
	if middleware != nil {
		ctx.Admin = root.Group("/admin", middleware.RequireAuth())
	}

	for _, registrar := range params.Registrars {
		registrar.RegisterRoutes(ctx)
	}
}
