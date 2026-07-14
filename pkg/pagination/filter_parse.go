package pagination

import (
	"strconv"
	"strings"
)

// FilterOperation represents a parsed filter
type FilterOperation struct {
	Operator FilterOperator
	Values   []string

	// overflow marks a value list that exceeded maxFilterValues. The operator
	// is preserved so buildFilterScope can fail closed (WHERE 1 = 0) when the
	// operator is otherwise valid, rather than dropping the filter and leaking
	// every row. Kept unexported — it's an internal parse signal, not API.
	overflow bool
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
			// Only list operators take comma-separated values. For scalar
			// operators (eq/like/gt/...) a comma is part of the value —
			// splitting "eq:Smith, John" would fail the single-value check
			// and silently drop the filter, widening the result to every row.
			switch op {
			case OperatorIn, OperatorNotIn, OperatorBetween:
				// Cap the value list to bound parse/allocation cost. A request
				// over the limit is flagged overflow so buildFilterScope can
				// fail closed instead of dropping the filter.
				values := strings.Split(rest, ",")
				if len(values) > maxFilterValues {
					return FilterOperation{Operator: op, overflow: true}
				}
				return FilterOperation{Operator: op, Values: values}
			default:
				return FilterOperation{Operator: op, Values: []string{rest}}
			}
		}
	}

	return FilterOperation{
		Operator: OperatorEquals,
		Values:   []string{value},
	}
}

// isValidOperation validates filter operation
func (p *Pagination) isValidOperation(op FilterOperation, config FilterConfig) bool {
	if !config.allowsOperator(op.Operator) {
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

// parseEach applies parse to each raw value, keeping only the ones that parse.
// Shared by the IN / NOT IN branches of buildOrderedScope, which differ only in
// what they do when nothing survives.
func parseEach(values []string, parse func(string) (any, bool)) []any {
	out := make([]any, 0, len(values))
	for _, raw := range values {
		if v, ok := parse(raw); ok {
			out = append(out, v)
		}
	}
	return out
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
