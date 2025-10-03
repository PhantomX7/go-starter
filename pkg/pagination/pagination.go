package pagination

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Constants for query parameter keys.
const (
	// QueryKeyLimit is the key for the page size.
	QueryKeyLimit = "limit"
	// QueryKeyOffset is the key for the page offset.
	QueryKeyOffset = "offset"
	// QueryKeySort is the key for sorting.
	QueryKeySort = "sort"
)

// FilterType represents the type of filter used for validation and scope building.
type FilterType string

// Defines the available filter types.
const (
	FilterTypeID       FilterType = "ID"
	FilterTypeNumber   FilterType = "NUMBER"
	FilterTypeString   FilterType = "STRING"
	FilterTypeBool     FilterType = "BOOL"
	FilterTypeDate     FilterType = "DATE"
	FilterTypeDateTime FilterType = "DATETIME"
	FilterTypeEnum     FilterType = "ENUM"
)

// FilterOperator represents the operation to perform for a filter.
type FilterOperator string

// Defines the available filter operators.
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
)

// operatorsByType maps filter types to their allowed operators.
var operatorsByType = map[FilterType][]FilterOperator{
	FilterTypeID:       {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorBetween, OperatorGt, OperatorGte, OperatorLt, OperatorLte},
	FilterTypeNumber:   {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorBetween, OperatorGt, OperatorGte, OperatorLt, OperatorLte},
	FilterTypeString:   {OperatorEquals, OperatorNotEquals, OperatorIn, OperatorNotIn, OperatorLike},
	FilterTypeBool:     {OperatorEquals},
	FilterTypeDate:     {OperatorEquals, OperatorBetween, OperatorGte, OperatorLte},
	FilterTypeDateTime: {OperatorEquals, OperatorBetween, OperatorGte, OperatorLte},
	FilterTypeEnum:     {OperatorEquals, OperatorIn},
}

// FilterConfig defines the configuration for a single filterable field.
// It specifies how a field should be filtered, including its type, database field name,
// allowed operators, and more.
type FilterConfig struct {
	// Field is the primary database column name for the filter.
	Field string
	// SearchFields is a list of database columns to apply an OR condition to.
	// If empty, `Field` is used.
	SearchFields []string
	// Type is the data type of the filter (e.g., STRING, NUMBER).
	Type FilterType
	// TableName is an optional table name or alias to prefix the field with.
	TableName string
	// Operators is a list of allowed operators for this filter. If empty, default operators for the type are used.
	Operators []FilterOperator
	// EnumValues is a list of valid values for ENUM type filters.
	EnumValues []string
}

// GetAllowedOperators returns the list of allowed operators for this filter configuration.
// If custom operators are defined in the config, they are returned; otherwise, it falls back to the default operators for the filter's type.
func (fc FilterConfig) GetAllowedOperators() []FilterOperator {
	if len(fc.Operators) > 0 {
		return fc.Operators
	}
	return operatorsByType[fc.Type]
}

// GetFields returns all fields to be used in the query, prefixed with the table name if provided.
// It prioritizes `SearchFields` and falls back to `Field`.
func (fc FilterConfig) GetFields() []string {
	fields := fc.SearchFields
	if len(fields) == 0 {
		fields = []string{fc.Field}
	}

	// Apply table prefix if needed
	if fc.TableName != "" {
		for i, field := range fields {
			if !strings.Contains(field, ".") {
				fields[i] = fmt.Sprintf("%s.%s", fc.TableName, field)
			}
		}
	}

	return fields
}

// SortConfig defines the configuration for a sortable field.
type SortConfig struct {
	// Field is the database column name for sorting.
	Field string
	// TableName is an optional table name or alias to prefix the field with.
	TableName string
	// Allowed indicates whether sorting by this field is permitted.
	Allowed bool
}

// FilterDefinition holds all filter and sort configurations for a query.
type FilterDefinition struct {
	filters map[string]FilterConfig
	sorts   map[string]SortConfig
}

// NewFilterDefinition creates and returns a new FilterDefinition instance.
func NewFilterDefinition() *FilterDefinition {
	return &FilterDefinition{
		filters: make(map[string]FilterConfig),
		sorts:   make(map[string]SortConfig),
	}
}

// AddFilter adds a filter configuration to the definition.
func (fd *FilterDefinition) AddFilter(field string, config FilterConfig) *FilterDefinition {
	fd.filters[field] = config
	return fd
}

// AddSort adds a sort configuration to the definition.
func (fd *FilterDefinition) AddSort(field string, config SortConfig) *FilterDefinition {
	fd.sorts[field] = config
	return fd
}

