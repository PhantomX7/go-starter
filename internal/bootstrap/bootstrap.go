// Package bootstrap wires the application's shared runtime dependencies.
package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/PhantomX7/athleton/docs"
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
	cvalidator "github.com/PhantomX7/athleton/pkg/validator"

	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-playground/validator/v10"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// SetUpLogger initializes the logger before fx starts.
// This should be called in main() before fx.New().
func SetUpLogger() error {
	cfg := config.Get()
	logConfig := cfg.GetLoggerConfig()

	// Initialize logger
	if err := logger.Init(logger.Config{
		Level:       logConfig.Level,
		FilePath:    logConfig.FilePath,
		MaxSize:     logConfig.MaxSize,
		MaxBackups:  logConfig.MaxBackups,
		MaxAge:      logConfig.MaxAge,
		Compress:    logConfig.Compress,
		Console:     logConfig.Console,
		Environment: cfg.App.Environment,
	}); err != nil {
		return err
	}

	logger.Info("Logger initialized successfully",
		zap.String("level", logConfig.Level),
		zap.String("file_path", logConfig.FilePath),
		zap.String("environment", cfg.App.Environment),
	)

	return nil
}

// RegisterLoggerLifecycle registers logger lifecycle hooks with fx.
// This ensures proper cleanup on shutdown.
func RegisterLoggerLifecycle(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Syncing logger before shutdown")
			return logger.Sync()
		},
	})
}

// SetupServer configures and returns the Gin engine.
func SetupServer(m *middlewares.Middleware, cv cvalidator.CustomValidator, db *gorm.DB) *gin.Engine {
	cfg := config.Get()

	logger.Info("Setting up HTTP server")

	// gin.New() (not gin.Default()) so the only request logger in play is our
	// structured zap-based m.Logger(); Gin's stdout logger would just duplicate it.
	server := gin.New()

	// list of custom validators
	validators := map[string]validator.Func{
		"unique":   cv.Unique(),
		"exist":    cv.Exist(),
		"filesize": cv.FileSize(),
		"fileext":  cv.FileExtension(),
	}

	registerValidators(validators)

	// Apply middleware in order (ORDER IS IMPORTANT!)
	server.Use(
		m.RequestID(), // 1. Generate/extract request ID (MUST be before logger)
		m.CORS(),      // 2. CORS handling
		// m.TimeoutMiddleware(cfg.Server.ReadTimeout), // 3. Request timeout
		m.Logger(),       // 4. Request logging (outer, so it sees recovered panics)
		gin.Recovery(),   // 5. Panic recovery (inner, so Logger still logs the 500)
		m.ErrorHandler(), // 6. Error handling (MUST be last)
	)

	// Uniform JSON envelope for unmatched paths and methods so clients see the
	// same shape as handler errors.
	server.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.BuildResponseFailed("route not found"))
	})
	server.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, response.BuildResponseFailed("method not allowed"))
	})

	registerHealthRoutes(server, db)

	// Swagger is a discovery surface and shouldn't ship to production.
	if !cfg.IsProduction() {
		docs.SwaggerInfo.Title = "Komputer Medan API"
		docs.SwaggerInfo.Description = "This is a sample server Komputer Medan server."
		docs.SwaggerInfo.Version = "1.0"
		docs.SwaggerInfo.Host = cfg.GetServerAddress()
		docs.SwaggerInfo.BasePath = "/api/v1/"
		docs.SwaggerInfo.Schemes = []string{"http", "https"}

		server.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler, ginswagger.URL("/doc/doc.json")))
		server.GET("/doc/doc.json", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/json", []byte(docs.SwaggerInfo.ReadDoc()))
		})
	}

	// register static files
	server.Static("/assets", cfg.App.Assets)
	server.Static("/sitemaps", filepath.Join(cfg.App.Assets, "sitemaps")) // Server for sub-sitemaps
	server.StaticFile("/sitemap.xml", filepath.Join(cfg.App.Assets, "sitemap.xml"))

	logger.Info("HTTP server setup completed")

	return server
}

