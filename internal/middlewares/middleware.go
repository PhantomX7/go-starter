// Package middlewares provides shared Gin middleware for the API.
package middlewares

import (
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/libs/casbin"
)

// Middleware aggregates the reusable Gin middleware dependencies.
type Middleware struct {
	authJWT      *authjwt.AuthJWT
	casbinClient casbin.Client
}

// NewMiddleware constructs the application's middleware bundle.
func NewMiddleware(
	authJWT *authjwt.AuthJWT,
	casbinClient casbin.Client,
) *Middleware {
	return &Middleware{
		authJWT:      authJWT,
		casbinClient: casbinClient,
	}
}
