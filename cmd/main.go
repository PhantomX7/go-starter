// Package main starts the Athleton API application.
package main

import (
	"log"

	_ "github.com/PhantomX7/athleton/docs"
	"github.com/PhantomX7/athleton/internal/bootstrap"
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/libs"

	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/validator"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

//	@title			Athleton API
//	@version		1.0
//	@description	REST API for the Athleton platform.

//	@contact.name	Lezenda
//	@contact.url	https://github.com/PhantomX7
//	@contact.email	tester@lezenda.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@BasePath	/api/v1

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token.
func main() {
	// Load config first (logger needs it), then set Gin mode.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	bootstrap.ConfigureGinMode(cfg)

	// Set up logger immediately after config (all other setup functions need logger)
	if err := bootstrap.SetUpLogger(cfg); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	fx.New(appOptions(cfg, logger.Log)).Run()
}

// appOptions builds the full fx option set for the application. The loaded
// config and logger are passed in (not read from globals) and supplied to the
// container so every component declares them as explicit dependencies;
// logger.Ctx/helpers still back the ergonomic call sites. Extracted from main
// so the dependency graph can be validated in a test via fx.ValidateApp.
func appOptions(cfg *config.Config, log *zap.Logger) fx.Option {
	return fx.Options(
		fx.NopLogger, // disable default fx logger
		fx.Supply(cfg),
		fx.Supply(log),
		fx.Provide(
			bootstrap.SetUpDatabase,
			middlewares.NewMiddleware,
			validator.New,
			bootstrap.SetupServer,
		),
		libs.Module, // provide libs
		modules.Module,
		fx.Invoke(
			bootstrap.RegisterLoggerLifecycle, // Register logger lifecycle for graceful shutdown
			routes.RegisterRoutes,
			bootstrap.StartCron,
			bootstrap.StartServer,
		),
	)
}
