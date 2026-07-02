// Package config provides configuration management using Viper library.
// It supports reading from .env files, environment variables, and provides
// automatic type conversion and configuration reloading capabilities.
package config

import (
	"errors"
	"fmt"
	"log"
	"net/url"
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
	// RequestTimeout bounds a single request's handler work (its context is
	// canceled past this); distinct from ReadTimeout, which bounds reading the
	// request off the wire.
	RequestTimeout time.Duration `mapstructure:"SERVER_REQUEST_TIMEOUT"`
	// MaxBodyBytes caps the request body size accepted by the API.
	MaxBodyBytes int64 `mapstructure:"SERVER_MAX_BODY_BYTES"`
	// TrustedProxies is the set of proxy IPs/CIDRs whose X-Forwarded-For header
	// is honored when deriving the client IP (used by the rate limiters). Empty
	// means trust none — c.ClientIP() falls back to the direct RemoteAddr, so a
	// spoofed header cannot mint a fresh rate-limit bucket. Set this to your load
	// balancer's address(es) when deployed behind one. Comma-separated in env.
	TrustedProxies []string `mapstructure:"SERVER_TRUSTED_PROXIES"`
	// CORSAllowedOrigins is the allowlist of origins echoed back in
	// Access-Control-Allow-Origin. Empty means wildcard ("*"), which is only
	// safe while the API is pure bearer-token (no cookies). Comma-separated.
	CORSAllowedOrigins []string `mapstructure:"SERVER_CORS_ALLOWED_ORIGINS"`
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
	// MaxActiveSessions caps concurrent refresh-token sessions per user; the
	// oldest session is revoked when a login would exceed it. 0 disables the cap.
	MaxActiveSessions int `mapstructure:"JWT_MAX_ACTIVE_SESSIONS"`
}

// AppConfig holds general application configuration
type AppConfig struct {
	Name        string `mapstructure:"APP_NAME"`
	Version     string `mapstructure:"APP_VERSION"`
	Environment string `mapstructure:"APP_ENVIRONMENT"`
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
	// UploadACL is the canned ACL applied to uploaded objects. Keep
	// "public-read" only for assets that are meant to be world-readable
	// (e.g. images served via CDN); use "private" for anything else.
	UploadACL string `mapstructure:"S3_UPLOAD_ACL"`
}

// BleveConfig holds Bleve-related configuration
type BleveConfig struct {
	IndexPath string `mapstructure:"BLEVE_INDEX_PATH"`
}

// AdminConfig holds admin-related configuration
type AdminConfig struct {
	DefaultPassword string `mapstructure:"ADMIN_DEFAULT_PASSWORD"`
	// Email is seeded into the root account.
	Email string `mapstructure:"ADMIN_EMAIL"`
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

// Load initializes and loads the configuration from various sources. The
// returned *Config is the single instance the application wires through its
// fx container (fx.Supply); there is no process-global accessor by design.
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
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("Warning: .env file not found, using environment variables and defaults")
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	// Unmarshal and validate
	cfg := &Config{}
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
	defaults := map[string]any{
		// Server
		"SERVER_HOST":                 "localhost",
		"SERVER_PORT":                 8080,
		"SERVER_READ_TIMEOUT":         "30s",
		"SERVER_WRITE_TIMEOUT":        "30s",
		"SERVER_IDLE_TIMEOUT":         "120s",
		"SERVER_REQUEST_TIMEOUT":      "30s",
		"SERVER_MAX_BODY_BYTES":       10 << 20, // 10 MiB
		"SERVER_TRUSTED_PROXIES":      "",       // trust none by default; set to LB CIDR(s) in prod
		"SERVER_CORS_ALLOWED_ORIGINS": "",       // empty = wildcard; set explicit origins in prod

		// Database
		"DATABASE_DRIVER":   "postgres",
		"DATABASE_HOST":     "localhost",
		"DATABASE_PORT":     5432,
		"DATABASE_USERNAME": "postgres",
		"DATABASE_PASSWORD": "",
		"DATABASE_DATABASE": "starter",
		"DATABASE_SSLMODE":  "disable",

		// JWT
		"JWT_SECRET":              "your-secret-key",
		"JWT_EXPIRATION":          "10m",
		"JWT_REFRESH_EXPIRATION":  "72h",
		"JWT_ISSUER":              "starter",
		"JWT_MAX_ACTIVE_SESSIONS": 10,

		// App
		"APP_NAME":        "Starter",
		"APP_VERSION":     "1.0.0",
		"APP_ENVIRONMENT": "development",
		"APP_ASSETS":      "./assets",

		// S3
		"S3_BUCKET":            "bucket",
		"S3_REGION":            "ap-southeast-1",
		"S3_ENDPOINT":          "",
		"S3_ACCESS_KEY_ID":     "",
		"S3_SECRET_ACCESS_KEY": "",
		"S3_UPLOAD_ACL":        "public-read",

		// Bleve
		"BLEVE_INDEX_PATH": "./bleve",

		// Admin — no default on purpose: a well-known seeded password is a
		// backdoor. Must be provided via env/.env.
		"ADMIN_DEFAULT_PASSWORD": "",
		"ADMIN_EMAIL":            "root@localhost",

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
	if c.Server.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be greater than 0")
	}
	if c.Server.MaxBodyBytes <= 0 {
		return fmt.Errorf("max body bytes must be greater than 0")
	}
	return nil
}

