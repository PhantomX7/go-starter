package pagination

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// likeEscaper escapes the LIKE wildcard characters and the escape char itself
// so user input such as "50%" matches the literal string instead of acting as
// a wildcard. Used together with the "ESCAPE '!'" clause.
//
// '!' is the escape character rather than '\' because MySQL in its default
// SQL mode (NO_BACKSLASH_ESCAPES off) treats '\' specially inside string
// literals: the SQL fragment ESCAPE '\' is an unterminated literal there and
// the query errors out. '!' is literal in MySQL, Postgres, and SQLite, so
// the same LIKE predicate works unchanged across all three drivers in
// go.mod. Any input character that is '!' is doubled ('!!') so it survives
// the escape round-trip and matches a literal '!' in the target column.
var likeEscaper = strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`)

// Apply applies all filters, custom scopes, and pagination to query
func (p *Pagination) Apply(db *gorm.DB) *gorm.DB {
	return p.applyFilters(db).
		Limit(p.Limit).
		Offset(p.Offset).
		Order(p.Order)
}

// ApplyWithoutMeta applies only filters and custom scopes (for counting)
func (p *Pagination) ApplyWithoutMeta(db *gorm.DB) *gorm.DB {
	return p.applyFilters(db)
}

// applyFilters applies filter and custom scopes.
//
// Conditions are walked in sorted key order. Map iteration is randomised in
// Go, so without sorting the same request would emit a different WHERE clause
// each time — defeating Postgres prepared-statement plan caching, breaking
// query-log diffing, and making slow-query forensics unnecessarily painful.
func (p *Pagination) applyFilters(db *gorm.DB) *gorm.DB {
	keys := make([]string, 0, len(p.conditions))
	for k := range p.conditions {
		if _, ok := p.filterDef.filters[k]; ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, field := range keys {
		config := p.filterDef.filters[field]
		// Apply each value as a separate AND condition. HTTP query strings can
		// repeat the same key (?status=eq:a&status=eq:b); silently dropping all
		// but the first violates the principle of least surprise.
		for _, v := range p.conditions[field] {
			if scope := p.buildFilterScope(config, v); scope != nil {
				db = scope(db)
			}
		}
	}

	for _, scope := range p.scopes {
		db = scope(db)
	}

	return db
}

// buildFilterScope creates a GORM scope for a filter.
//
// Identifier safety is enforced at registration time in AddFilter — we don't
// re-check every field here, the registry is trusted. This is the hot path.
func (p *Pagination) buildFilterScope(config FilterConfig, value string) func(*gorm.DB) *gorm.DB {
	op := parseFilterOperation(value)

	// An over-cap value list fails closed: if the operator is valid for this
	// filter, emit a no-match predicate so the response is empty rather than
	// unfiltered. If the operator isn't even allowed here, drop it as usual —
	// no spurious 1 = 0 for input that would have been rejected anyway.
	if op.overflow {
		if config.allowsOperator(op.Operator) {
			return func(db *gorm.DB) *gorm.DB { return db.Where("1 = 0") }
		}
		return nil
	}

	if !p.isValidOperation(op, config) {
		return nil
	}

	fields := config.GetFields()
	if len(fields) == 0 {
		return nil
	}

	switch config.Type {
	case FilterTypeID:
		return p.buildOrderedScope(fields[0], op, parseID)
	case FilterTypeNumber:
		return p.buildOrderedScope(fields[0], op, parseNumber)
	case FilterTypeString:
		return p.buildStringScope(fields, op)
	case FilterTypeBool:
		return p.buildBoolScope(fields[0], op)
	case FilterTypeDate:
		return p.buildDateScope(fields[0], op)
	case FilterTypeDateTime:
		return p.buildDateTimeScope(fields[0], op)
	case FilterTypeEnum:
		return p.buildEnumScope(fields[0], op, config.EnumValues)
	}

	return nil
}

// nullScope emits the type-agnostic IS NULL / IS NOT NULL predicate shared by
// every single-field scope builder. (Multi-field search builds its own null
// fragment per column with the OR/AND joiner, so it does not use this.)
func nullScope(field string, op FilterOperator) func(*gorm.DB) *gorm.DB {
	if op == OperatorIsNull {
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	}
	return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
}

// buildOrderedScope handles the full numeric-style operator surface (eq, neq,
// in, not_in, between, gt, gte, lt, lte, is_null, is_not_null) using a
// caller-supplied parser. buildIDScope and buildNumericScope are thin
// wrappers around this — they share structure and differ only in how a value
// is parsed (integer-only vs int-or-float fallback). Keeping one
// implementation means a bug fix or new operator lands in one place.
func (p *Pagination) buildOrderedScope(
	field string, op FilterOperation,
	parse func(string) (any, bool),
) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" = ?", v) }
	case OperatorNotEquals:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" != ?", v) }
	case OperatorIn:
		vals := parseEach(op.Values, parse)
		// All values unparseable: the user asked for "rows matching this
		// set" and the set is empty, so the answer is zero rows. Dropping
		// the filter here would silently leak every row (matching
		// buildEnumScope's fail-closed behaviour).
		if len(vals) == 0 {
			return func(db *gorm.DB) *gorm.DB { return db.Where("1 = 0") }
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" IN ?", vals) }
	case OperatorNotIn:
		vals := parseEach(op.Values, parse)
		// Symmetric to OperatorIn: "exclude this set"; if the set is empty
		// (all unparseable), nothing is excluded — the filter is a no-op,
		// which is what returning nil effectively does.
		if len(vals) == 0 {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" NOT IN ?", vals) }
	case OperatorBetween:
		lo, ok1 := parse(op.Values[0])
		hi, ok2 := parse(op.Values[1])
		if !ok1 || !ok2 {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" BETWEEN ? AND ?", lo, hi)
		}
	case OperatorGt:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" > ?", v) }
	case OperatorGte:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" >= ?", v) }
	case OperatorLt:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" < ?", v) }
	case OperatorLte:
		v, ok := parse(op.Values[0])
		if !ok {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" <= ?", v) }
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}
	return nil
}

// buildStringScope builds string filter scope
func (p *Pagination) buildStringScope(fields []string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	if len(fields) == 1 {
		return p.buildSingleStringScope(fields[0], op)
	}
	return p.buildMultiStringScope(fields, op)
}

// buildSingleStringScope builds single field string filter
func (p *Pagination) buildSingleStringScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" = ?", op.Values[0]) }
	case OperatorNotEquals:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" != ?", op.Values[0]) }
	case OperatorLike:
		pattern := "%" + likeEscaper.Replace(op.Values[0]) + "%"
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("LOWER("+field+") LIKE LOWER(?) ESCAPE '!'", pattern)
		}
	case OperatorIn:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" IN ?", op.Values) }
	case OperatorNotIn:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" NOT IN ?", op.Values) }
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}
	return nil
}

// buildMultiStringScope builds a multi-field filter over the given fields.
//
// Matching operators (Equals / Like / In / IsNull) combine with OR — the
// classic "search anywhere" pattern. Excluding operators (NotEquals / NotIn
// / IsNotNull) combine with AND: OR-ing negations yields a predicate that is
// trivially true ("name != x OR email != x" matches every row where either
// differs, which is ~all of them). AND expresses the user's real intent —
// "the value doesn't appear in any of these fields".
//
// The SQL fragment and its bound args are assembled at scope-creation time
// so the returned closure is a thin pass to db.Where.
func (p *Pagination) buildMultiStringScope(fields []string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	joiner := " OR "
	switch op.Operator {
	case OperatorNotEquals, OperatorNotIn, OperatorIsNotNull:
		joiner = " AND "
	}

	// Hoist the LIKE pattern: identical across every field, so Replace()
	// runs once instead of len(fields) times.
	var likePattern string
	if op.Operator == OperatorLike {
		likePattern = "%" + likeEscaper.Replace(op.Values[0]) + "%"
	}

	var sb strings.Builder
	sb.WriteByte('(')
	args := make([]any, 0, len(fields))

	for i, field := range fields {
		if i > 0 {
			sb.WriteString(joiner)
		}
		switch op.Operator {
		case OperatorEquals:
			sb.WriteString(field)
			sb.WriteString(" = ?")
			args = append(args, op.Values[0])
		case OperatorNotEquals:
			sb.WriteString(field)
			sb.WriteString(" != ?")
			args = append(args, op.Values[0])
		case OperatorLike:
			sb.WriteString("LOWER(")
			sb.WriteString(field)
			sb.WriteString(") LIKE LOWER(?) ESCAPE '!'")
			args = append(args, likePattern)
		case OperatorIn:
			sb.WriteString(field)
			sb.WriteString(" IN ?")
			args = append(args, op.Values)
		case OperatorNotIn:
			sb.WriteString(field)
			sb.WriteString(" NOT IN ?")
			args = append(args, op.Values)
		case OperatorIsNull:
			sb.WriteString(field)
			sb.WriteString(" IS NULL")
		case OperatorIsNotNull:
			sb.WriteString(field)
			sb.WriteString(" IS NOT NULL")
		default:
			return nil
		}
	}
	sb.WriteByte(')')

	clause := sb.String()
	return func(db *gorm.DB) *gorm.DB { return db.Where(clause, args...) }
}

// buildBoolScope builds boolean filter scope. Accepts the same set of literals
// as strconv.ParseBool ("1", "t", "true", "TRUE", "0", "f", "false", ...).
// Anything else is treated as invalid input and the filter is dropped rather
// than silently coerced to false.
func (p *Pagination) buildBoolScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		value, err := strconv.ParseBool(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" = ?", value) }
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}
	return nil
}

// buildEnumScope builds enum filter scope.
//
// Invalid enum values are dropped *individually* rather than rejecting the
// whole filter. The previous behaviour ("any typo discards the entire
// filter") silently widened the result set to ALL rows on a single typo —
// `?role=in:admin,tpyo,user` would return everything, including roles the
// user explicitly didn't ask for. If no values remain valid, we emit a
// no-match scope so the response is empty rather than unfiltered.
func (p *Pagination) buildEnumScope(field string, op FilterOperation, allowedValues []string) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}

	valid := make([]string, 0, len(op.Values))
	for _, v := range op.Values {
		if slices.Contains(allowedValues, v) {
			valid = append(valid, v)
		}
	}
	if len(valid) == 0 {
		return func(db *gorm.DB) *gorm.DB { return db.Where("1 = 0") }
	}

	switch op.Operator {
	case OperatorEquals:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" = ?", valid[0]) }
	case OperatorIn:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" IN ?", valid) }
	case OperatorNotEquals:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" != ?", valid[0]) }
	case OperatorNotIn:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" NOT IN ?", valid) }
	}
	return nil
}
