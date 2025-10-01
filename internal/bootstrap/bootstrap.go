package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/PhantomX7/go-starter/pkg/config"
	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupServer configures and returns the Gin engine.
// It sets up middleware including CORS and logging.
func SetupServer(cfg *config.Config) *gin.Engine {

	server := gin.Default()

	// Enable CORS middleware and custom logger
	// the order is important
	server.Use(
	// m.CORS(),         // Your custom CORS middleware
	// m.Logger(),       // Your custom detailed logger middleware
	// m.ErrorHandler(), // Your custom error handler middleware
	)

	// register static files
	server.Static("/assets", cfg.App.Assets)
	// Register Prometheus metrics endpoint
	// server.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return server
}

// StartServer starts the HTTP server using the provided Gin engine and configuration.
// It sets up lifecycle hooks to start and stop the server.
func StartServer(lc fx.Lifecycle, server *gin.Engine, cfg *config.Config) {

	// Print application information
	myFigure := figure.NewColorFigure(cfg.App.Name, "", "green", true)
	myFigure.Print()

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
			log.Println("Shutting down server...")
			return srv.Shutdown(ctx)
		},
	})
}

// SetUpConfig loads the application configuration and sets the Gin mode based on the environment.
// It returns the loaded configuration.
func SetUpConfig() (*config.Config, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	}

	// Return the loaded configuration
	return cfg, nil
}

func SetUpDatabase(lc fx.Lifecycle, cfg *config.Config) (*gorm.DB, error) {
	// Set up database connection
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseURL()), &gorm.Config{})
	// db, err := gorm.Open(cfg.Database.Dialect, cfg.Database.ConnectionString)
	if err != nil {
		return nil, err
	}

	var sqlDB *sql.DB
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Connecting to the database...")
			sqlDB, err = db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Ping()
		},
		OnStop: func(ctx context.Context) error {
			log.Println("Closing database connection...")
			if sqlDB != nil {
				return sqlDB.Close()
			}
			return nil
		},
	})

	return db, nil
}
