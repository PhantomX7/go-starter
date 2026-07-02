package validator

import (
	"github.com/go-playground/validator/v10"
)

// Exist checks that the value of the request field exists in the database.
//
// Tag format: exist=tablename.columnname
//
// Table and column names come from struct tags, so both are validated as
// strict SQL identifiers before being spliced into the query. A malformed tag
// or unsafe identifier fails CLOSED (the field is rejected) and logs a warning
// so the developer notices — a broken tag must never silently disable the check.
func (cv customValidator) Exist() validator.Func {
	return func(fl validator.FieldLevel) bool {
		table, column, ok := parseTableColumn("exist", fl.Param())
		if !ok {
			return false // Malformed tag: fail closed.
		}

		var count int64

		query := cv.db.Table(table).Where(column+" = ?", fl.Field().Interface())

		// Check for soft deletes - use IS NULL instead of double negative.
		// The column lookup is cached per table (see hasDeletedAtColumn).
		if cv.hasDeletedAtColumn(table) {
			query = query.Where("deleted_at IS NULL")
		}

		err := query.Count(&count).Error

		// If there's a database error, fail closed (assume doesn't exist)
		if err != nil {
			return false
		}

		return count > 0
	}
}
