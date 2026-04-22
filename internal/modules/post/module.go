// Package post wires the post module.
package post

import (
	"github.com/PhantomX7/athleton/internal/modules/post/controller"
	"github.com/PhantomX7/athleton/internal/modules/post/repository"
	"github.com/PhantomX7/athleton/internal/modules/post/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module wires the post module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.NewPostController,
		service.NewPostService,
		repository.NewPostRepository,
		fx.Annotate(
			NewRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
