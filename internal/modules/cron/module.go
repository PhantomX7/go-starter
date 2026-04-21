// Package cron wires the cron module.
package cron

import (
	"go.uber.org/fx"

	"github.com/PhantomX7/athleton/internal/modules/cron/controller"
	"github.com/PhantomX7/athleton/internal/modules/cron/service"
)

// Module wires the cron module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.NewCron,
		service.NewCronService,
	),
)