// PaginationOptions holds configuration for pagination behavior, such as limits and default ordering.
type PaginationOptions struct {
	DefaultLimit int
	MaxLimit     int
	DefaultOrder string
	Timezone     *time.Location
}

// Pagination holds the parsed pagination state and provides methods to apply it to a GORM query.
type Pagination struct {
	Limit      int
	Offset     int
	Order      string
	conditions map[string][]string
	filterDef  *FilterDefinition
	options    PaginationOptions
	scopes     []func(*gorm.DB) *gorm.DB
}

// NewPagination creates a new Pagination instance with the given parameters.
// It initializes pagination settings, parses limit, offset, and order from the
// query conditions, and sets up default values.
func NewPagination(conditions map[string][]string, filterDef *FilterDefinition, options PaginationOptions) *Pagination {
	// Set defaults
	if options.DefaultLimit == 0 {
		options.DefaultLimit = 20
	}
	if options.MaxLimit == 0 {
		options.MaxLimit = 100
	}
	if options.DefaultOrder == "" {
		options.DefaultOrder = "id desc"
	}
	if options.Timezone == nil {
		options.Timezone, _ = time.LoadLocation("Asia/Jakarta")
	}

	return &Pagination{
		conditions: conditions,
		filterDef:  filterDef,
		options:    options,
		Limit:      parseLimit(conditions, options.DefaultLimit, options.MaxLimit),
		Offset:     parseOffset(conditions),
		Order:      parseOrder(conditions, options.DefaultOrder, filterDef),
		scopes:     make([]func(*gorm.DB) *gorm.DB, 0),
	}
}

// AddCustomScope adds one or more custom GORM scopes to the pagination instance.
// These scopes are applied along with the filter scopes.
func (p *Pagination) AddCustomScope(scopes ...func(*gorm.DB) *gorm.DB) {
	p.scopes = append(p.scopes, scopes...)
}

// Apply applies all generated filter and meta (limit, offset, order) scopes to the GORM query.
func (p *Pagination) Apply(db *gorm.DB) *gorm.DB {
	filterScopes, metaScopes := p.buildScopes()

	// Apply filter scopes first
	for _, scope := range filterScopes {
		db = scope(db)
	}

	// Apply meta scopes (limit, offset, order)
	for _, scope := range metaScopes {
		db = scope(db)
	}

	return db
}

// ApplyWithoutMeta applies only the filter scopes to the GORM query.
// This is useful for operations like counting total records that match the filters, without applying limit and offset.
func (p *Pagination) ApplyWithoutMeta(db *gorm.DB) *gorm.DB {
	filterScopes, _ := p.buildScopes()

	for _, scope := range filterScopes {
		db = scope(db)
	}

	return db
}

// buildScopes constructs and returns the filter and meta scopes separately based on the pagination state.
func (p *Pagination) buildScopes() (filterScopes []func(*gorm.DB) *gorm.DB, metaScopes []func(*gorm.DB) *gorm.DB) {
	// Build filter scopes from query conditions
	for field, values := range p.conditions {
		if config, exists := p.filterDef.filters[field]; exists {
			if scope := p.buildFilterScope(config, values); scope != nil {
				filterScopes = append(filterScopes, scope)
			}
		}
	}

	// Add custom scopes
	filterScopes = append(filterScopes, p.scopes...)

	// Build meta scopes (limit, offset, order)
	metaScopes = []func(*gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB { return db.Limit(p.Limit) },
		func(db *gorm.DB) *gorm.DB { return db.Offset(p.Offset) },
		func(db *gorm.DB) *gorm.DB { return db.Order(p.Order) },
	}

	return filterScopes, metaScopes
}

// FilterOperation represents a parsed filter operation, including the operator and its values.
type FilterOperation struct {
	Operator FilterOperator
	Values   []string
}

// parseFilterOperation parses a raw filter string (e.g., "eq:value" or "in:v1,v2") into a FilterOperation struct.
// If the string contains no operator, it defaults to "eq".
func parseFilterOperation(value string) FilterOperation {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return FilterOperation{
			Operator: OperatorEquals,
			Values:   []string{value},
		}
	}

	return FilterOperation{
		Operator: FilterOperator(parts[0]),
		Values:   strings.Split(parts[1], ","),
	}
}

