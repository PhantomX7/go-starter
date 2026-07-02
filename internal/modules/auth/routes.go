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
	// and refresh-token abuse; see rate_limit.go for the per-IP limiters. Each
	// endpoint gets its OWN limiter instance so traffic on one cannot exhaust
	// another's budget (e.g. a burst of refreshes must not lock out login).
	// Refresh uses a deliberately looser limiter: legitimate clients refresh
	// routinely and present an unguessable token, not guessable credentials.
	publicAuth := ctx.Root.Group("/auth")
	publicAuth.POST("/register", ctx.MW.AuthRateLimiter(), r.controller.Register)
	publicAuth.POST("/login", ctx.MW.AuthRateLimiter(), ctx.MW.LoginHandler())
	publicAuth.POST("/refresh", ctx.MW.RefreshRateLimiter(), r.controller.Refresh)

	privateAuth := ctx.Root.Group("/auth", ctx.MW.RequireAuth())
	privateAuth.GET("/me", r.controller.GetMe)
	privateAuth.POST("/change-password", r.controller.ChangePassword)
	privateAuth.POST("/logout", r.controller.Logout)
}
