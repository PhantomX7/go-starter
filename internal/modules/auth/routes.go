// Package auth wires the authentication module into the application container.
package auth

import (
	"github.com/PhantomX7/athleton/internal/modules/auth/controller"
	"github.com/PhantomX7/athleton/internal/routes"
)

type routeRegistrar struct {
	controller controller.AuthController
}

// NewRoutes constructs the authentication route registrar.
func NewRoutes(controller controller.AuthController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the authentication endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	// Unauthenticated entry points. Rate-limit these to blunt credential-stuffing
	// and refresh-token abuse; see rate_limit.go for the per-IP limiter.
	rl := ctx.MW.AuthRateLimiter()
	publicAuth := ctx.Root.Group("/auth")
	publicAuth.POST("/register", rl, r.controller.Register)
	publicAuth.POST("/login", rl, ctx.MW.LoginHandler())
	publicAuth.POST("/refresh", rl, r.controller.Refresh)

	privateAuth := ctx.Root.Group("/auth", ctx.MW.RequireAuth())
	privateAuth.GET("/me", r.controller.GetMe)
	privateAuth.POST("/change-password", r.controller.ChangePassword)
	privateAuth.POST("/logout", r.controller.Logout)
}
