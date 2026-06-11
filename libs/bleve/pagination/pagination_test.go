package pagination_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/libs/bleve"
	"github.com/PhantomX7/athleton/libs/bleve/pagination"
)

func testDefinition() *pagination.SearchDefinition {
	return pagination.NewSearchDefinition().
		AddFilter("category", pagination.FilterConfig{Field: "category", Type: pagination.FilterTypeString}).
		AddFilter("price", pagination.FilterConfig{Field: "price", Type: pagination.FilterTypeNumber}).
		AddFilter("is_active", pagination.FilterConfig{Field: "is_active", Type: pagination.FilterTypeBool}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true}).
		AddSort("price", pagination.SortConfig{Field: "price", Allowed: true}).
		AddSort("secret", pagination.SortConfig{Field: "secret", Allowed: false}).
		AddFacet("category").
		AddFacet("brand", 5).
		SetHighlight("name", "description").
		AddExactSearchField("name", bleve.BoostExactMatch).
		AddFuzzySearchField("name", bleve.BoostFuzzyMatch, 2)
}

func TestNewSearchPaginationDefaults(t *testing.T) {
	def := testDefinition()
	opts := pagination.DefaultSearchOptions()

	sp := pagination.NewSearchPagination(map[string][]string{}, def, opts)

	require.Empty(t, sp.Query)
	require.Equal(t, 20, sp.Limit)
	require.Zero(t, sp.Offset)
	require.Equal(t, []string{"-_score"}, sp.Sort)
	require.Empty(t, sp.Filters)
	// No query means no highlighting.
	require.Empty(t, sp.Highlight)
	// Definition facets pass through when no facet param is given.
	require.Equal(t, def.Facets, sp.Facets)
}

func TestNewSearchPaginationQueryEnablesHighlight(t *testing.T) {
	params := map[string][]string{"q": {"  running shoes "}}

	sp := pagination.NewSearchPagination(params, testDefinition(), pagination.DefaultSearchOptions())

	require.Equal(t, "running shoes", sp.Query)
	require.Equal(t, []string{"name", "description"}, sp.Highlight)
}

func TestNewSearchPaginationLimitAndOffset(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition()

	t.Run("limit clamped to max", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"limit": {"500"}}, def, opts)
		require.Equal(t, 100, sp.Limit)
	})

	t.Run("invalid limit falls back to default", func(t *testing.T) {
		for _, v := range []string{"0", "-5", "abc"} {
			sp := pagination.NewSearchPagination(map[string][]string{"limit": {v}}, def, opts)
			require.Equal(t, 20, sp.Limit, "limit=%q", v)
		}
	})

	t.Run("page computes offset", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"page": {"3"}, "limit": {"10"}}, def, opts)
		require.Equal(t, 10, sp.Limit)
		require.Equal(t, 20, sp.Offset)
	})

	t.Run("explicit offset wins over page", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"page": {"3"}, "offset": {"7"}}, def, opts)
		require.Equal(t, 7, sp.Offset)
	})
}

func TestNewSearchPaginationSortParsing(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition()

	cases := []struct {
		name string
		sort string
		want []string
	}{
		{"ascending", "name", []string{"name"}},
		{"dash prefix descending", "-name", []string{"-name"}},
		{"desc keyword", "name desc", []string{"-name"}},
		{"descending keyword", "price descending", []string{"-price"}},
		{"asc keyword", "name asc", []string{"name"}},
		{"multiple fields", "name,-price", []string{"name", "-price"}},
		{"disallowed falls back to default", "secret", []string{"-_score"}},
		{"unknown falls back to default", "nope", []string{"-_score"}},
		{"disallowed mixed with allowed", "secret,price", []string{"price"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sp := pagination.NewSearchPagination(map[string][]string{"sort": {tc.sort}}, def, opts)
			require.Equal(t, tc.want, sp.Sort)
		})
	}
}

