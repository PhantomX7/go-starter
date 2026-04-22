// Package config wires the configuration module.
package config

import (
	"github.com/PhantomX7/athleton/internal/modules/config/controller"
	"github.com/PhantomX7/athleton/internal/modules/config/repository"
	"github.com/PhantomX7/athleton/internal/modules/config/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module wires the config module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.NewConfigController,
		service.NewConfigService,
		repository.NewConfigRepository,
		fx.Annotate(
			NewAdminRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
		fx.Annotate(
			NewPublicRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
