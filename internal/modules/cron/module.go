package cron

import (
	"go.uber.org/fx"

	"github.com/PhantomX7/athleton/internal/modules/cron/controller"
	"github.com/PhantomX7/athleton/internal/modules/cron/service"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewCron,
		service.NewCronService,
	),
)