func TestNewSearchPaginationStringFilters(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition()

	t.Run("plain value is exact", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"category": {"shoes"}}, def, opts)
		require.Equal(t, bleve.Filter{Type: bleve.FilterExact, Value: "shoes"}, sp.Filters["category"])
	})

	t.Run("eq operator", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"category": {"eq:shoes"}}, def, opts)
		require.Equal(t, bleve.Filter{Type: bleve.FilterExact, Value: "shoes"}, sp.Filters["category"])
	})

	t.Run("like operator", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"category": {"like:shoe"}}, def, opts)
		require.Equal(t, bleve.Filter{Type: bleve.FilterMatch, Value: "shoe"}, sp.Filters["category"])
	})

	t.Run("in operator with multiple values", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"category": {"in:shoes,apparel"}}, def, opts)
		require.Equal(t, bleve.Filter{Type: bleve.FilterIn, Value: []string{"shoes", "apparel"}}, sp.Filters["category"])
	})

	t.Run("in operator with single value collapses to exact", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"category": {"in:shoes"}}, def, opts)
		require.Equal(t, bleve.Filter{Type: bleve.FilterExact, Value: "shoes"}, sp.Filters["category"])
	})

	t.Run("unconfigured params are ignored", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"unknown": {"x"}}, def, opts)
		require.Empty(t, sp.Filters)
	})
}

func TestNewSearchPaginationNumberFilters(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition()

	rangeOf := func(t *testing.T, sp *pagination.SearchPagination) bleve.RangeValue {
		t.Helper()
		f, ok := sp.Filters["price"]
		require.True(t, ok)
		require.Equal(t, bleve.FilterRange, f.Type)
		rv, ok := f.Value.(bleve.RangeValue)
		require.True(t, ok)
		return rv
	}

	t.Run("eq sets min and max", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"price": {"50"}}, def, opts)
		rv := rangeOf(t, sp)
		require.NotNil(t, rv.Min)
		require.NotNil(t, rv.Max)
		require.InDelta(t, 50, *rv.Min, 1e-9)
		require.InDelta(t, 50, *rv.Max, 1e-9)
	})

	t.Run("gte and lte", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"price": {"gte:10"}}, def, opts)
		rv := rangeOf(t, sp)
		require.NotNil(t, rv.Min)
		require.InDelta(t, 10, *rv.Min, 1e-9)
		require.Nil(t, rv.Max)

		sp = pagination.NewSearchPagination(map[string][]string{"price": {"lte:99"}}, def, opts)
		rv = rangeOf(t, sp)
		require.Nil(t, rv.Min)
		require.NotNil(t, rv.Max)
		require.InDelta(t, 99, *rv.Max, 1e-9)
	})

	t.Run("gt and lt adjust boundary", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"price": {"gt:10"}}, def, opts)
		rv := rangeOf(t, sp)
		require.NotNil(t, rv.Min)
		require.Greater(t, *rv.Min, 10.0)

		sp = pagination.NewSearchPagination(map[string][]string{"price": {"lt:10"}}, def, opts)
		rv = rangeOf(t, sp)
		require.NotNil(t, rv.Max)
		require.Less(t, *rv.Max, 10.0)
	})

	t.Run("between", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"price": {"between:10,20"}}, def, opts)
		rv := rangeOf(t, sp)
		require.NotNil(t, rv.Min)
		require.NotNil(t, rv.Max)
		require.InDelta(t, 10, *rv.Min, 1e-9)
		require.InDelta(t, 20, *rv.Max, 1e-9)
	})

	t.Run("non-numeric value is dropped", func(t *testing.T) {
		sp := pagination.NewSearchPagination(map[string][]string{"price": {"gt:abc"}}, def, opts)
		require.NotContains(t, sp.Filters, "price")
	})
}

func TestNewSearchPaginationBoolFilter(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition()

	sp := pagination.NewSearchPagination(map[string][]string{"is_active": {"true"}}, def, opts)
	require.Equal(t, bleve.Filter{Type: bleve.FilterBool, Value: true}, sp.Filters["is_active"])

	sp = pagination.NewSearchPagination(map[string][]string{"is_active": {"TRUE"}}, def, opts)
	require.Equal(t, bleve.Filter{Type: bleve.FilterBool, Value: true}, sp.Filters["is_active"])

	sp = pagination.NewSearchPagination(map[string][]string{"is_active": {"false"}}, def, opts)
	require.Equal(t, bleve.Filter{Type: bleve.FilterBool, Value: false}, sp.Filters["is_active"])
}

