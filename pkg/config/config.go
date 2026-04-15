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
	Server   ServerConfig   `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`
	App      AppConfig      `mapstructure:",squash"`
	S3       S3Config       `mapstructure:",squash"`
	Bleve    BleveConfig    `mapstructure:",squash"`
	Admin    AdminConfig    `mapstructure:",squash"`
	Log      LogConfig      `mapstructure:",squash"`
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
	Driver   string `mapstructure:"DATABASE_DRIVER"`
	Host     string `mapstructure:"DATABASE_HOST"`
	Port     int    `mapstructure:"DATABASE_PORT"`
	Username string `mapstructure:"DATABASE_USERNAME"`
	Password string `mapstructure:"DATABASE_PASSWORD"`
	Database string `mapstructure:"DATABASE_DATABASE"`
	SSLMode  string `mapstructure:"DATABASE_SSLMODE"`
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret            string        `mapstructure:"JWT_SECRET"`
	Expiration        time.Duration `mapstructure:"JWT_EXPIRATION"`
	RefreshExpiration time.Duration `mapstructure:"JWT_REFRESH_EXPIRATION"`
	Issuer            string        `mapstructure:"JWT_ISSUER"`
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

// S3Config holds S3-related configuration
type S3Config struct {
	Bucket          string `mapstructure:"S3_BUCKET"`
	Region          string `mapstructure:"S3_REGION"`
	Endpoint        string `mapstructure:"S3_ENDPOINT"`
	CdnURL          string `mapstructure:"S3_CDN_URL"`
	AccessKeyID     string `mapstructure:"S3_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"S3_SECRET_ACCESS_KEY"`
}

// BleveConfig holds Bleve-related configuration
type BleveConfig struct {
	IndexPath string `mapstructure:"BLEVE_INDEX_PATH"`
	IndexType string `mapstructure:"BLEVE_INDEX_TYPE"`
}

// AdminConfig holds admin-related configuration
type AdminConfig struct {
	DefaultPassword string `mapstructure:"ADMIN_DEFAULT_PASSWORD"`
}

// LogConfig holds logging-related configuration
type LogConfig struct {
	Level      string `mapstructure:"LOG_LEVEL"`
	FilePath   string `mapstructure:"LOG_FILE_PATH"`
	MaxSize    int    `mapstructure:"LOG_MAX_SIZE"`
	MaxBackups int    `mapstructure:"LOG_MAX_BACKUPS"`
	MaxAge     int    `mapstructure:"LOG_MAX_AGE"`
	Compress   bool   `mapstructure:"LOG_COMPRESS"`
	Console    bool   `mapstructure:"LOG_CONSOLE"`
}

var cfg *Config

