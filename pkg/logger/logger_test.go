package logger_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// swapGlobalLogger replaces the package-global logger for the duration of the
// test. Tests using it must not run in parallel.
func swapGlobalLogger(t *testing.T, l *zap.Logger) {
	t.Helper()

	prev := logger.Log
	logger.Log = l
	t.Cleanup(func() { logger.Log = prev })
}

func TestInitWithValidConfig(t *testing.T) {
	swapGlobalLogger(t, logger.Log)

	// Note: nothing is logged here on purpose — lumberjack opens the file
	// lazily, and keeping it closed lets t.TempDir clean up on Windows.
	err := logger.Init(logger.Config{
		Level:    "debug",
		FilePath: filepath.Join(t.TempDir(), "logs", "app.log"),
		Console:  false,
	})

	require.NoError(t, err)
	require.NotNil(t, logger.Log)
	require.True(t, logger.Log.Core().Enabled(zap.DebugLevel))
}

func TestInitRejectsInvalidLevel(t *testing.T) {
	swapGlobalLogger(t, logger.Log)

	err := logger.Init(logger.Config{
		Level:    "verbose",
		FilePath: filepath.Join(t.TempDir(), "app.log"),
	})

	require.Error(t, err)
}

func TestSyncWithNilLoggerIsNoop(t *testing.T) {
	swapGlobalLogger(t, nil)

	require.NoError(t, logger.Sync())
}

func TestCtxReturnsNopWhenUninitialized(t *testing.T) {
	swapGlobalLogger(t, nil)

	log := logger.Ctx(context.Background())
	require.NotNil(t, log)
	// Must be safe to use even though the global logger is nil.
	log.Info("ignored")
}

func TestCtxAttachesRequestScopedFields(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	swapGlobalLogger(t, zap.New(core))

	ctx := utils.SetRequestIDToContext(context.Background(), "req-123")
	roleAdmin := "admin"
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{
		UserID: 42,
		Role:   roleAdmin,
	})

	logger.Ctx(ctx, zap.String("extra", "value")).Info("hello")

	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, "hello", entries[0].Message)

	fields := entries[0].ContextMap()
	require.Equal(t, "req-123", fields["request_id"])
	require.EqualValues(t, 42, fields["user_id"])
	require.Equal(t, roleAdmin, fields["role"])
	require.Equal(t, "value", fields["extra"])
}

func TestCtxWithPlainContextAddsOnlyCallerFields(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	swapGlobalLogger(t, zap.New(core))

	logger.Ctx(context.Background(), zap.String("extra", "value")).Info("plain")

	entries := logs.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	require.Equal(t, "value", fields["extra"])
	require.NotContains(t, fields, "request_id")
	require.NotContains(t, fields, "user_id")
	require.NotContains(t, fields, "role")
}

func TestCtxWithEmptyRoleOmitsRoleField(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	swapGlobalLogger(t, zap.New(core))

	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID: 7,
		Role:   "",
	})

	logger.Ctx(ctx).Info("no role")

	entries := logs.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	require.EqualValues(t, 7, fields["user_id"])
	require.NotContains(t, fields, "role")
}