// buildFilterScope constructs a GORM scope for a given filter configuration and its values.
// It validates the operation and dispatches to the appropriate type-specific scope builder.
func (p *Pagination) buildFilterScope(config FilterConfig, values []string) func(*gorm.DB) *gorm.DB {
	if len(values) == 0 {
		return nil
	}

	op := parseFilterOperation(values[0])

	// Validate the parsed operation against the filter's configuration.
	if !p.isValidOperation(op, config) {
		return nil
	}

	fields := config.GetFields()
	if len(fields) == 0 {
		return nil
	}

	// Build scope based on filter type
	switch config.Type {
	case FilterTypeID, FilterTypeNumber:
		return p.buildNumericScope(fields[0], op)
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

// isValidOperation checks if a filter operation is valid for a given configuration.
// It verifies that the operator is allowed and that the number of values is correct for the operator.
func (p *Pagination) isValidOperation(op FilterOperation, config FilterConfig) bool {
	// Check if the operator is in the list of allowed operators for the filter type.
	allowedOps := config.GetAllowedOperators()
	operatorAllowed := slices.Contains(allowedOps, op.Operator)

	if !operatorAllowed {
		return false
	}

	// Validate the number of values required by the operator.
	switch op.Operator {
	case OperatorBetween:
		return len(op.Values) == 2
	case OperatorIn, OperatorNotIn:
		return len(op.Values) > 0
	default:
		return len(op.Values) == 1
	}
}

// Scope builders

// buildNumericScope builds a GORM scope for numeric and ID filters.
func (p *Pagination) buildNumericScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	builders := map[FilterOperator]func(*gorm.DB) *gorm.DB{
		OperatorEquals:    func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s = ?", field), op.Values[0]) },
		OperatorNotEquals: func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s != ?", field), op.Values[0]) },
		OperatorIn:        func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s IN ?", field), op.Values) },
		OperatorNotIn:     func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s NOT IN ?", field), op.Values) },
		OperatorBetween: func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), op.Values[0], op.Values[1])
		},
		OperatorGt:  func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s > ?", field), op.Values[0]) },
		OperatorGte: func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s >= ?", field), op.Values[0]) },
		OperatorLt:  func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s < ?", field), op.Values[0]) },
		OperatorLte: func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s <= ?", field), op.Values[0]) },
	}

	return builders[op.Operator]
}

// buildStringScope builds a GORM scope for string filters.
// It handles both single-field and multi-field (OR condition) filtering.
func (p *Pagination) buildStringScope(fields []string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	if len(fields) == 1 {
		return p.buildSingleStringScope(fields[0], op)
	}
	return p.buildMultiStringScope(fields, op)
}

// buildSingleStringScope builds a GORM scope for a string filter on a single field.
func (p *Pagination) buildSingleStringScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	builders := map[FilterOperator]func(*gorm.DB) *gorm.DB{
		OperatorEquals:    func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s = ?", field), op.Values[0]) },
		OperatorNotEquals: func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s != ?", field), op.Values[0]) },
		OperatorLike: func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", field), fmt.Sprintf("%%%s%%", op.Values[0]))
		},
		OperatorIn:    func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s IN ?", field), op.Values) },
		OperatorNotIn: func(db *gorm.DB) *gorm.DB { return db.Where(fmt.Sprintf("%s NOT IN ?", field), op.Values) },
	}

	if builder, exists := builders[op.Operator]; exists {
		return builder
	}
	return func(db *gorm.DB) *gorm.DB { return db }
}

// buildMultiStringScope builds a GORM scope for a string filter across multiple fields with an OR condition.
func (p *Pagination) buildMultiStringScope(fields []string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		var conditions []string
		var args []interface{}

		for _, field := range fields {
			switch op.Operator {
			case OperatorEquals:
				conditions = append(conditions, fmt.Sprintf("%s = ?", field))
				args = append(args, op.Values[0])
			case OperatorNotEquals:
				conditions = append(conditions, fmt.Sprintf("%s != ?", field))
				args = append(args, op.Values[0])
			case OperatorLike:
				conditions = append(conditions, fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", field))
				args = append(args, fmt.Sprintf("%%%s%%", op.Values[0]))
			case OperatorIn:
				// For IN, we need to repeat the argument for each field.
				conditions = append(conditions, fmt.Sprintf("%s IN ?", field))
				args = append(args, op.Values)
			case OperatorNotIn:
				conditions = append(conditions, fmt.Sprintf("%s NOT IN ?", field))
				args = append(args, op.Values)
			}
		}

		if len(conditions) == 0 {
			return db
		}

		return db.Where(fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), args...)
	}
}

// buildBoolScope builds a GORM scope for boolean filters.
func (p *Pagination) buildBoolScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	if op.Operator != OperatorEquals {
		return nil
	}

	value := strings.ToLower(op.Values[0]) == "true"
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s = ?", field), value)
	}
}

