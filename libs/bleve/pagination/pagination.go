// Package pagination adapts HTTP query parameters into Bleve search requests.
package pagination

import (
	"strconv"
	"strings"

	"github.com/PhantomX7/athleton/libs/bleve"
)

// FilterType represents the data type for filtering
type FilterType uint8

const (
	// FilterTypeString treats the filter input as a string value.
	FilterTypeString FilterType = iota
	// FilterTypeNumber treats the filter input as a numeric value.
	FilterTypeNumber
	// FilterTypeBool treats the filter input as a boolean value.
	FilterTypeBool
	// FilterTypeRange treats the filter input as a numeric range expression.
	FilterTypeRange
)

// FilterConfig defines filterable field configuration
type FilterConfig struct {
	Field string
	Type  FilterType
}

// SortConfig defines sortable field configuration
type SortConfig struct {
	Field   string
	Allowed bool
}

// SearchFieldDef defines a searchable field configuration
type SearchFieldDef struct {
	Field     string  // Field name in the index
	Boost     float64 // Boost value for ranking
	Fuzziness int     // 0 = no fuzziness, 1-2 = fuzzy matching
	Phrase    bool    // true = phrase match, false = term match
}

// SearchDefinition holds all search configurations
type SearchDefinition struct {
	Filters       map[string]FilterConfig
	Sorts         map[string]SortConfig
	Facets        []bleve.FacetRequest
	Highlight     []string
	SearchFields  []SearchFieldDef
	DynamicPrefix string
}

// NewSearchDefinition creates a new search definition builder
func NewSearchDefinition() *SearchDefinition {
	return &SearchDefinition{
		Filters:      make(map[string]FilterConfig),
		Sorts:        make(map[string]SortConfig),
		SearchFields: make([]SearchFieldDef, 0),
	}
}

// AddFilter registers a filterable field definition.
func (sd *SearchDefinition) AddFilter(name string, cfg FilterConfig) *SearchDefinition {
	sd.Filters[name] = cfg
	return sd
}

// AddSort registers a sortable field definition.
func (sd *SearchDefinition) AddSort(name string, cfg SortConfig) *SearchDefinition {
	sd.Sorts[name] = cfg
	return sd
}

// AddFacet registers a facet request.
func (sd *SearchDefinition) AddFacet(field string, size ...int) *SearchDefinition {
	facetSize := 0
	if len(size) > 0 {
		facetSize = size[0]
	}
	sd.Facets = append(sd.Facets, bleve.FacetRequest{Field: field, Size: facetSize})
	return sd
}

// SetHighlight configures which fields Bleve should highlight.
func (sd *SearchDefinition) SetHighlight(fields ...string) *SearchDefinition {
	sd.Highlight = fields
	return sd
}

// SetDynamicFilterPrefix configures the prefix for dynamic field filters.
func (sd *SearchDefinition) SetDynamicFilterPrefix(prefix string) *SearchDefinition {
	sd.DynamicPrefix = prefix
	return sd
}

// AddSearchField adds a searchable field with full configuration
func (sd *SearchDefinition) AddSearchField(field string, boost float64, fuzziness int, phrase bool) *SearchDefinition {
	sd.SearchFields = append(sd.SearchFields, SearchFieldDef{
		Field:     field,
		Boost:     boost,
		Fuzziness: fuzziness,
		Phrase:    phrase,
	})
	return sd
}

// AddExactSearchField adds a phrase match field (highest priority for exact matches)
func (sd *SearchDefinition) AddExactSearchField(field string, boost float64) *SearchDefinition {
	return sd.AddSearchField(field, boost, 0, true)
}

// AddStandardSearchField adds a standard match field (all terms must match, no fuzziness)
func (sd *SearchDefinition) AddStandardSearchField(field string, boost float64) *SearchDefinition {
	return sd.AddSearchField(field, boost, 0, false)
}

// AddFuzzySearchField adds a fuzzy match field (typo tolerance)
func (sd *SearchDefinition) AddFuzzySearchField(field string, boost float64, fuzziness int) *SearchDefinition {
	return sd.AddSearchField(field, boost, fuzziness, false)
}

