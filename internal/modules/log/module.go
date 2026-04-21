// Package log wires the log module.
package log

import (
	"github.com/PhantomX7/athleton/internal/modules/log/controller"
	"github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/internal/modules/log/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module wires the log module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.NewLogController,
		service.NewLogService,
		repository.NewLogRepository,
		fx.Annotate(
			NewRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
