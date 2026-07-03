package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"go.uber.org/zap"
)

// Unique checks that the value of the request field is unique in the database.
//
// Tag formats:
//
//	unique=table.column                 — plain uniqueness (create)
//	unique=table.column.idcolumn.IDField — self-excluding uniqueness (update)
//
// The self-excluding form ignores the row whose idcolumn equals the value of
// the sibling struct field IDField, so a "PUT the whole object back" with an
// unchanged unique value no longer conflicts with the record itself. IDField
// must be populated from a trusted source (e.g. the path param) before
// validation, not from the request body. When IDField is absent or zero the
// check degrades to plain uniqueness, so the same tag is safe on create.
//
// All identifiers come from struct tags and are validated as strict SQL
// identifiers before being spliced into the query. A malformed tag or unsafe
// identifier fails CLOSED (the field is rejected) and logs a warning so the
// developer notices — a broken tag must never silently disable the check.
func (cv customValidator) Unique() validator.Func {
	return func(fl validator.FieldLevel) bool {
		table, column, idColumn, idField, ok := parseUniqueTag(fl.Param())
		if !ok {
			return false // Malformed tag: fail closed.
		}

		query := cv.db.Table(table).Where(column+" = ?", fl.Field().Interface())

		// Check for soft deletes.
		// The column lookup is cached per table (see hasDeletedAtColumn).
		if cv.hasDeletedAtColumn(table) {
			query = query.Where("deleted_at IS NULL")
		}

		// Self-exclusion (update DTOs): drop the record's own row from the
		// uniqueness check. A missing/zero id means "no record yet", so the
		// check behaves exactly like the plain create form.
		if idColumn != "" {
			if excludeID, hasID := siblingFieldValue(fl, idField); hasID {
				query = query.Where(idColumn+" != ?", excludeID)
			}
		}

		var count int64
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

// parseUniqueTag parses the unique tag parameter. It accepts either the 2-part
// "table.column" form or the 4-part "table.column.idcolumn.IDField"
// self-exclusion form, validating every segment as a strict SQL/Go identifier.
// Any other shape is a developer error and fails closed (ok == false).
func parseUniqueTag(param string) (table, column, idColumn, idField string, ok bool) {
	arr := strings.Split(param, ".")
	switch len(arr) {
	case 2:
		if !isIdent(arr[0]) || !isIdent(arr[1]) {
			break
		}
		return arr[0], arr[1], "", "", true
	case 4:
		if !isIdent(arr[0]) || !isIdent(arr[1]) || !isIdent(arr[2]) || !isIdent(arr[3]) {
			break
		}
		return arr[0], arr[1], arr[2], arr[3], true
	}

	warn("invalid unique tag parameter: expected \"table.column\" or "+
		"\"table.column.idcolumn.IDField\" with safe identifiers; failing closed",
		zap.String("param", param),
	)
	return "", "", "", "", false
}

// siblingFieldValue reads the value of the named sibling field from the struct
// currently being validated (fl.Parent()). It returns (value, true) only when
// the field exists and holds a non-zero value; a missing or zero field yields
// (nil, false) so the caller skips self-exclusion and treats the row as new.
func siblingFieldValue(fl validator.FieldLevel, fieldName string) (any, bool) {
	parent := fl.Parent()
	for parent.Kind() == reflect.Pointer {
		if parent.IsNil() {
			return nil, false
		}
		parent = parent.Elem()
	}
	if parent.Kind() != reflect.Struct {
		return nil, false
	}

	fv := parent.FieldByName(fieldName)
	if !fv.IsValid() {
		return nil, false
	}
	for fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return nil, false
		}
		fv = fv.Elem()
	}
	if fv.IsZero() {
		return nil, false
	}
	return fv.Interface(), true
}
