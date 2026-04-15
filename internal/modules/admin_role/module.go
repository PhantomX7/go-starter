package admin_role

import (
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.NewAdminRoleController,
		service.NewAdminRoleService,
		repository.NewAdminRoleRepository,
	),
)