// supportedDrivers are the drivers GetDatabaseURL can build a DSN for.
var supportedDrivers = []string{"postgres", "mysql"}

// validateDatabase validates database configuration
func (c *Config) validateDatabase() error {
	if !slices.Contains(supportedDrivers, c.Database.Driver) {
		return fmt.Errorf("invalid driver: %q (must be one of %v)", c.Database.Driver, supportedDrivers)
	}
	if c.Database.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

// placeholderJWTSecrets are values that ship in defaults or .env.example and
// must never reach production; they pass the length check but are public.
var placeholderJWTSecrets = []string{
	"your-secret-key",
	"your-super-secret-jwt-key-change-this-in-production",
}

// validateJWT validates JWT configuration
func (c *Config) validateJWT() error {
	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key" {
		return fmt.Errorf("secret must be set and not use default value")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("secret must be at least 32 characters (got %d)", len(c.JWT.Secret))
	}
	if slices.Contains(placeholderJWTSecrets, c.JWT.Secret) {
		if c.IsProduction() {
			return fmt.Errorf("secret is a known placeholder value; generate a random secret")
		}
		log.Println("WARNING: JWT_SECRET is a known placeholder value; this is rejected in production")
	}
	if c.JWT.Expiration <= 0 {
		return fmt.Errorf("expiration must be greater than 0")
	}
	if c.JWT.RefreshExpiration <= 0 {
		return fmt.Errorf("refresh expiration must be greater than 0")
	}
	if c.JWT.MaxActiveSessions < 0 {
		return fmt.Errorf("max active sessions cannot be negative")
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

// weakAdminPasswords are well-known values (former defaults, keyboard walks)
// that would hand out working superuser credentials if seeded.
var weakAdminPasswords = []string{
	"q1w2e3r4", "password", "admin", "admin123", "12345678", "changeme",
}

// validateAdmin validates admin configuration
func (c *Config) validateAdmin() error {
	if c.Admin.DefaultPassword == "" {
		return fmt.Errorf("default password must be set (no default is provided on purpose)")
	}
	if slices.Contains(weakAdminPasswords, strings.ToLower(c.Admin.DefaultPassword)) {
		if c.IsProduction() {
			return fmt.Errorf("default password is a well-known weak value; choose a strong password")
		}
		log.Println("WARNING: ADMIN_DEFAULT_PASSWORD is a well-known weak value; this is rejected in production")
	}
	if len(c.Admin.DefaultPassword) < 12 && c.IsProduction() {
		return fmt.Errorf("default password must be at least 12 characters in production")
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

// GetDatabaseURL constructs and returns the database connection URL.
// Credentials are URL-escaped so passwords containing @ : / % # cannot
// corrupt the DSN (or silently redirect the host portion).
func (c *Config) GetDatabaseURL() string {
	switch c.Database.Driver {
	case "postgres":
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(c.Database.Username, c.Database.Password),
			Host:     fmt.Sprintf("%s:%d", c.Database.Host, c.Database.Port),
			Path:     c.Database.Database,
			RawQuery: fmt.Sprintf("sslmode=%s&TimeZone=Asia/Jakarta", c.Database.SSLMode),
		}
		return u.String()
	case "mysql":
		// go-sql-driver DSN: the username must not contain ':' or '@'; the
		// password is read up to the last '@' so it tolerates special chars.
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Database.Username, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.Database)
	default:
		// unreachable: validateDatabase rejects unknown drivers at startup
		return ""
	}
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
