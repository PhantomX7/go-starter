package main

import (
	"context"
	"log"
	"net/http"

	_ "ariga.io/atlas-provider-gorm/gormschema"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/PhantomX7/go-starter/pkg/config"
)

func main() {
	app := fx.New(
		fx.NopLogger, // disable logger for fx
		fx.Provide(
			setUpConfig,
			// setupDatabase,
			// customValidator.New, // initiate custom validator
			// middleware.New,      // initiate middleware
			setupServer,
		),
		// modules.RepositoryModule,
		// modules.ServiceModule,
		// modules.ControllerModule,
		// libs.Module,
		// routes.Module,
		fx.Invoke(
			// startCron,
			startServer,
		),
	)
	app.Run()
}

func startServer(lc fx.Lifecycle, server *gin.Engine, cfg *config.Config) {

	// Initialize HTTP server
	srv := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      server,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Printf("Starting %s v%s on %s (environment: %s)",
					cfg.App.Name,
					cfg.App.Version,
					cfg.GetServerAddress(),
					cfg.App.Environment,
				)

				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("Failed to start server: %v", err)
				}

			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}

func setUpConfig() *config.Config {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	}

	// Return the loaded configuration
	return cfg
}

// setupServer configures and returns the Gin engine.
// It sets up middleware including CORS and logging.
func setupServer(cfg *config.Config) *gin.Engine {

	server := gin.Default()

	// Enable CORS middleware and custom logger
	// the order is important
	server.Use(
	// m.CORS(),         // Your custom CORS middleware
	// m.Logger(),       // Your custom detailed logger middleware
	// m.ErrorHandler(), // Your custom error handler middleware
	)

	// register static files
	server.Static("/assets", "./assets")
	// Register Prometheus metrics endpoint
	// server.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return server
}
