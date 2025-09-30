package post

import (
	"github.com/PhantomX7/go-starter/internal/modules/post/controller"
	"github.com/PhantomX7/go-starter/internal/modules/post/repository"
	"github.com/PhantomX7/go-starter/internal/modules/post/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewPostController,
		service.NewPostService,
		repository.NewPostRepository,
	),
)
