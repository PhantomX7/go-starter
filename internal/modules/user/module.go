package user

import (
	"github.com/PhantomX7/go-starter/internal/modules/user/controller"
	"github.com/PhantomX7/go-starter/internal/modules/user/repository"
	"github.com/PhantomX7/go-starter/internal/modules/user/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewUserController,
		service.NewUserService,
		repository.NewUserRepository,
	),
)