func TestNewSearchPaginationDynamicFilters(t *testing.T) {
	opts := pagination.DefaultSearchOptions()
	def := testDefinition().SetDynamicFilterPrefix("specs.")

	sp := pagination.NewSearchPagination(map[string][]string{
		"specs.color": {"red"},
		"specs.size":  {"in:s,m"},
		"other":       {"ignored"},
	}, def, opts)

	require.Equal(t, bleve.Filter{Type: bleve.FilterExact, Value: "red"}, sp.Filters["specs.color"])
	require.Equal(t, bleve.Filter{Type: bleve.FilterIn, Value: []string{"s", "m"}}, sp.Filters["specs.size"])
	require.NotContains(t, sp.Filters, "other")
}

func TestNewSearchPaginationFacetParam(t *testing.T) {
	opts := pagination.DefaultSearchOptions()

	t.Run("subset of defined facets", func(t *testing.T) {
		def := testDefinition()
		sp := pagination.NewSearchPagination(map[string][]string{"facet": {"brand"}}, def, opts)
		require.Equal(t, []bleve.FacetRequest{{Field: "brand", Size: 5}}, sp.Facets)
	})

	t.Run("unknown facets fall back to definition", func(t *testing.T) {
		def := testDefinition()
		sp := pagination.NewSearchPagination(map[string][]string{"facet": {"nope"}}, def, opts)
		require.Equal(t, def.Facets, sp.Facets)
	})

	t.Run("dynamic prefix facets are allowed", func(t *testing.T) {
		def := testDefinition().SetDynamicFilterPrefix("specs.")
		sp := pagination.NewSearchPagination(map[string][]string{"facet": {"category, specs.color"}}, def, opts)
		require.Equal(t, []bleve.FacetRequest{
			{Field: "category"},
			{Field: "specs.color"},
		}, sp.Facets)
	})
}

func TestSearchDefinitionBuilders(t *testing.T) {
	def := testDefinition()

	require.Len(t, def.Filters, 3)
	require.Len(t, def.Sorts, 3)
	require.Equal(t, []bleve.FacetRequest{{Field: "category"}, {Field: "brand", Size: 5}}, def.Facets)
	require.Equal(t, []string{"name", "description"}, def.Highlight)

	require.Equal(t, []pagination.SearchFieldDef{
		{Field: "name", Boost: bleve.BoostExactMatch, Fuzziness: 0, Phrase: true},
		{Field: "name", Boost: bleve.BoostFuzzyMatch, Fuzziness: 2, Phrase: false},
	}, def.SearchFields)

	def.AddStandardSearchField("description", 2)
	require.Equal(t,
		pagination.SearchFieldDef{Field: "description", Boost: 2, Fuzziness: 0, Phrase: false},
		def.SearchFields[2],
	)
}

func TestToSearchRequestMapsAllFields(t *testing.T) {
	def := testDefinition()
	opts := pagination.DefaultSearchOptions()

	sp := pagination.NewSearchPagination(map[string][]string{
		"q":        {"running"},
		"limit":    {"5"},
		"page":     {"2"},
		"sort":     {"-price"},
		"category": {"shoes"},
	}, def, opts)

	req := sp.ToSearchRequest()

	require.Equal(t, "running", req.Query)
	require.Equal(t, 5, req.Size)
	require.Equal(t, 5, req.From)
	require.Equal(t, []string{"-price"}, req.Sort)
	require.Equal(t, map[string]bleve.Filter{
		"category": {Type: bleve.FilterExact, Value: "shoes"},
	}, req.Filters)
	require.Equal(t, []string{"name", "description"}, req.Highlight)
	require.Equal(t, def.Facets, req.Facets)
	require.Equal(t, []bleve.SearchFieldConfig{
		{Field: "name", Boost: bleve.BoostExactMatch, Fuzziness: 0, Phrase: true},
		{Field: "name", Boost: bleve.BoostFuzzyMatch, Fuzziness: 2, Phrase: false},
	}, req.SearchFields)
}

func TestConvertSearchFieldsEmptyYieldsNil(t *testing.T) {
	def := pagination.NewSearchDefinition()
	sp := pagination.NewSearchPagination(map[string][]string{}, def, pagination.DefaultSearchOptions())
	require.Nil(t, sp.SearchFields)
}
