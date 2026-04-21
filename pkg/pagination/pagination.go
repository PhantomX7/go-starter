// Package pagination provides safe, typed pagination and filtering helpers for GORM queries.
package pagination

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "time/tzdata" // ensures Asia/Jakarta resolves on systems without OS tzdata (e.g. Windows containers)

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TypedColumn is anything that can describe itself as a clause.Column. Every
// scalar helper the GORM CLI generates (field.String / field.Number[T] /
// field.Bool / field.Time / field.Bytes / field.Field[T]) satisfies this via
// its Column() method, so callers can write
//
//	Column: generated.User.Email
//
// instead of Field: "email" — a rename in internal/models breaks the filter
// registry at compile time after the next `make gorm-gen`.
type TypedColumn interface {
	Column() clause.Column
}

// Defensive caps to keep a single request's parse work bounded. Requests that
// exceed these are rejected — a caller sending ?sort=a,b,…×100_000 would
// otherwise allocate and validate on every one before the DB ever sees it.
const (
	maxFilterValues = 256 // max comma-separated values in an IN/NOT IN list
	maxSortParts    = 16  // max comma-separated parts in ?sort=
)

// Query parameter keys
const (
	QueryKeyLimit  = "limit"
	QueryKeyOffset = "offset"
	QueryKeySort   = "sort"
)

// FilterType represents the data type for filtering
type FilterType string

// Supported filter value types.
const (
	FilterTypeID       FilterType = "ID"
	FilterTypeNumber   FilterType = "NUMBER"
	FilterTypeString   FilterType = "STRING"
	FilterTypeBool     FilterType = "BOOL"
	FilterTypeDate     FilterType = "DATE"
	FilterTypeDateTime FilterType = "DATETIME"
	FilterTypeEnum     FilterType = "ENUM"
)

// FilterOperator represents filter operations
type FilterOperator string

// Supported filter operators.
const (
	OperatorEquals    FilterOperator = "eq"
	OperatorNotEquals FilterOperator = "neq"
	OperatorIn        FilterOperator = "in"
	OperatorNotIn     FilterOperator = "not_in"
	OperatorLike      FilterOperator = "like"
	OperatorBetween   FilterOperator = "between"
	OperatorGt        FilterOperator = "gt"
	OperatorGte       FilterOperator = "gte"
	OperatorLt        FilterOperator = "lt"
	OperatorLte       FilterOperator = "lte"
	OperatorIsNull    FilterOperator = "is_null"
	OperatorIsNotNull FilterOperator = "is_not_null"
)

// knownOperators maps the wire-format prefix to the operator. Used to decide
// whether a leading "<token>:" is actually an operator or just part of the value
// (e.g. a URL like "http://x").
var knownOperators = map[string]FilterOperator{
	string(OperatorEquals):    OperatorEquals,
	string(OperatorNotEquals): OperatorNotEquals,
	string(OperatorIn):        OperatorIn,
	string(OperatorNotIn):     OperatorNotIn,
	string(OperatorLike):      OperatorLike,
	string(OperatorBetween):   OperatorBetween,
	string(OperatorGt):        OperatorGt,
	string(OperatorGte):       OperatorGte,
	string(OperatorLt):        OperatorLt,
	string(OperatorLte):       OperatorLte,
	string(OperatorIsNull):    OperatorIsNull,
	string(OperatorIsNotNull): OperatorIsNotNull,
}

// operatorsByType defines valid operators per filter type
var operatorsByType = map[FilterType][]FilterOperator{
	FilterTypeID:       {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorBetween, OperatorGt, OperatorGte, OperatorLt, OperatorLte, OperatorIsNull, OperatorIsNotNull},
	FilterTypeNumber:   {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorBetween, OperatorGt, OperatorGte, OperatorLt, OperatorLte, OperatorIsNull, OperatorIsNotNull},
	FilterTypeString:   {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorLike, OperatorIsNull, OperatorIsNotNull},
	FilterTypeBool:     {OperatorEquals, OperatorIsNull, OperatorIsNotNull},
	FilterTypeDate:     {OperatorEquals, OperatorBetween, OperatorGte, OperatorLte, OperatorIsNull, OperatorIsNotNull},
	FilterTypeDateTime: {OperatorEquals, OperatorBetween, OperatorGte, OperatorLte, OperatorIsNull, OperatorIsNotNull},
	FilterTypeEnum:     {OperatorEquals, OperatorIn, OperatorIsNull, OperatorIsNotNull},
}

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

