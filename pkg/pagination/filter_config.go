package pagination

import (
	"slices"
	"strings"

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

// allowsOperator reports whether op is permitted for this filter. It reads the
// operator list directly instead of cloning it (as GetAllowedOperators must
// for external callers) — this is the per-request hot path and the lookup is
// read-only.
func (fc FilterConfig) allowsOperator(op FilterOperator) bool {
	if len(fc.Operators) > 0 {
		return slices.Contains(fc.Operators, op)
	}
	return slices.Contains(operatorsByType[fc.Type], op)
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
		// Fields that already carry a table qualifier (e.g. a subquery alias)
		// are left as-is; only bare column names get the table prefix.
		if !strings.Contains(field, ".") {
			field = fc.TableName + "." + field
		}
		prefixed[i] = field
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
