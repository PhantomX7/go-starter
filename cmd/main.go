package main

import (
	"github.com/PhantomX7/go-starter/internal/bootstrap"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.NopLogger, // disable logger for fx
		fx.Provide(
			bootstrap.SetUpConfig,
			bootstrap.SetupServer,
		),
		fx.Invoke(
			bootstrap.StartServer,
		),
	)
	app.Run()
}
