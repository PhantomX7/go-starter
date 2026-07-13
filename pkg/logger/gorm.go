package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger adapts gorm's logger interface onto the shared zap logger, so
// database logs are JSON-structured, request-correlated (request_id), and
// captured by the same file rotation as the rest of the application — the
// stdlib logger gorm ships writes plain text to stdout only.
//
// It logs query errors at Error level and queries slower than SlowThreshold
// at Warn level. It also implements gorm's ParamsFilter so bind values (which
// may contain sensitive data) never appear in logged SQL.
type GormLogger struct {
	// SlowThreshold is the duration above which a successful query is logged
	// as slow. Zero disables slow-query logging.
	SlowThreshold time.Duration
	// IgnoreRecordNotFoundError suppresses gorm.ErrRecordNotFound, which is an
	// expected outcome (mapped to 404s), not a database failure.
	IgnoreRecordNotFoundError bool

	level gormlogger.LogLevel
}

// NewGormLogger returns a GormLogger at Warn level (errors + slow queries),
// which is the right production default: per-query trace logging stays off.
func NewGormLogger(slowThreshold time.Duration) *GormLogger {
	return &GormLogger{
		SlowThreshold:             slowThreshold,
		IgnoreRecordNotFoundError: true,
		level:                     gormlogger.Warn,
	}
}

// LogMode implements gormlogger.Interface. It returns a copy so per-session
// overrides never mutate the shared instance.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

// Info implements gormlogger.Interface.
func (l *GormLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		Ctx(ctx).Sugar().Infof(msg, args...)
	}
}

// Warn implements gormlogger.Interface.
func (l *GormLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		Ctx(ctx).Sugar().Warnf(msg, args...)
	}
}

// Error implements gormlogger.Interface.
func (l *GormLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		Ctx(ctx).Sugar().Errorf(msg, args...)
	}
}

// Trace implements gormlogger.Interface. fc is only invoked when something
// will actually be logged, so the happy path pays no SQL-formatting cost.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= gormlogger.Error &&
		(!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := fc()
		Ctx(ctx).Error("database query failed",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	case l.SlowThreshold > 0 && elapsed >= l.SlowThreshold && l.level >= gormlogger.Warn:
		sql, rows := fc()
		Ctx(ctx).Warn("slow database query",
			zap.Duration("elapsed", elapsed),
			zap.Duration("threshold", l.SlowThreshold),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
	case l.level >= gormlogger.Info:
		sql, rows := fc()
		Ctx(ctx).Info("database query",
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
	}
}

// ParamsFilter implements gormlogger.ParamsFilter: returning nil params makes
// gorm log the parameterized SQL instead of interpolating bind values into it.
func (l *GormLogger) ParamsFilter(_ context.Context, sql string, _ ...any) (string, []any) {
	return sql, nil
}
