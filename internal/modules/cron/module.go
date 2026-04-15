package cron

import (
	"github.com/PhantomX7/athleton/internal/modules/cron/controller"
	"github.com/PhantomX7/athleton/internal/modules/cron/service"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewCron,
		service.NewCronService,
	),
)