// FilterConfig defines filterable field configuration.
//
// Prefer the typed Column / SearchColumns for columns that exist on generated
// models — a rename in internal/models will break registration at compile
// time. Use the string Field / SearchFields for joined columns, computed
// expressions, or anything outside the generator's reach.
//
// When both are supplied, the typed value wins.
type FilterConfig struct {
	// Typed, preferred. Satisfied by generated scalar field helpers
	// (field.String, field.Number[T], field.Bool, field.Time, ...).
	Column        TypedColumn
	SearchColumns []TypedColumn

	// String escape hatch. Keep empty when Column/SearchColumns is set.
	Field        string
	SearchFields []string

	Type       FilterType
	TableName  string
	Operators  []FilterOperator
	EnumValues []string

	// resolvedFields caches the output of computeFields. Populated by
	// AddFilter at registration time; GetFields returns it when set so the
	// hot path skips the allocation every request.
	resolvedFields []string
}

// normalizeTyped promotes typed Column/SearchColumns into the string Field/
// SearchFields path used by the rest of the engine. Returns a copy so we
// don't mutate the caller's struct. Also inherits TableName from the typed
// column's Table when the caller didn't set one explicitly.
func (fc FilterConfig) normalizeTyped() FilterConfig {
	if fc.Column != nil {
		col := fc.Column.Column()
		fc.Field = col.Name
		if fc.TableName == "" && col.Table != "" {
			fc.TableName = col.Table
		}
	}
	if len(fc.SearchColumns) > 0 {
		fields := make([]string, 0, len(fc.SearchColumns))
		for _, c := range fc.SearchColumns {
			if c == nil {
				continue
			}
			col := c.Column()
			if col.Table != "" && !strings.Contains(col.Name, ".") {
				fields = append(fields, col.Table+"."+col.Name)
			} else {
				fields = append(fields, col.Name)
			}
		}
		fc.SearchFields = fields
	}
	return fc
}

// GetAllowedOperators returns the operators valid for this filter. The result
// is a fresh slice — callers may mutate it without poisoning the shared
// package-level operatorsByType lookup table or the config's own Operators.
func (fc FilterConfig) GetAllowedOperators() []FilterOperator {
	if len(fc.Operators) > 0 {
		return slices.Clone(fc.Operators)
	}
	return slices.Clone(operatorsByType[fc.Type])
}

// GetFields returns the fully-resolved (optionally table-prefixed) field
// list. When called via a registered FilterDefinition this is a cache hit;
// direct callers (typically in tests) still get the computed result.
func (fc FilterConfig) GetFields() []string {
	if fc.resolvedFields != nil {
		return fc.resolvedFields
	}
	return fc.computeFields()
}

// computeFields does the actual prefix computation. Separated so AddFilter
// can populate the cache and GetFields can be a cheap conditional read.
func (fc FilterConfig) computeFields() []string {
	fields := fc.SearchFields
	if len(fields) == 0 {
		fields = []string{fc.Field}
	}

	if fc.TableName == "" {
		return fields
	}

	prefixed := make([]string, len(fields))
	for i, field := range fields {
		if strings.Contains(field, ".") {
			prefixed[i] = field
		} else {
			prefixed[i] = fc.TableName + "." + field
		}
	}
	return prefixed
}

// SortConfig defines sortable field configuration.
//
// Prefer the typed Column for columns that exist on generated models; use the
// string Field for joined columns or computed expressions. When both are
// supplied, the typed value wins.
type SortConfig struct {
	// Typed, preferred.
	Column TypedColumn

	// String escape hatch. Keep empty when Column is set.
	Field string

	TableName string
	Allowed   bool
}

func (sc SortConfig) normalizeTyped() SortConfig {
	if sc.Column != nil {
		col := sc.Column.Column()
		sc.Field = col.Name
		if sc.TableName == "" && col.Table != "" {
			sc.TableName = col.Table
		}
	}
	return sc
}

// FilterDefinition holds filter and sort configurations
type FilterDefinition struct {
	filters map[string]FilterConfig
	sorts   map[string]SortConfig
}

// NewFilterDefinition creates a new filter definition
func NewFilterDefinition() *FilterDefinition {
	return &FilterDefinition{
		filters: make(map[string]FilterConfig),
		sorts:   make(map[string]SortConfig),
	}
}

