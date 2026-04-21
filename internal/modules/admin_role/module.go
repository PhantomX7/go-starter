// Package admin_role wires the admin-role module into the application container.
package admin_role

import (
	"github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module registers the admin-role module dependencies.
var Module = fx.Options(
	fx.Provide(
		controller.NewAdminRoleController,
		service.NewAdminRoleService,
		repository.NewAdminRoleRepository,
		fx.Annotate(
			NewRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
