package main

import (
	"log"
	"net/http"

	_ "ariga.io/atlas-provider-gorm/gormschema"
	"github.com/gin-gonic/gin"

	"github.com/PhantomX7/go-starter/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else if !cfg.App.Debug {
		gin.SetMode(gin.TestMode)
	}

	// Create Gin router
	r := gin.Default()

	// Health check endpoint
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":     "pong",
			"app_name":    cfg.App.Name,
			"version":     cfg.App.Version,
			"environment": cfg.App.Environment,
		})
	})

	// Configuration info endpoint (for development only)
	if cfg.IsDevelopment() {
		r.GET("/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"server": gin.H{
					"host": cfg.Server.Host,
					"port": cfg.Server.Port,
				},
				"app": gin.H{
					"name":        cfg.App.Name,
					"version":     cfg.App.Version,
					"environment": cfg.App.Environment,
					"debug":       cfg.App.Debug,
				},
				"database": gin.H{
					"driver": cfg.Database.Driver,
					"host":   cfg.Database.Host,
					"port":   cfg.Database.Port,
				},
			})
		})
	}

	// Create HTTP server with configuration
	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("Starting %s v%s on %s (environment: %s)",
		cfg.App.Name,
		cfg.App.Version,
		cfg.GetServerAddress(),
		cfg.App.Environment,
	)

	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
