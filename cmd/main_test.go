package main

import (
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/pkg/config"
)

// TestAppOptionsGraphResolves validates the full fx dependency graph wired in
// main without running constructors or binding any resources. It catches a
// missing/duplicate provider (e.g. a component that needs *config.Config but
// nothing supplies it) at test time instead of at process startup.
func TestAppOptionsGraphResolves(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 8080, RequestTimeout: 30 * time.Second, MaxBodyBytes: 1 << 20},
		JWT:    config.JWTConfig{Secret: "test-secret-of-at-least-32-characters!", Expiration: time.Minute, RefreshExpiration: time.Hour, Issuer: "test"},
		App:    config.AppConfig{Name: "Athleton", Environment: "development", Assets: "./assets"},
	}

	if err := fx.ValidateApp(appOptions(cfg, zap.NewNop())); err != nil {
		t.Fatalf("fx dependency graph failed to validate: %v", err)
	}
}
