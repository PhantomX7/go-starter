package log

import (
	"github.com/PhantomX7/athleton/internal/modules/log/controller"
	"github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/internal/modules/log/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewLogController,
		service.NewLogService,
		repository.NewLogRepository,
	),
)
