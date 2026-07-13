package logger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func newObservedGormLogger(t *testing.T, slowThreshold time.Duration) (*logger.GormLogger, *observer.ObservedLogs) {
	t.Helper()

	core, observed := observer.New(zapcore.DebugLevel)
	swapGlobalLogger(t, zap.New(core))
	return logger.NewGormLogger(slowThreshold), observed
}

func TestGormLoggerLogsQueryErrorsAtErrorLevel(t *testing.T) {
	gl, observed := newObservedGormLogger(t, time.Second)

	queryErr := errors.New("connection refused")
	ctx := utils.SetRequestIDToContext(context.Background(), "req-9")

	gl.Trace(ctx, time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = $1", 0
	}, queryErr)

	entries := observed.All()
	require.Len(t, entries, 1)
	require.Equal(t, zapcore.ErrorLevel, entries[0].Level)
	fields := entries[0].ContextMap()
	require.Equal(t, "SELECT * FROM users WHERE id = $1", fields["sql"])
	require.Equal(t, "connection refused", fields["error"])
	// The request ID must be preserved so a failed query stays correlated to
	// its request.
	require.Equal(t, "req-9", fields["request_id"])
}

func TestGormLoggerLogsSlowQueriesAtWarnLevel(t *testing.T) {
	gl, observed := newObservedGormLogger(t, 10*time.Millisecond)

	begin := time.Now().Add(-50 * time.Millisecond)
	gl.Trace(context.Background(), begin, func() (string, int64) {
		return "SELECT * FROM logs", 120
	}, nil)

	entries := observed.All()
	require.Len(t, entries, 1)
	require.Equal(t, zapcore.WarnLevel, entries[0].Level)
	fields := entries[0].ContextMap()
	require.Equal(t, "SELECT * FROM logs", fields["sql"])
	require.NotZero(t, fields["elapsed"])
}

func TestGormLoggerIgnoresRecordNotFound(t *testing.T) {
	gl, observed := newObservedGormLogger(t, time.Second)

	gl.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users WHERE id = $1", 0
	}, gorm.ErrRecordNotFound)

	require.Empty(t, observed.All())
}

func TestGormLoggerIsQuietForFastSuccessfulQueries(t *testing.T) {
	gl, observed := newObservedGormLogger(t, time.Second)

	gl.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	require.Empty(t, observed.All())
}

func TestGormLoggerLogModeSilentSuppressesEverything(t *testing.T) {
	gl, observed := newObservedGormLogger(t, time.Nanosecond)

	silent := gl.LogMode(gormlogger.Silent)
	silent.Trace(context.Background(), time.Now().Add(-time.Second), func() (string, int64) {
		return "SELECT * FROM logs", 0
	}, errors.New("boom"))

	require.Empty(t, observed.All())
	// LogMode must return a copy, leaving the original logger untouched.
	gl.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM users", 0
	}, errors.New("boom"))
	require.Len(t, observed.All(), 1)
}

func TestGormLoggerParamsFilterStripsQueryValues(t *testing.T) {
	gl, _ := newObservedGormLogger(t, time.Second)

	// Implementing gorm's ParamsFilter with nil params keeps bind values (which
	// may contain sensitive data) out of every logged SQL statement.
	sql, params := gl.ParamsFilter(context.Background(), "SELECT * FROM users WHERE email = $1", "secret@example.com")

	require.Equal(t, "SELECT * FROM users WHERE email = $1", sql)
	require.Nil(t, params)
}
