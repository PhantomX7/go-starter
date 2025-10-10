package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/PhantomX7/go-starter/docs"
	"github.com/PhantomX7/go-starter/internal/middlewares"
	"github.com/PhantomX7/go-starter/pkg/config"
	custom_validator "github.com/PhantomX7/go-starter/pkg/validator"

	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupServer configures and returns the Gin engine.
// It sets up middleware including CORS and logging.
func SetupServer(cfg *config.Config, m *middlewares.Middleware, cv custom_validator.CustomValidator) *gin.Engine {

	// programmatically set swagger info
	docs.SwaggerInfo.Title = "Go Starter API"
	docs.SwaggerInfo.Description = "This is a sample server Go Starter server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = cfg.GetServerAddress()
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	server := gin.Default()

	// list of custom validators
	validators := map[string]validator.Func{
		"unique": cv.Unique(),
		"exist":  cv.Exist(),
	}

	registerValidators(validators)

	// Enable CORS middleware and custom logger
	// the order is important
	server.Use(
		// m.CORS(),         // Your custom CORS middleware
		// m.Logger(),       // Your custom detailed logger middleware
		m.ErrorHandler(), // Your custom error handler middleware
	)
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseURL()), &gorm.Config{
		SkipDefaultTransaction: true,
	})
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

			// Configure connection pool
			ConfigureConnectionPool(sqlDB)

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

	return db.Debug(), nil
}

func registerValidators(validators map[string]validator.Func) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Register each validator
		for name, fn := range validators {
			if err := v.RegisterValidation(name, fn); err != nil {
				log.Printf("error when applying %s validator: %v", name, err)
			}
		}
	}
}

func ConfigureConnectionPool(sqlDB *sql.DB) {
	// SetMaxIdleConns sets the maximum number of connections in the idle pool
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum time a connection can be reused
	sqlDB.SetConnMaxLifetime(time.Hour)

	// SetConnMaxIdleTime sets the maximum time a connection can be idle
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
}
