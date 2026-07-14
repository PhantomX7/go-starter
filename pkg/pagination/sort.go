package pagination

import (
	"strings"
)

// isIdent reports whether s is a safe SQL identifier: one letter-or-underscore
// followed by letters, digits, or underscores. Hand-rolled rather than regex
// because this runs on the hot path (once per sort part) and the regex
// engine's overhead is unjustified for such a tight grammar.
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

// isQualifiedIdent reports whether s is either `col` or `table.col`, with each
// segment satisfying isIdent. Empty segments (`a..b`, leading/trailing dot)
// are rejected.
func isQualifiedIdent(s string) bool {
	if s == "" {
		return false
	}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			if !isIdent(s[start:i]) {
				return false
			}
			start = i + 1
		}
	}
	return isIdent(s[start:])
}

// isValidDirection reports whether s is a sort direction ("asc"/"desc",
// case-insensitive). Shared by isValidOrderLiteral and parseOrder, which apply
// the same rule to developer config and user ?sort= respectively.
func isValidDirection(s string) bool {
	d := strings.ToLower(s)
	return d == "asc" || d == "desc"
}

// isValidOrderLiteral checks that s is a syntactically safe ORDER BY clause:
// "[table.]field [asc|desc]" parts, comma-separated, with identifiers that
// pass isIdent. Used to gate PaginationOptions.DefaultOrder so developer
// config (YAML, env vars, etc.) cannot inject through GORM's un-escaped
// Order() even by accident.
func isValidOrderLiteral(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.Split(s, ",")
	if len(parts) > maxSortParts {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return false
		}
		tokens := strings.Fields(p)
		if len(tokens) == 0 || len(tokens) > 2 {
			return false
		}
		if !isQualifiedIdent(tokens[0]) {
			return false
		}
		if len(tokens) == 2 && !isValidDirection(tokens[1]) {
			return false
		}
	}
	return true
}

// parseOrder parses the ?sort= query parameter into a SQL ORDER BY clause.
//
// The clause is *reconstructed* from validated parts rather than passed
// through, because GORM's Order() does not escape its argument. A bypass like
// `?sort=id desc; DROP TABLE users--` would otherwise reach the database.
//
// Each comma-separated part must be of the form "[table.]field [asc|desc]".
// Field and table names must match identRe and must be allow-listed in the
// FilterDefinition. If any part fails validation, the whole sort is rejected
// and the default order is used (fail closed).
func parseOrder(conditions map[string][]string, defaultOrder string, filterDef *FilterDefinition) string {
	orderStr, exists := conditions[QueryKeySort]
	if !exists || len(orderStr) == 0 || orderStr[0] == "" {
		return defaultOrder
	}

	parts := strings.Split(orderStr[0], ",")
	if len(parts) > maxSortParts {
		return defaultOrder
	}
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tokens := strings.Fields(part)
		if len(tokens) == 0 || len(tokens) > 2 {
			return defaultOrder
		}

		fieldRaw := tokens[0]
		direction := "asc"
		if len(tokens) == 2 {
			if !isValidDirection(tokens[1]) {
				return defaultOrder
			}
			direction = strings.ToLower(tokens[1])
		}

		tableName := ""
		fieldName := fieldRaw
		if tbl, fld, found := strings.Cut(fieldRaw, "."); found {
			tableName = tbl
			fieldName = fld
		}

		if !isIdent(fieldName) {
			return defaultOrder
		}
		if tableName != "" && !isIdent(tableName) {
			return defaultOrder
		}

		cfg, ok := filterDef.sorts[fieldName]
		if !ok || !cfg.Allowed {
			return defaultOrder
		}
		// A table prefix in the request is only honored when the SortConfig
		// declares one and they match. Allowing arbitrary prefixes broadens
		// the surface for crafted inputs and joins the request couldn't
		// otherwise reach.
		if tableName != "" && tableName != cfg.TableName {
			return defaultOrder
		}

		// Resolve the user-facing sort key to the actual DB column. This is
		// what makes ?sort=created work when the config declares
		// {Field: "created_at"} — previously we emitted the request key
		// verbatim and SortConfig.Field was dead.
		emittedField := fieldName
		if cfg.Field != "" {
			emittedField = cfg.Field
		}

		var b strings.Builder
		if tableName != "" {
			b.WriteString(tableName)
			b.WriteByte('.')
		}
		b.WriteString(emittedField)
		b.WriteByte(' ')
		b.WriteString(direction)
		out = append(out, b.String())
	}

	if len(out) == 0 {
		return defaultOrder
	}
	return strings.Join(out, ", ")
}
