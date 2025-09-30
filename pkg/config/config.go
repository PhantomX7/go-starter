// Package config provides configuration management using Viper library.
// It supports reading from .env files, environment variables, and provides
// automatic type conversion and configuration reloading capabilities.
package config

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:",squash"`

	// Database configuration
	Database DatabaseConfig `mapstructure:",squash"`

	// JWT configuration
	JWT JWTConfig `mapstructure:",squash"`

	// Application configuration
	App AppConfig `mapstructure:",squash"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string        `mapstructure:"SERVER_HOST"`
	Port         int           `mapstructure:"SERVER_PORT"`
	ReadTimeout  time.Duration `mapstructure:"SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `mapstructure:"SERVER_IDLE_TIMEOUT"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Driver   string `mapstructure:"DB_DRIVER"`
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Username string `mapstructure:"DB_USERNAME"`
	Password string `mapstructure:"DB_PASSWORD"`
	Database string `mapstructure:"DB_DATABASE"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret     string        `mapstructure:"JWT_SECRET"`
	Expiration time.Duration `mapstructure:"JWT_EXPIRATION"`
	Issuer     string        `mapstructure:"JWT_ISSUER"`
}

// AppConfig holds general application configuration
type AppConfig struct {
	Name        string `mapstructure:"APP_NAME"`
	Version     string `mapstructure:"APP_VERSION"`
	Environment string `mapstructure:"APP_ENVIRONMENT"`
	Debug       bool   `mapstructure:"APP_DEBUG"`
	LogLevel    string `mapstructure:"APP_LOG_LEVEL"`
	Assets      string `mapstructure:"APP_ASSETS"`
}

var (
	// cfg holds the global configuration instance
	cfg *Config
)

// Load initializes and loads the configuration from various sources.
// It reads from .env file, environment variables, and sets up automatic reloading.
func Load() (*Config, error) {
	v := viper.New()

	// Set configuration file name and paths
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	// Enable automatic environment variable binding
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults(v)

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Warning: .env file not found, using environment variables and defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	// Unmarshal configuration into struct
	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("SERVER_HOST", "localhost")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_READ_TIMEOUT", "30s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "30s")
	v.SetDefault("SERVER_IDLE_TIMEOUT", "120s")

	// Database defaults
	v.SetDefault("DB_DRIVER", "postgres")
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USERNAME", "postgres")
	v.SetDefault("DB_PASSWORD", "")
	v.SetDefault("DB_DATABASE", "starter")
	v.SetDefault("DB_SSLMODE", "disable")

	// JWT defaults
	v.SetDefault("JWT_SECRET", "your-secret-key")
	v.SetDefault("JWT_EXPIRATION", "24h")
	v.SetDefault("JWT_ISSUER", "starter")

	// App defaults
	v.SetDefault("APP_NAME", "Starter")
	v.SetDefault("APP_VERSION", "1.0.0")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_DEBUG", true)
	v.SetDefault("APP_LOG_LEVEL", "info")
	v.SetDefault("APP_ASSETS", "./assets")
}

// validateConfig validates the loaded configuration
func validateConfig(cfg *Config) error {
	// Validate server configuration
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// Validate database configuration
	if cfg.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	// Validate JWT configuration
	if cfg.JWT.Secret == "" || cfg.JWT.Secret == "your-secret-key" {
		return fmt.Errorf("JWT secret must be set and not use default value")
	}

	// Validate app configuration
	if cfg.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	validEnvironments := []string{"development", "staging", "production"}
	validEnv := slices.Contains(validEnvironments, cfg.App.Environment)
	if !validEnv {
		return fmt.Errorf("invalid environment: %s, must be one of %v",
			cfg.App.Environment,
			validEnvironments)
	}

	return nil
}

// Get returns the current configuration instance
func Get() *Config {
	if cfg == nil {
		log.Fatal("Configuration not loaded. Call config.Load() first.")
	}
	return cfg
}

// GetString returns a string configuration value by key
func GetString(key string) string {
	return viper.GetString(key)
}

// GetInt returns an integer configuration value by key
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool returns a boolean configuration value by key
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetDuration returns a duration configuration value by key
func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

// GetDatabaseURL constructs and returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	switch c.Database.Driver {
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Database,
			c.Database.SSLMode,
		)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Database,
		)
	default:
		return ""
	}
}

// GetServerAddress returns the server address in host:port format
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsProduction returns true if the application is running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if the application is running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}
