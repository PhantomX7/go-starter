// Package user wires the user module.
package user

import (
	"github.com/PhantomX7/athleton/internal/modules/user/controller"
	"github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module wires the user module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.NewUserController,
		service.NewUserService,
		repository.NewUserRepository,
		fx.Annotate(
			NewRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