// AddFilter adds a filter configuration (chainable).
//
// Configurations with unsafe identifiers (Field, TableName, or any
// SearchFields entry) are silently dropped. This is deliberately quiet
// rather than a panic: a registration bug shouldn't take down a production
// server, and the affected filter will simply no-op on its query param —
// equivalent behaviour to an unknown filter key. The check runs once per
// startup; the hot path trusts the registry.
func (fd *FilterDefinition) AddFilter(name string, config FilterConfig) *FilterDefinition {
	config = config.normalizeTyped()
	if config.TableName != "" && !isIdent(config.TableName) {
		return fd
	}
	fields := config.computeFields()
	if len(fields) == 0 {
		return fd
	}
	for _, f := range fields {
		if !isQualifiedIdent(f) {
			return fd
		}
	}
	// Populate cache. We store a value copy in the map, so this assignment
	// doesn't mutate the caller's FilterConfig.
	config.resolvedFields = fields
	fd.filters[name] = config
	return fd
}

// AddSort adds a sort configuration (chainable). Unsafe identifiers are
// silently dropped for the same reason as AddFilter.
func (fd *FilterDefinition) AddSort(name string, config SortConfig) *FilterDefinition {
	config = config.normalizeTyped()
	if !isIdent(name) {
		return fd
	}
	if config.Field != "" && !isIdent(config.Field) {
		return fd
	}
	if config.TableName != "" && !isIdent(config.TableName) {
		return fd
	}
	fd.sorts[name] = config
	return fd
}

// PaginationOptions configures pagination behavior.
//
//nolint:revive // PaginationOptions is kept for external API compatibility.
type PaginationOptions struct {
	DefaultLimit int
	MaxLimit     int
	DefaultOrder string
	Timezone     *time.Location
}

// safeDefaultOrder is the fallback used when a caller-supplied DefaultOrder
// is empty or malformed. We intentionally don't validate against the
// FilterDefinition sort registry here — DefaultOrder often references a
// primary key the caller doesn't bother to register as a user-visible sort.
const safeDefaultOrder = "id desc"

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
		if len(tokens) == 2 {
			d := strings.ToLower(tokens[1])
			if d != "asc" && d != "desc" {
				return false
			}
		}
	}
	return true
}

// defaultTimezone resolves the canonical default timezone, falling back to UTC
// rather than nil if the location cannot be loaded — a nil *time.Location
// passed to time.ParseInLocation panics.
func defaultTimezone() *time.Location {
	if tz, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return tz
	}
	return time.UTC
}

// DefaultPaginationOptions returns sensible defaults
func DefaultPaginationOptions() PaginationOptions {
	return PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultOrder: "id desc",
		Timezone:     defaultTimezone(),
	}
}

// Pagination manages query pagination and filtering
type Pagination struct {
	Limit      int
	Offset     int
	Order      string
	conditions map[string][]string
	filterDef  *FilterDefinition
	options    PaginationOptions
	scopes     []func(*gorm.DB) *gorm.DB
}

// NewPagination creates a pagination instance.
//
// The caller's conditions map is deep-copied so subsequent mutations of the
// original (including mutations of the inner []string values) cannot change
// the query shape the Pagination was constructed with.
func NewPagination(conditions map[string][]string, filterDef *FilterDefinition, options PaginationOptions) *Pagination {
	if filterDef == nil {
		filterDef = NewFilterDefinition()
	}

	if options.DefaultLimit <= 0 {
		options.DefaultLimit = 20
	}
	if options.MaxLimit <= 0 {
		options.MaxLimit = 100
	}
	// MaxLimit is the hard cap. A caller-configured DefaultLimit that exceeds
	// it would otherwise slip through for requests without ?limit=, silently
	// widening the response past the stated cap.
	if options.DefaultLimit > options.MaxLimit {
		options.DefaultLimit = options.MaxLimit
	}
	if !isValidOrderLiteral(options.DefaultOrder) {
		options.DefaultOrder = safeDefaultOrder
	}
	if options.Timezone == nil {
		options.Timezone = defaultTimezone()
	}

	internal := deepCloneConditions(conditions)

	return &Pagination{
		conditions: internal,
		filterDef:  filterDef,
		options:    options,
		Limit:      parseLimit(internal, options.DefaultLimit, options.MaxLimit),
		Offset:     parseOffset(internal),
		Order:      parseOrder(internal, options.DefaultOrder, filterDef),
		scopes:     make([]func(*gorm.DB) *gorm.DB, 0),
	}
}

// deepCloneConditions copies the outer map AND each inner []string so the
// result shares no mutable state with the input. Used at both ingress
// (NewPagination) and egress (GetConditions).
func deepCloneConditions(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[k] = slices.Clone(v)
	}
	return out
}

