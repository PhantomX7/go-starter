// Package main starts the Athleton API application.
package main

import (
	"log"

	_ "github.com/PhantomX7/athleton/docs"
	"github.com/PhantomX7/athleton/internal/bootstrap"
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/libs"

	"github.com/PhantomX7/athleton/pkg/validator"

	"github.com/PhantomX7/athleton/internal/modules/admin_role"
	"github.com/PhantomX7/athleton/internal/modules/auth"
	"github.com/PhantomX7/athleton/internal/modules/config"
	"github.com/PhantomX7/athleton/internal/modules/cron"
	logs "github.com/PhantomX7/athleton/internal/modules/log"
	"github.com/PhantomX7/athleton/internal/modules/refresh_token"
	"github.com/PhantomX7/athleton/internal/modules/user"

	"go.uber.org/fx"
)

//	@contact.name	Lezenda
//	@contact.url	https://github.com/PhantomX7
//	@contact.email	tester@lezenda.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token.
func main() {
	// Set up config first (logger needs config)
	if err := bootstrap.SetUpConfig(); err != nil {
		log.Fatalf("Failed to set up config: %v", err)
	}

	// Set up logger immediately after config (all other setup functions need logger)
	if err := bootstrap.SetUpLogger(); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	app := fx.New(
		fx.NopLogger, // disable default fx logger
		fx.Provide(
			bootstrap.SetUpDatabase,
			middlewares.NewMiddleware,
			validator.New,
			bootstrap.SetupServer,
		),
		libs.Module, // provide libs
		admin_role.Module,
		auth.Module,
		config.Module,
		cron.Module,
		logs.Module,
		refresh_token.Module,
		user.Module,
		fx.Invoke(
			bootstrap.RegisterLoggerLifecycle, // Register logger lifecycle for graceful shutdown
			routes.RegisterRoutes,
			bootstrap.StartCron,
			bootstrap.StartServer,
		),
	)
	app.Run()
}
