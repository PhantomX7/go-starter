package main

import (
	"github.com/PhantomX7/go-starter/internal/bootstrap"
	"github.com/PhantomX7/go-starter/internal/middlewares"
	"github.com/PhantomX7/go-starter/internal/routes"
	"github.com/PhantomX7/go-starter/libs"

	postModule "github.com/PhantomX7/go-starter/internal/modules/post"

	"go.uber.org/fx"
)

//	@contact.name	PhantomX7
//	@contact.url	https://github.com/PhantomX7
//	@contact.email	zrphntm@gmail.com

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html
func main() {

	app := fx.New(
		fx.NopLogger, // disable logger for fx
		fx.Provide(
			bootstrap.SetUpConfig,
			bootstrap.SetUpDatabase,
			middlewares.NewMiddleware,
			bootstrap.SetupServer, 
			libs.Module, // provide libs
		),
		postModule.Module,
		fx.Invoke(
			routes.RegisterRoutes,
			bootstrap.StartServer,
		),
	)
	app.Run()
}