// AddCustomScope adds custom GORM scopes
func (p *Pagination) AddCustomScope(scopes ...func(*gorm.DB) *gorm.DB) *Pagination {
	p.scopes = append(p.scopes, scopes...)
	return p
}

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

// GetConditions returns a deep clone of the query conditions so callers can
// freely mutate either the outer map or any inner []string without affecting
// the pagination's internal state.
func (p *Pagination) GetConditions() map[string][]string {
	return deepCloneConditions(p.conditions)
}

// GetPage returns the current page number (1-indexed)
func (p *Pagination) GetPage() int {
	if p.Limit <= 0 {
		return 1
	}
	return (p.Offset / p.Limit) + 1
}

// GetPageSize returns the page size
func (p *Pagination) GetPageSize() int {
	return p.Limit
}

// GetTotalPages calculates total pages from total count
func (p *Pagination) GetTotalPages(total int64) int {
	if p.Limit <= 0 || total <= 0 {
		return 0
	}
	limit := int64(p.Limit)
	pages := (total + limit - 1) / limit
	return int(pages)
}

// FilterOperation represents a parsed filter
type FilterOperation struct {
	Operator FilterOperator
	Values   []string
}

// parseFilterOperation parses "operator:value" format. The "operator:" prefix
// is only stripped when the prefix matches a known operator — this prevents
// values that legitimately contain colons (URLs, timestamps, "ns:metric")
// from being misparsed. is_null/is_not_null may also appear with no colon.
func parseFilterOperation(value string) FilterOperation {
	if op, ok := knownOperators[value]; ok {
		if op == OperatorIsNull || op == OperatorIsNotNull {
			return FilterOperation{Operator: op}
		}
	}

	if prefix, rest, found := strings.Cut(value, ":"); found && prefix != "" {
		if op, ok := knownOperators[prefix]; ok {
			if op == OperatorIsNull || op == OperatorIsNotNull {
				return FilterOperation{Operator: op}
			}
			// Cap the value list to bound parse/allocation cost. A request
			// over the limit becomes a nil-operator FilterOperation which
			// fails isValidOperation and is silently dropped.
			values := strings.Split(rest, ",")
			if len(values) > maxFilterValues {
				return FilterOperation{}
			}
			return FilterOperation{Operator: op, Values: values}
		}
	}

	return FilterOperation{
		Operator: OperatorEquals,
		Values:   []string{value},
	}
}

// buildFilterScope creates a GORM scope for a filter.
//
// Identifier safety is enforced at registration time in AddFilter — we don't
// re-check every field here, the registry is trusted. This is the hot path.
func (p *Pagination) buildFilterScope(config FilterConfig, value string) func(*gorm.DB) *gorm.DB {
	op := parseFilterOperation(value)

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

// isValidOperation validates filter operation
func (p *Pagination) isValidOperation(op FilterOperation, config FilterConfig) bool {
	if !slices.Contains(config.GetAllowedOperators(), op.Operator) {
		return false
	}

	switch op.Operator {
	case OperatorBetween:
		return len(op.Values) == 2
	case OperatorIn, OperatorNotIn:
		return len(op.Values) > 0
	case OperatorIsNull, OperatorIsNotNull:
		return true
	default:
		return len(op.Values) == 1
	}
}

// parseNumber parses a numeric literal as int64 when possible and falls back
// to float64. Returning a typed value (rather than the raw string) matters
// for strict-typed databases like Postgres, where `INT IN ('1','2')` errors
// with "operator does not exist: integer = text".
func parseNumber(s string) (any, bool) {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return nil, false
}

// parseID parses an integer-only identifier. IDs must never accept floats —
// `id=1.5` is nonsensical against an integer column, and above 2^53 a float
// fallback silently collides neighbouring IDs onto the same rounded value,
// potentially matching the wrong row.
func parseID(s string) (any, bool) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, false
	}
	return i, true
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
		vals := make([]any, 0, len(op.Values))
		for _, raw := range op.Values {
			if v, ok := parse(raw); ok {
				vals = append(vals, v)
			}
		}
		// All values unparseable: the user asked for "rows matching this
		// set" and the set is empty, so the answer is zero rows. Dropping
		// the filter here would silently leak every row (matching
		// buildEnumScope's fail-closed behaviour).
		if len(vals) == 0 {
			return func(db *gorm.DB) *gorm.DB { return db.Where("1 = 0") }
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" IN ?", vals) }
	case OperatorNotIn:
		vals := make([]any, 0, len(op.Values))
		for _, raw := range op.Values {
			if v, ok := parse(raw); ok {
				vals = append(vals, v)
			}
		}
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
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
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
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
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
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
	}
	return nil
}

