package validator

import (
	"github.com/go-playground/validator/v10"
)

// Unique checks that the value of the request field is unique in the database.
//
// Tag format: unique=tablename.columnname
//
// Table and column names come from struct tags, so both are validated as
// strict SQL identifiers before being spliced into the query. A malformed tag
// or unsafe identifier fails CLOSED (the field is rejected) and logs a warning
// so the developer notices — a broken tag must never silently disable the check.
func (cv customValidator) Unique() validator.Func {
	return func(fl validator.FieldLevel) bool {
		table, column, ok := parseTableColumn("unique", fl.Param())
		if !ok {
			return false // Malformed tag: fail closed.
		}

		var count int64

		query := cv.db.Table(table).Where(column+" = ?", fl.Field().Interface())

		// Check for soft deletes.
		// The column lookup is cached per table (see hasDeletedAtColumn).
		if cv.hasDeletedAtColumn(table) {
			query = query.Where("deleted_at IS NULL")
		}

		err := query.Count(&count).Error

		// Fail closed on a database error: report "not unique" so the request is
		// rejected rather than admitting a value we could not verify. This mirrors
		// Exist (which also fails closed) — a transient DB outage should never be
		// the reason a duplicate slips past validation into the table.
		if err != nil {
			return false
		}

		return count == 0
	}
}
