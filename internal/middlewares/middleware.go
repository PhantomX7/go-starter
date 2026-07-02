// Package middlewares provides shared Gin middleware for the API.
package middlewares

import (
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/pkg/config"
)

// Middleware aggregates the reusable Gin middleware dependencies.
type Middleware struct {
	cfg          *config.Config
	authJWT      *authjwt.AuthJWT
	casbinClient casbin.Client
}

// NewMiddleware constructs the application's middleware bundle.
func NewMiddleware(
	cfg *config.Config,
	authJWT *authjwt.AuthJWT,
	casbinClient casbin.Client,
) *Middleware {
	return &Middleware{
		cfg:          cfg,
		authJWT:      authJWT,
		casbinClient: casbinClient,
	}
}
