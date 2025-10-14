package main

import (
	"github.com/PhantomX7/go-starter/internal/bootstrap"
	"github.com/PhantomX7/go-starter/internal/middlewares"
	"github.com/PhantomX7/go-starter/internal/routes"
	"github.com/PhantomX7/go-starter/libs"
	"github.com/PhantomX7/go-starter/pkg/validator"

	"github.com/PhantomX7/go-starter/internal/modules/auth"
	"github.com/PhantomX7/go-starter/internal/modules/post"
	"github.com/PhantomX7/go-starter/internal/modules/refresh_token"
	"github.com/PhantomX7/go-starter/internal/modules/user"

	"go.uber.org/fx"
)

//	@contact.name	PhantomX7
//	@contact.url	https://github.com/PhantomX7
//	@contact.email	zrphntm@gmail.com

// @license.name	Apache 2.0
// @license.url	http://www.apache.org/licenses/LICENSE-2.0.html
func main() {

	app := fx.New(
		// fx.NopLogger, // disable logger for fx
		fx.Provide(
			bootstrap.SetUpConfig,
			bootstrap.SetUpDatabase,
			middlewares.NewMiddleware,
			validator.New,
			bootstrap.SetupServer, 
		),
		libs.Module, // provide libs
		auth.Module,
		post.Module,
		refresh_token.Module,
		user.Module,
		fx.Invoke(
			routes.RegisterRoutes,
			bootstrap.StartServer,
		),
	)
	app.Run()
}
