package main

import (
	"github.com/PhantomX7/go-starter/internal/bootstrap"
	"github.com/PhantomX7/go-starter/internal/routes"

	postModule "github.com/PhantomX7/go-starter/internal/modules/post"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		// fx.NopLogger, // disable logger for fx
		fx.Provide(
			bootstrap.SetUpConfig,
			bootstrap.SetUpDatabase,
			bootstrap.SetupServer,
		),
		postModule.Module,
		fx.Invoke(
			routes.RegisterRoutes,
			bootstrap.StartServer,
		),
	)
	app.Run()
}