// parseDate is a helper to parse a date string into a time.Time object.
func (p *Pagination) parseDate(dateStr string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", dateStr, p.options.Timezone)
}

// parseDateTime is a helper to parse a datetime string into a time.Time object.
func (p *Pagination) parseDateTime(dateTimeStr string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", dateTimeStr, p.options.Timezone)
}

// buildDateScope builds a GORM scope for date filters.
func (p *Pagination) buildDateScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil // Silently ignore invalid date formats
		}
		endOfDay := t.Add(24*time.Hour - time.Nanosecond)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), t, endOfDay)
		}

	case OperatorBetween:
		start, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		end, err := p.parseDate(op.Values[1])
		if err != nil {
			return nil
		}
		endOfDay := end.Add(24*time.Hour - time.Nanosecond)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), start, endOfDay)
		}

	case OperatorGte:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s >= ?", field), t)
		}

	case OperatorLte:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		endOfDay := t.Add(24*time.Hour - time.Nanosecond)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s <= ?", field), endOfDay)
		}
	}

	return nil
}

// buildDateTimeScope builds a GORM scope for datetime filters.
func (p *Pagination) buildDateTimeScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil // Silently ignore invalid datetime formats
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s = ?", field), t)
		}

	case OperatorBetween:
		start, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		end, err := p.parseDateTime(op.Values[1])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), start, end)
		}

	case OperatorGte:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s >= ?", field), t)
		}

	case OperatorLte:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s <= ?", field), t)
		}
	}

	return nil
}

// buildEnumScope builds a GORM scope for enum filters, ensuring values are within the allowed list.
func (p *Pagination) buildEnumScope(field string, op FilterOperation, allowedValues []string) func(*gorm.DB) *gorm.DB {
	// Validate that all provided values are in the list of allowed enum values.
	for _, val := range op.Values {
		if !slices.Contains(allowedValues, val) {
			return nil // Silently ignore if any value is not allowed.
		}
	}

	switch op.Operator {
	case OperatorEquals:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s = ?", field), op.Values[0])
		}
	case OperatorIn:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(fmt.Sprintf("%s IN ?", field), op.Values)
		}
	}

	return nil
}

// Helper functions

// parseLimit parses the limit from query conditions, applying default and max values.
func parseLimit(conditions map[string][]string, defaultLimit, maxLimit int) int {
	if limitStr, exists := conditions[QueryKeyLimit]; exists && len(limitStr) > 0 {
		if limit, err := strconv.Atoi(limitStr[0]); err == nil && limit > 0 {
			if limit <= maxLimit {
				return limit
			}
			return maxLimit
		}
	}
	return defaultLimit
}

// parseOffset parses the offset from query conditions.
func parseOffset(conditions map[string][]string) int {
	if offsetStr, exists := conditions[QueryKeyOffset]; exists && len(offsetStr) > 0 {
		if offset, err := strconv.Atoi(offsetStr[0]); err == nil && offset >= 0 {
			return offset
		}
	}
	return 0
}

// parseOrder parses and validates the sort order from query conditions.
func parseOrder(conditions map[string][]string, defaultOrder string, filterDef *FilterDefinition) string {
	if orderStr, exists := conditions[QueryKeySort]; exists && len(orderStr) > 0 {
		order := orderStr[0]
		if validateOrder(order, filterDef) {
			return order
		}
	}
	return defaultOrder
}

// validateOrder checks if the sort order string is valid against the sort configurations.
func validateOrder(order string, filterDef *FilterDefinition) bool {
	if order == "" {
		return true // No order is valid.
	}

	// An order string can contain multiple fields, e.g., "name asc, age desc".
	for _, part := range strings.Split(order, ",") {
		fieldParts := strings.Fields(strings.TrimSpace(part))
		if len(fieldParts) == 0 {
			continue // Skip empty parts.
		}

		field := fieldParts[0]
		// The field might be prefixed with a table name.
		fieldNameOnly := field
		if strings.Contains(field, ".") {
			fieldNameOnly = strings.SplitN(field, ".", 2)[1]
		}

		sortConfig, exists := filterDef.sorts[fieldNameOnly]
		if !exists || !sortConfig.Allowed {
			return false // Field is not defined for sorting or not allowed.
		}

		// Check if the table name matches, if provided.
		if sortConfig.TableName != "" && strings.Contains(field, ".") {
			tableName := strings.SplitN(field, ".", 2)[0]
			if tableName != sortConfig.TableName {
				return false
			}
		}
	}
	return true
}