// SearchOptions configures search behavior
type SearchOptions struct {
	DefaultLimit int
	MaxLimit     int
	DefaultSort  string
}

// DefaultSearchOptions returns sensible defaults
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultSort:  "-_score",
	}
}

// SearchPagination manages Bleve search pagination
type SearchPagination struct {
	Query        string
	Limit        int
	Offset       int
	Sort         []string
	Filters      map[string]bleve.Filter
	Facets       []bleve.FacetRequest
	Highlight    []string
	SearchFields []bleve.SearchFieldConfig
}

// NewSearchPagination creates a search pagination instance
func NewSearchPagination(params map[string][]string, def *SearchDefinition, opts SearchOptions) *SearchPagination {
	sp := &SearchPagination{
		Query:   getParam(params, "q"),
		Limit:   clamp(getIntParam(params, "limit", opts.DefaultLimit), 1, opts.MaxLimit),
		Filters: make(map[string]bleve.Filter),
		Facets:  parseFacets(params, def),
	}

	// Calculate offset from page or explicit offset
	page := getIntParam(params, "page", 1)
	sp.Offset = getIntParam(params, "offset", (page-1)*sp.Limit)

	// Parse sort
	sp.Sort = parseSort(params, def.Sorts, opts.DefaultSort)

	// Parse filters
	sp.parseFilters(params, def)

	// Set highlight if query present
	if sp.Query != "" {
		sp.Highlight = def.Highlight
	}

	// Convert search fields from definition
	sp.SearchFields = convertSearchFields(def.SearchFields)

	return sp
}

// convertSearchFields converts SearchFieldDef to bleve.SearchFieldConfig
func convertSearchFields(defs []SearchFieldDef) []bleve.SearchFieldConfig {
	if len(defs) == 0 {
		return nil // Will use defaults in bleve client
	}

	fields := make([]bleve.SearchFieldConfig, len(defs))
	for i, d := range defs {
		fields[i] = bleve.SearchFieldConfig{
			Field:     d.Field,
			Boost:     d.Boost,
			Fuzziness: d.Fuzziness,
			Phrase:    d.Phrase,
		}
	}
	return fields
}

func (sp *SearchPagination) parseFilters(params map[string][]string, def *SearchDefinition) {
	// Static filters
	for name, cfg := range def.Filters {
		if val := getParam(params, name); val != "" {
			if f := buildFilter(cfg.Type, val); f != nil {
				sp.Filters[cfg.Field] = *f
			}
		}
	}

	// Dynamic filters (e.g., specs.*)
	if def.DynamicPrefix != "" {
		for key, values := range params {
			if strings.HasPrefix(key, def.DynamicPrefix) && len(values) > 0 {
				if _, exists := def.Filters[key]; exists {
					continue
				}
				if f := buildFilter(FilterTypeString, values[0]); f != nil {
					sp.Filters[key] = *f
				}
			}
		}
	}
}

func buildFilter(filterType FilterType, value string) *bleve.Filter {
	op, vals := parseOperator(value)

	switch filterType {
	case FilterTypeBool:
		return &bleve.Filter{
			Type:  bleve.FilterBool,
			Value: strings.EqualFold(vals[0], "true"),
		}

	case FilterTypeNumber, FilterTypeRange:
		return buildRangeFilter(op, vals)

	case FilterTypeString:
		return buildStringFilter(op, vals)

	default:
		return nil
	}
}

func buildRangeFilter(op string, vals []string) *bleve.Filter {
	var minValue, maxValue *float64

	parseFloat := func(s string) *float64 {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return &v
		}
		return nil
	}

	switch op {
	case "eq":
		if v := parseFloat(vals[0]); v != nil {
			minValue, maxValue = v, v
		}
	case "between":
		if len(vals) >= 2 {
			minValue, maxValue = parseFloat(vals[0]), parseFloat(vals[1])
		}
	case "gt":
		if v := parseFloat(vals[0]); v != nil {
			adjusted := *v + 0.0001
			minValue = &adjusted
		}
	case "gte":
		minValue = parseFloat(vals[0])
	case "lt":
		if v := parseFloat(vals[0]); v != nil {
			adjusted := *v - 0.0001
			maxValue = &adjusted
		}
	case "lte":
		maxValue = parseFloat(vals[0])
	}

	if minValue == nil && maxValue == nil {
		return nil
	}

	return &bleve.Filter{
		Type:  bleve.FilterRange,
		Value: bleve.RangeValue{Min: minValue, Max: maxValue},
	}
}

