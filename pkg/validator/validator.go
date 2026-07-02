// Package validator provides custom go-playground validators used by the application.
package validator

import (
	"strings"
	"sync"

	"github.com/PhantomX7/athleton/pkg/logger"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CustomValidator exposes the custom validator functions registered by the app.
type CustomValidator interface {
	Unique() validator.Func
	Exist() validator.Func
	FileSize() validator.Func
	FileExtension() validator.Func
	FileMimeType() validator.Func
}

type customValidator struct {
	validator *validator.Validate
	db        *gorm.DB

	// deletedAtCache memoizes whether a table has a deleted_at column so the
	// Exist/Unique validators do not hit information_schema on every request
	// field. Key: table name (string), value: bool.
	deletedAtCache *sync.Map
}

// New creates a CustomValidator backed by the provided database connection.
func New(db *gorm.DB) CustomValidator {
	return &customValidator{
		validator:      validator.New(),
		db:             db,
		deletedAtCache: &sync.Map{},
	}
}

// isIdent reports whether s is a safe SQL identifier: one letter-or-underscore
// followed by letters, digits, or underscores. This mirrors the identifier
// grammar enforced by pkg/pagination so table/column names taken from struct
// tags can never smuggle SQL into the query built by Exist/Unique.
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c == '_':
			// always allowed
		case c >= '0' && c <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// parseTableColumn parses a "table.column" tag parameter and validates both
// segments as strict SQL identifiers. A malformed parameter is a developer
// error, so it is logged loudly; callers must fail CLOSED (reject the value)
// when ok is false — silently skipping the check would disable validation.
func parseTableColumn(tag, param string) (table, column string, ok bool) {
	arr := strings.Split(param, ".")
	if len(arr) != 2 || !isIdent(arr[0]) || !isIdent(arr[1]) {
		warn("invalid validator tag parameter: expected \"table.column\" with safe identifiers; failing closed",
			zap.String("tag", tag),
			zap.String("param", param),
		)
		return "", "", false
	}
	return arr[0], arr[1], true
}

// hasDeletedAtColumn reports whether table has a deleted_at column, caching
// the migrator lookup per table so repeated validations do not round-trip to
// information_schema.
func (cv customValidator) hasDeletedAtColumn(table string) bool {
	if v, found := cv.deletedAtCache.Load(table); found {
		if b, isBool := v.(bool); isBool {
			return b
		}
	}
	has := cv.db.Migrator().HasColumn(table, "deleted_at")
	cv.deletedAtCache.Store(table, has)
	return has
}

// warn logs through the shared application logger, tolerating an
// uninitialized logger (e.g. in unit tests) instead of panicking.
func warn(msg string, fields ...zap.Field) {
	if logger.Log == nil {
		return
	}
	logger.Log.Warn(msg, fields...)
}
