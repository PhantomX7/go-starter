package post

import (
	"github.com/PhantomX7/go-starter/internal/modules/auth/controller"
	"github.com/PhantomX7/go-starter/internal/modules/auth/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewAuthController,
		service.NewAuthService,
	),
)
