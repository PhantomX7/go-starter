package auth

import (
	"github.com/PhantomX7/athleton/internal/modules/auth/controller"
	jwtauth "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/internal/modules/auth/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewAuthController,
		service.NewAuthService,
		jwtauth.NewAuthJWT,
	),
)
