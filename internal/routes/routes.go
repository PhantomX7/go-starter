// Package routes registers the application's HTTP routes.
package routes

import (
	"reflect"
	"sort"

	"github.com/PhantomX7/athleton/internal/middlewares"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// Registrar lets a feature module mount its own routes.
type Registrar interface {
	RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware)
}

type registerParams struct {
	fx.In

	Registrars []Registrar `group:"routes"`
}

// RegisterRoutes mounts every API route on the provided Gin engine.
func RegisterRoutes(route *gin.Engine, middleware *middlewares.Middleware, params registerParams) {
	api := route.Group("/api/v1")
	sort.Slice(params.Registrars, func(i, j int) bool {
		return reflect.TypeOf(params.Registrars[i]).String() < reflect.TypeOf(params.Registrars[j]).String()
	})

	for _, registrar := range params.Registrars {
		registrar.RegisterRoutes(api, middleware)
	}
}