// registerHealthRoutes exposes liveness and readiness probes for load balancers
// and orchestrators. /livez and /healthz are cheap pings; /readyz also verifies
// the database is reachable, so it can fail even while the process is alive.
func registerHealthRoutes(server *gin.Engine, db *gorm.DB) {
	server.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	server.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	server.GET("/readyz", func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "database not initialized"})
			return
		}
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "database handle unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "reason": "database ping failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// StartServer starts the HTTP server using the provided Gin engine and configuration.
func StartServer(lc fx.Lifecycle, server *gin.Engine) {
	cfg := config.Get()

	// Print application information
	myFigure := figure.NewColorFigure(cfg.App.Name, "", "green", true)
	myFigure.Print()

	// Initialize HTTP server
	srv := &http.Server{
		Handler:      server,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Bind the listener synchronously so port-in-use failures surface
			// as an fx startup error, triggering orderly shutdown of already
			// started components instead of a goroutine Fatal.
			ln, err := net.Listen("tcp", cfg.GetServerAddress())
			if err != nil {
				logger.Error("Failed to bind server address", zap.String("address", cfg.GetServerAddress()), zap.Error(err))
				return err
			}

			logger.Info("Starting HTTP server",
				zap.String("name", cfg.App.Name),
				zap.String("version", cfg.App.Version),
				zap.String("address", cfg.GetServerAddress()),
				zap.String("environment", cfg.App.Environment),
			)

			go func() {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server exited with error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
}

// StartCron starts and stops the shared cron scheduler with the application lifecycle.
func StartCron(lc fx.Lifecycle, cron gocron.Scheduler) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting cron scheduler")
			cron.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping cron scheduler")
			return cron.Shutdown()
		},
	})
}

// SetUpConfig loads the application configuration and sets the Gin mode based on the environment.
func SetUpConfig() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	}

	return nil
}

// SetUpDatabase opens the shared GORM database connection and registers lifecycle hooks.
func SetUpDatabase(lc fx.Lifecycle) (*gorm.DB, error) {
	cfg := config.Get()

	logger.Info("Setting up database connection",
		zap.String("driver", cfg.Database.Driver),
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
	)

	// Configure GORM logger to use zap
	gormLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormlogger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  !cfg.IsProduction(),
			ParameterizedQueries:      true,
		},
	)

	// Set up database connection
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseURL()), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 gormLogger,
	})
	if err != nil {
		logger.Error("Failed to open database connection", zap.Error(err))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get database instance", zap.Error(err))
		return nil, err
	}

	// Configure connection pool
	ConfigureConnectionPool(sqlDB)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("Connecting to database")

				if err := sqlDB.PingContext(ctx); err != nil {
					logger.Error("Failed to ping database", zap.Error(err))
					return err
				}

				logger.Info("Database connection established successfully")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("Closing database connection")
				if sqlDB != nil {
					return sqlDB.Close()
				}
				return nil
			},
		})
	}

	return db, nil
}

func registerValidators(validators map[string]validator.Func) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		for name, fn := range validators {
			if err := v.RegisterValidation(name, fn); err != nil {
				logger.Error("Failed to register validator",
					zap.String("validator", name),
					zap.Error(err),
				)
			} else {
				logger.Debug("Registered custom validator", zap.String("validator", name))
			}
		}
	}
}

// ConfigureConnectionPool applies the default connection-pool settings for the application database.
func ConfigureConnectionPool(sqlDB *sql.DB) {
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	logger.Info("Database connection pool configured",
		zap.Int("max_idle_conns", 10),
		zap.Int("max_open_conns", 100),
		zap.Duration("conn_max_lifetime", time.Hour),
		zap.Duration("conn_max_idle_time", 10*time.Minute),
	)
}
