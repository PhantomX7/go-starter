// internal/middlewares/middleware.go
package middlewares

import (
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/libs/casbin"
)

type Middleware struct {
	authJWT      *authjwt.AuthJWT
	casbinClient casbin.Client
}

func NewMiddleware(
	authJWT *authjwt.AuthJWT,
	casbinClient casbin.Client,
) *Middleware {
	return &Middleware{
		authJWT:      authJWT,
		casbinClient: casbinClient,
	}
}