func buildStringFilter(op string, vals []string) *bleve.Filter {
	switch op {
	case "eq":
		return &bleve.Filter{Type: bleve.FilterExact, Value: vals[0]}
	case "like":
		return &bleve.Filter{Type: bleve.FilterMatch, Value: vals[0]}
	case "in":
		if len(vals) == 1 {
			return &bleve.Filter{Type: bleve.FilterExact, Value: vals[0]}
		}
		return &bleve.Filter{Type: bleve.FilterIn, Value: vals}
	default:
		return &bleve.Filter{Type: bleve.FilterExact, Value: vals[0]}
	}
}

// parseOperator parses "operator:value1,value2" format
func parseOperator(s string) (op string, vals []string) {
	if idx := strings.IndexByte(s, ':'); idx > 0 {
		return s[:idx], strings.Split(s[idx+1:], ",")
	}
	return "eq", []string{s}
}

func parseFacets(params map[string][]string, def *SearchDefinition) []bleve.FacetRequest {
	facetStr := getParam(params, "facet")
	if facetStr == "" {
		return def.Facets
	}

	requested := strings.Split(facetStr, ",")
	result := make([]bleve.FacetRequest, 0, len(requested))

	// Index existing facets for quick lookup
	facetMap := make(map[string]bleve.FacetRequest)
	for _, f := range def.Facets {
		facetMap[f.Field] = f
	}

	for _, f := range requested {
		f = strings.TrimSpace(f)
		if cfg, ok := facetMap[f]; ok {
			result = append(result, cfg)
		} else if def.DynamicPrefix != "" && strings.HasPrefix(f, def.DynamicPrefix) {
			result = append(result, bleve.FacetRequest{Field: f})
		}
	}

	if len(result) == 0 {
		return def.Facets
	}
	return result
}

func parseSort(params map[string][]string, sorts map[string]SortConfig, defaultSort string) []string {
	sortStr := getParam(params, "sort")
	if sortStr == "" {
		return []string{defaultSort}
	}

	var result []string
	for _, s := range strings.Split(sortStr, ",") {
		s = strings.TrimSpace(s)
		field, desc := parseField(s)

		if cfg, ok := sorts[field]; ok && cfg.Allowed {
			if desc {
				result = append(result, "-"+cfg.Field)
			} else {
				result = append(result, cfg.Field)
			}
		}
	}

	if len(result) == 0 {
		return []string{defaultSort}
	}
	return result
}

// parseField handles both "-name" and "name desc" formats
func parseField(s string) (field string, desc bool) {
	if strings.HasPrefix(s, "-") {
		return s[1:], true
	}

	parts := strings.Fields(s)
	if len(parts) == 2 {
		dir := strings.ToLower(parts[1])
		return parts[0], dir == "desc" || dir == "descending"
	}

	return s, false
}

// ToSearchRequest converts to Bleve search request
func (sp *SearchPagination) ToSearchRequest() *bleve.SearchRequest {
	return &bleve.SearchRequest{
		Query:        sp.Query,
		Filters:      sp.Filters,
		Sort:         sp.Sort,
		Size:         sp.Limit,
		From:         sp.Offset,
		Highlight:    sp.Highlight,
		Facets:       sp.Facets,
		SearchFields: sp.SearchFields,
	}
}

// Helper functions

func getParam(params map[string][]string, key string) string {
	if vals, ok := params[key]; ok && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}

func getIntParam(params map[string][]string, key string, defaultVal int) int {
	if s := getParam(params, key); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return defaultVal
}

func clamp(val, minValue, maxValue int) int {
	if val < minValue {
		return minValue
	}
	if val > maxValue {
		return maxValue
	}
	return val
}
