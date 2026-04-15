package config

import (
	"github.com/PhantomX7/athleton/internal/modules/config/controller"
	"github.com/PhantomX7/athleton/internal/modules/config/repository"
	"github.com/PhantomX7/athleton/internal/modules/config/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewConfigController,
		service.NewConfigService,
		repository.NewConfigRepository,
	),
)