// Load initializes and loads the configuration from various sources.
func Load() (*Config, error) {
	v := viper.New()

	// Configure viper
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("Warning: .env file not found, using environment variables and defaults")
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	// Unmarshal and validate
	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	defaults := map[string]interface{}{
		// Server
		"SERVER_HOST":          "localhost",
		"SERVER_PORT":          8080,
		"SERVER_READ_TIMEOUT":  "30s",
		"SERVER_WRITE_TIMEOUT": "30s",
		"SERVER_IDLE_TIMEOUT":  "120s",

		// Database
		"DATABASE_DRIVER":   "postgres",
		"DATABASE_HOST":     "localhost",
		"DATABASE_PORT":     5432,
		"DATABASE_USERNAME": "postgres",
		"DATABASE_PASSWORD": "",
		"DATABASE_DATABASE": "starter",
		"DATABASE_SSLMODE":  "disable",

		// JWT
		"JWT_SECRET":             "your-secret-key",
		"JWT_EXPIRATION":         "10m",
		"JWT_REFRESH_EXPIRATION": "72h",
		"JWT_ISSUER":             "starter",

		// App
		"APP_NAME":        "Starter",
		"APP_VERSION":     "1.0.0",
		"APP_ENVIRONMENT": "development",
		"APP_DEBUG":       true,
		"APP_LOG_LEVEL":   "info",
		"APP_ASSETS":      "./assets",

		// S3
		"S3_BUCKET":            "bucket",
		"S3_REGION":            "ap-southeast-1",
		"S3_ENDPOINT":          "",
		"S3_ACCESS_KEY_ID":     "",
		"S3_SECRET_ACCESS_KEY": "",

		// Bleve
		"BLEVE_INDEX_PATH": "./bleve",
		"BLEVE_INDEX_TYPE": "scorch",

		// Admin
		"ADMIN_DEFAULT_PASSWORD": "q1w2e3r4",

		// Log
		"LOG_LEVEL":       "info",
		"LOG_FILE_PATH":   "logs/app.log",
		"LOG_MAX_SIZE":    100,
		"LOG_MAX_BACKUPS": 7,
		"LOG_MAX_AGE":     30,
		"LOG_COMPRESS":    true,
		"LOG_CONSOLE":     true,
	}

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

// validate validates the loaded configuration
func (c *Config) validate() error {
	validators := []struct {
		name string
		fn   func() error
	}{
		{"server", c.validateServer},
		{"database", c.validateDatabase},
		{"jwt", c.validateJWT},
		{"app", c.validateApp},
		{"admin", c.validateAdmin},
		{"log", c.validateLog},
	}

	for _, v := range validators {
		if err := v.fn(); err != nil {
			return fmt.Errorf("%s validation failed: %w", v.name, err)
		}
	}

	return nil
}

// validateServer validates server configuration
func (c *Config) validateServer() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1-65535)", c.Server.Port)
	}
	return nil
}

// validateDatabase validates database configuration
func (c *Config) validateDatabase() error {
	if c.Database.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

// validateJWT validates JWT configuration
func (c *Config) validateJWT() error {
	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key" {
		return fmt.Errorf("secret must be set and not use default value")
	}
	if c.JWT.Expiration <= 0 {
		return fmt.Errorf("expiration must be greater than 0")
	}
	if c.JWT.RefreshExpiration <= 0 {
		return fmt.Errorf("refresh expiration must be greater than 0")
	}
	return nil
}

// validateApp validates app configuration
func (c *Config) validateApp() error {
	if c.App.Name == "" {
		return fmt.Errorf("name is required")
	}

	validEnvironments := []string{"development", "staging", "production"}
	if !slices.Contains(validEnvironments, c.App.Environment) {
		return fmt.Errorf("invalid environment: %s (must be one of %v)",
			c.App.Environment, validEnvironments)
	}

	return nil
}

// validateAdmin validates admin configuration
func (c *Config) validateAdmin() error {
	if c.Admin.DefaultPassword == "" {
		return fmt.Errorf("default password must be set")
	}
	return nil
}

// validateLog validates log configuration
func (c *Config) validateLog() error {
	validLogLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}
	if !slices.Contains(validLogLevels, c.Log.Level) {
		return fmt.Errorf("invalid level: %s (must be one of %v)", c.Log.Level, validLogLevels)
	}

	if c.Log.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	if c.Log.MaxSize <= 0 {
		return fmt.Errorf("max size must be greater than 0")
	}

	if c.Log.MaxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative")
	}

	if c.Log.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative")
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

// GetDatabaseURL constructs and returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	urls := map[string]string{
		"postgres": fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&TimeZone=Asia/Jakarta",
			c.Database.Username, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.Database, c.Database.SSLMode),
		"mysql": fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Database.Username, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.Database),
	}

	return urls[c.Database.Driver]
}

// GetServerAddress returns the server address in host:port format
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsStaging returns true if running in staging
func (c *Config) IsStaging() bool {
	return c.App.Environment == "staging"
}

// GetLoggerConfig returns logger configuration
func (c *Config) GetLoggerConfig() LogConfig {
	return c.Log
}
