// Package auth wires the authentication module into the application container.
package auth

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/auth/controller"
	"github.com/PhantomX7/athleton/internal/routes"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.AuthController
}

// NewRoutes constructs the authentication route registrar.
func NewRoutes(controller controller.AuthController) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the authentication endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	authRoute := api.Group("/auth")
	authRoute.POST("/register", r.controller.Register)
	authRoute.POST("/login", middleware.LoginHandler())
	authRoute.POST("/refresh", r.controller.Refresh)

	authenticatedAuthRoute := authRoute.Group("", middleware.RequireAuth())
	authenticatedAuthRoute.GET("/me", r.controller.GetMe)
	authenticatedAuthRoute.POST("/change-password", r.controller.ChangePassword)
	authenticatedAuthRoute.POST("/logout", r.controller.Logout)
}
