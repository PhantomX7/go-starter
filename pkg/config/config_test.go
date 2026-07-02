// White-box tests for the config package. The validate method and its
// sub-validators are unexported, so this file lives in package config.
//
// Load() is intentionally not tested: it reads ./.env relative to the test
// working directory and pulls in arbitrary process environment variables via
// viper.AutomaticEnv, so its outcome depends on the developer's machine and
// cannot be made hermetic without changing production code.
package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// validConfig returns a configuration that passes every sub-validator.
func validConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "localhost",
			Port:           8080,
			RequestTimeout: 30 * time.Second,
			MaxBodyBytes:   10 << 20,
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			Host:   "localhost",
			Port:   5432,
		},
		JWT: JWTConfig{
			Secret:            "unit-test-secret-of-at-least-32-chars",
			Expiration:        10 * time.Minute,
			RefreshExpiration: 72 * time.Hour,
			Issuer:            "starter",
		},
		App: AppConfig{
			Name:        "Starter",
			Environment: "development",
		},
		Admin: AdminConfig{
			DefaultPassword: "admin-password",
		},
		Log: LogConfig{
			Level:      "info",
			FilePath:   "logs/app.log",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
		},
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	t.Parallel()

	require.NoError(t, validConfig().validate())
}

func TestValidateServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"zero port", 0, true},
		{"negative port", -1, true},
		{"port above range", 65536, true},
		{"lowest valid port", 1, false},
		{"highest valid port", 65535, false},
	}

	c := validConfig()
	c.Server.RequestTimeout = 0
	require.ErrorContains(t, c.validateServer(), "request timeout")

	c = validConfig()
	c.Server.MaxBodyBytes = 0
	require.ErrorContains(t, c.validateServer(), "max body bytes")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := validConfig()
			c.Server.Port = tt.port
			err := c.validateServer()
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "invalid port")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDatabase(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.Database.Driver = ""
	require.ErrorContains(t, c.validateDatabase(), "invalid driver")

	c = validConfig()
	c.Database.Driver = "oracle" // unsupported drivers must fail at startup, not connect time
	require.ErrorContains(t, c.validateDatabase(), "invalid driver")

	c = validConfig()
	c.Database.Host = ""
	require.ErrorContains(t, c.validateDatabase(), "host is required")

	require.NoError(t, validConfig().validateDatabase())
}

func TestValidateJWT(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.JWT.Secret = ""
	require.ErrorContains(t, c.validateJWT(), "secret must be set")

	c = validConfig()
	c.JWT.Secret = "your-secret-key" // the shipped default must be rejected
	require.ErrorContains(t, c.validateJWT(), "not use default value")

	c = validConfig()
	c.JWT.Secret = "too-short" // non-default but under the 32-char minimum
	require.ErrorContains(t, c.validateJWT(), "at least 32 characters")

	c = validConfig()
	c.JWT.Expiration = 0
	require.ErrorContains(t, c.validateJWT(), "expiration must be greater than 0")

	c = validConfig()
	c.JWT.RefreshExpiration = -time.Hour
	require.ErrorContains(t, c.validateJWT(), "refresh expiration must be greater than 0")

	// Placeholder secrets from .env.example pass the length check but are
	// public knowledge: rejected in production, tolerated (warned) in dev.
	c = validConfig()
	c.JWT.Secret = "your-super-secret-jwt-key-change-this-in-production"
	require.NoError(t, c.validateJWT())
	c.App.Environment = "production"
	require.ErrorContains(t, c.validateJWT(), "known placeholder")

	c = validConfig()
	c.JWT.MaxActiveSessions = -1
	require.ErrorContains(t, c.validateJWT(), "max active sessions")

	require.NoError(t, validConfig().validateJWT())
}

func TestValidateApp(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.App.Name = ""
	require.ErrorContains(t, c.validateApp(), "name is required")

	c = validConfig()
	c.App.Environment = "qa"
	require.ErrorContains(t, c.validateApp(), "invalid environment")

	for _, env := range []string{"development", "staging", "production"} {
		c = validConfig()
		c.App.Environment = env
		require.NoError(t, c.validateApp(), env)
	}
}

