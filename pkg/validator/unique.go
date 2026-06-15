package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// check if value of request is unique in database
// tag format : unique=tablename.columnname
func (cv customValidator) Unique() validator.Func {
	return func(fl validator.FieldLevel) bool {
		var count int64

		arr := strings.Split(fl.Param(), ".")
		// Validate parameter format
		if len(arr) != 2 {
			return true // Invalid format, validation passes (fail open)
		}

		table, column := arr[0], arr[1]

		query := cv.db.Table(table).Where(column+" = ?", fl.Field().Interface())

		// Check for soft deletes
		if cv.db.Migrator().HasColumn(table, "deleted_at") {
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