// buildDateScope builds date filter scope.
//
// Day ranges use a half-open interval [start, nextDay) rather than a closed
// BETWEEN ending at 23:59:59.999999999. Closed ranges with sub-microsecond
// precision are unsafe against Postgres `timestamptz` (μs-only) and MySQL
// `DATETIME` (μs at best): the database truncates or rounds the literal
// before comparison, so rows stored in the last microsecond of a day may be
// silently excluded or pulled in from the next day. Half-open intervals are
// precision-independent.
func (p *Pagination) buildDateScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		t, err := time.ParseInLocation("2006-01-02", op.Values[0], p.options.Timezone)
		if err != nil {
			return nil
		}
		nextDay := t.AddDate(0, 0, 1) // calendar-correct across DST transitions
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", t, nextDay)
		}

	case OperatorBetween:
		start, err := time.ParseInLocation("2006-01-02", op.Values[0], p.options.Timezone)
		if err != nil {
			return nil
		}
		end, err := time.ParseInLocation("2006-01-02", op.Values[1], p.options.Timezone)
		if err != nil {
			return nil
		}
		nextDay := end.AddDate(0, 0, 1)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", start, nextDay)
		}

	case OperatorGte:
		t, err := time.ParseInLocation("2006-01-02", op.Values[0], p.options.Timezone)
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ?", t)
		}

	case OperatorLte:
		t, err := time.ParseInLocation("2006-01-02", op.Values[0], p.options.Timezone)
		if err != nil {
			return nil
		}
		nextDay := t.AddDate(0, 0, 1) // calendar-correct across DST transitions
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" < ?", nextDay)
		}
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
	}
	return nil
}

// buildDateTimeScope builds datetime filter scope.
//
// The input grammar "YYYY-MM-DD HH:MM:SS" is 1-second resolution, but the
// underlying column can store fractional seconds (Postgres timestamptz keeps
// microseconds, MySQL DATETIME(6) keeps microseconds). A closed range like
// "BETWEEN '…23:59:59' AND '…23:59:59'" silently drops any row whose
// fractional component is non-zero — the user thinks they asked "all rows in
// that second" but only got rows stored at exactly *.000000. The same
// precision trap that buildDateScope avoids for day boundaries applies at
// every second boundary here, so the same fix applies: half-open
// [t, t + 1s). This works identically on Postgres, MySQL, and SQLite.
func (p *Pagination) buildDateTimeScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	parseDateTime := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02 15:04:05", s, p.options.Timezone)
	}

	switch op.Operator {
	case OperatorEquals:
		t, err := parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		nextSec := t.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", t, nextSec)
		}

	case OperatorBetween:
		start, err := parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		end, err := parseDateTime(op.Values[1])
		if err != nil {
			return nil
		}
		endExclusive := end.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", start, endExclusive)
		}

	case OperatorGte:
		t, err := parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" >= ?", t) }

	case OperatorLte:
		t, err := parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		endExclusive := t.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" < ?", endExclusive)
		}
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
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
	case OperatorIsNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NULL") }
	case OperatorIsNotNull:
		return func(db *gorm.DB) *gorm.DB { return db.Where(field + " IS NOT NULL") }
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

// Helper functions

func parseLimit(conditions map[string][]string, defaultLimit, maxLimit int) int {
	if limitStr, exists := conditions[QueryKeyLimit]; exists && len(limitStr) > 0 {
		if limit, err := strconv.Atoi(limitStr[0]); err == nil && limit > 0 {
			if limit > maxLimit {
				return maxLimit
			}
			return limit
		}
	}
	return defaultLimit
}

func parseOffset(conditions map[string][]string) int {
	if offsetStr, exists := conditions[QueryKeyOffset]; exists && len(offsetStr) > 0 {
		if offset, err := strconv.Atoi(offsetStr[0]); err == nil && offset >= 0 {
			return offset
		}
	}
	return 0
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
			switch strings.ToLower(tokens[1]) {
			case "asc":
				direction = "asc"
			case "desc":
				direction = "desc"
			default:
				return defaultOrder
			}
		}

		tableName := ""
		fieldName := fieldRaw
		if idx := strings.IndexByte(fieldRaw, '.'); idx >= 0 {
			tableName = fieldRaw[:idx]
			fieldName = fieldRaw[idx+1:]
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