func TestValidateAdmin(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.Admin.DefaultPassword = ""
	require.ErrorContains(t, c.validateAdmin(), "default password must be set")

	// Well-known weak values are rejected in production, warned in dev.
	c = validConfig()
	c.Admin.DefaultPassword = "q1w2e3r4"
	require.NoError(t, c.validateAdmin())
	c.App.Environment = "production"
	require.ErrorContains(t, c.validateAdmin(), "weak value")

	c = validConfig()
	c.Admin.DefaultPassword = "short-pass" // 10 chars: fine in dev, too short in prod
	require.NoError(t, c.validateAdmin())
	c.App.Environment = "production"
	require.ErrorContains(t, c.validateAdmin(), "at least 12 characters")

	require.NoError(t, validConfig().validateAdmin())
}

func TestValidateLog(t *testing.T) {
	t.Parallel()

	for _, level := range []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"} {
		c := validConfig()
		c.Log.Level = level
		require.NoError(t, c.validateLog(), level)
	}

	c := validConfig()
	c.Log.Level = "verbose"
	require.ErrorContains(t, c.validateLog(), "invalid level")

	c = validConfig()
	c.Log.Level = "INFO" // whitelist is case-sensitive
	require.Error(t, c.validateLog())

	c = validConfig()
	c.Log.FilePath = ""
	require.ErrorContains(t, c.validateLog(), "file path is required")

	c = validConfig()
	c.Log.MaxSize = 0
	require.ErrorContains(t, c.validateLog(), "max size must be greater than 0")

	c = validConfig()
	c.Log.MaxBackups = -1
	require.ErrorContains(t, c.validateLog(), "max backups cannot be negative")

	c = validConfig()
	c.Log.MaxAge = -1
	require.ErrorContains(t, c.validateLog(), "max age cannot be negative")
}

func TestValidateWrapsSectionName(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.JWT.Secret = ""
	require.ErrorContains(t, c.validate(), "jwt validation failed")

	c = validConfig()
	c.Log.Level = "nope"
	require.ErrorContains(t, c.validate(), "log validation failed")
}

func TestGetDatabaseURL(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.Database = DatabaseConfig{
		Driver:   "postgres",
		Host:     "db.local",
		Port:     5432,
		Username: "user",
		Password: "pass",
		Database: "app",
		SSLMode:  "disable",
	}
	require.Equal(t,
		"postgres://user:pass@db.local:5432/app?sslmode=disable&TimeZone=Asia/Jakarta",
		c.GetDatabaseURL())

	c.Database.Driver = "mysql"
	require.Equal(t,
		"user:pass@tcp(db.local:5432)/app?charset=utf8mb4&parseTime=True&loc=Local",
		c.GetDatabaseURL())

	// Credentials with URL metacharacters must be escaped, not spliced raw
	// (p@ss@evil-host would otherwise re-parse "evil-host" as the host).
	c.Database.Driver = "postgres"
	c.Database.Password = "p@ss:w/rd%23"
	require.Equal(t,
		"postgres://user:p%40ss%3Aw%2Frd%2523@db.local:5432/app?sslmode=disable&TimeZone=Asia/Jakarta",
		c.GetDatabaseURL())

	c.Database.Driver = "oracle"
	require.Empty(t, c.GetDatabaseURL())
}

func TestServerAddressAndEnvironmentHelpers(t *testing.T) {
	t.Parallel()

	c := validConfig()
	require.Equal(t, "localhost:8080", c.GetServerAddress())

	c.App.Environment = "production"
	require.True(t, c.IsProduction())
	require.False(t, c.IsDevelopment())
	require.False(t, c.IsStaging())

	c.App.Environment = "development"
	require.True(t, c.IsDevelopment())

	c.App.Environment = "staging"
	require.True(t, c.IsStaging())

	require.Equal(t, c.Log, c.GetLoggerConfig())
}
