// Package logger provides the application's shared structured logger.
package logger

import (
	"context"
	"os"
	"path/filepath"

	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
)

// Config holds the logger configuration
type Config struct {
	// Log level (debug, info, warn, error, dpanic, panic, fatal)
	Level string
	// Log file path
	FilePath string
	// Maximum size in megabytes before rotation
	MaxSize int
	// Maximum number of old log files to retain
	MaxBackups int
	// Maximum number of days to retain old log files
	MaxAge int
	// Whether to compress rotated files
	Compress bool
	// Whether to output to console as well
	Console bool
	// Environment (development or production)
	Environment string
}

// Init initializes the global logger with the provided configuration
func Init(config Config) error {
	// Set default values
	if config.Level == "" {
		config.Level = "info"
	}
	if config.FilePath == "" {
		config.FilePath = "logs/app.log"
	}
	if config.MaxSize == 0 {
		config.MaxSize = 100 // 100MB
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 7
	}
	if config.MaxAge == 0 {
		config.MaxAge = 30 // 30 days
	}
	if config.Environment == "" {
		config.Environment = "production"
	}

	// Parse log level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		return err
	}

	// Ensure log directory exists
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return err
	}

	// Configure file rotation
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
		LocalTime:  true,
	})

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if config.Environment == "development" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Create cores
	var cores []zapcore.Core

	// File core
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(fileEncoder, fileWriter, level)
	cores = append(cores, fileCore)

	// Console core (if enabled)
	if config.Console {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// Create logger
	core := zapcore.NewTee(cores...)
	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// Sync flushes any buffered log entries
func Sync() error {
	if Log != nil {
		return Log.Sync()
	}
	return nil
}

// Helper functions for easy logging

// Debug logs a debug-level message with the shared global logger.
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Info logs an info-level message with the shared global logger.
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Warn logs a warning-level message with the shared global logger.
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error logs an error-level message with the shared global logger.
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a fatal-level message with the shared global logger and exits.
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// Panic logs a panic-level message with the shared global logger and panics.
func Panic(msg string, fields ...zap.Field) {
	Log.Panic(msg, fields...)
}

// With creates a child logger with the provided fields
func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

// Ctx returns a logger pre-bound with request-scoped fields (request_id,
// user_id, role) pulled from ctx, plus any extra fields supplied by the caller.
// Use it as the default logging entry point in handlers, services, and
// repositories to avoid re-plumbing those fields on every call site.
//
// The returned *zap.Logger is intended for direct use — log.Info(...), etc. —
// not through the package-level helpers (Info/Warn/Error). Caller-skip is
// adjusted so file:line points at the .Info/.Error call, matching what the
// global helpers report.
func Ctx(ctx context.Context, fields ...zap.Field) *zap.Logger {
	if Log == nil {
		return zap.NewNop()
	}
	if id := utils.GetRequestIDFromContext(ctx); id != "" {
		fields = append(fields, zap.String("request_id", id))
	}
	if uid, ok := utils.GetUserIDFromContext(ctx); ok {
		fields = append(fields, zap.Uint("user_id", uid))
	}
	if role, ok := utils.GetRoleFromContext(ctx); ok && role != "" {
		fields = append(fields, zap.String("role", role))
	}
	return Log.WithOptions(zap.AddCallerSkip(-1)).With(fields...)
}
