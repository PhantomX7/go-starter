package bleve_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/libs/bleve"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
)

type doc struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	IsActive bool    `json:"is_active"`
}

// newTestClient builds a Bleve client rooted in a temp directory. The client
// is closed before the temp dir is removed (cleanups run LIFO).
func newTestClient(t *testing.T) bleve.Client {
	t.Helper()

	restore := config.SetForTesting(&config.Config{
		Bleve: config.BleveConfig{IndexPath: t.TempDir()},
	})
	t.Cleanup(restore)

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() { logger.Log = prev })

	c, err := bleve.NewBleveClient()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	return c
}

func seedDocs(t *testing.T, c bleve.Client, indexName string) {
	t.Helper()

	require.NoError(t, c.GetOrCreateIndex(indexName, nil))

	docs := map[string]any{
		"1": doc{Name: "red running shoes", Category: "shoes", Price: 50, IsActive: true},
		"2": doc{Name: "blue running shorts", Category: "apparel", Price: 25, IsActive: true},
		"3": doc{Name: "green tennis racket", Category: "equipment", Price: 120, IsActive: false},
		"4": doc{Name: "red tennis shoes", Category: "shoes", Price: 80, IsActive: true},
	}
	require.NoError(t, c.BulkIndexDocument(context.Background(), indexName, docs))
}

func hitIDs(res *bleve.SearchResult) []string {
	ids := make([]string, len(res.Hits))
	for i, h := range res.Hits {
		ids[i] = h.ID
	}
	return ids
}

func TestGetOrCreateIndexAndExists(t *testing.T) {
	c := newTestClient(t)

	require.False(t, c.IndexExists("products"))
	require.NoError(t, c.GetOrCreateIndex("products", nil))
	require.True(t, c.IndexExists("products"))

	// Idempotent on an already-open index.
	require.NoError(t, c.GetOrCreateIndex("products", nil))
}

func TestOperationsOnMissingIndexReturnErrIndexNotFound(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	require.ErrorIs(t, c.IndexDocument(ctx, "missing", "1", doc{Name: "x"}), bleve.ErrIndexNotFound)
	require.ErrorIs(t, c.DeleteDocument(ctx, "missing", "1"), bleve.ErrIndexNotFound)

	_, err := c.Search(ctx, "missing", &bleve.SearchRequest{Query: "x"})
	require.ErrorIs(t, err, bleve.ErrIndexNotFound)

	_, err = c.GetStats("missing")
	require.ErrorIs(t, err, bleve.ErrIndexNotFound)

	err = c.BulkIndexDocument(ctx, "missing", map[string]any{"1": doc{Name: "x"}})
	require.ErrorIs(t, err, bleve.ErrIndexNotFound)
}

func TestIndexAndSearchDocuments(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	res, err := c.Search(context.Background(), "products", &bleve.SearchRequest{Query: "running"})
	require.NoError(t, err)
	require.EqualValues(t, 2, res.Total)
	require.ElementsMatch(t, []string{"1", "2"}, hitIDs(res))
	require.Positive(t, res.MaxScore)
}

func TestSearchMatchAllWhenNoQueryOrFilters(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	res, err := c.Search(context.Background(), "products", &bleve.SearchRequest{})
	require.NoError(t, err)
	require.EqualValues(t, 4, res.Total)
	require.Len(t, res.Hits, 4)
}

func TestSearchPagination(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	ctx := context.Background()
	sort := []string{"price"}

	page1, err := c.Search(ctx, "products", &bleve.SearchRequest{Size: 2, From: 0, Sort: sort})
	require.NoError(t, err)
	require.EqualValues(t, 4, page1.Total)
	require.Equal(t, []string{"2", "1"}, hitIDs(page1))

	page2, err := c.Search(ctx, "products", &bleve.SearchRequest{Size: 2, From: 2, Sort: sort})
	require.NoError(t, err)
	require.EqualValues(t, 4, page2.Total)
	require.Equal(t, []string{"4", "3"}, hitIDs(page2))

	// From beyond the result set yields no hits but keeps the total.
	empty, err := c.Search(ctx, "products", &bleve.SearchRequest{Size: 2, From: 10, Sort: sort})
	require.NoError(t, err)
	require.EqualValues(t, 4, empty.Total)
	require.Empty(t, empty.Hits)
}

func TestSearchFilters(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")
	ctx := context.Background()

	t.Run("exact", func(t *testing.T) {
		res, err := c.Search(ctx, "products", &bleve.SearchRequest{
			Filters: map[string]bleve.Filter{
				"category": {Type: bleve.FilterExact, Value: "shoes"},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"1", "4"}, hitIDs(res))
	})

	t.Run("bool", func(t *testing.T) {
		res, err := c.Search(ctx, "products", &bleve.SearchRequest{
			Filters: map[string]bleve.Filter{
				"is_active": {Type: bleve.FilterBool, Value: false},
			},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"3"}, hitIDs(res))
	})

	t.Run("range", func(t *testing.T) {
		minPrice, maxPrice := 40.0, 100.0
		res, err := c.Search(ctx, "products", &bleve.SearchRequest{
			Filters: map[string]bleve.Filter{
				"price": {Type: bleve.FilterRange, Value: bleve.RangeValue{Min: &minPrice, Max: &maxPrice}},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"1", "4"}, hitIDs(res))
	})

	t.Run("in", func(t *testing.T) {
		res, err := c.Search(ctx, "products", &bleve.SearchRequest{
			Filters: map[string]bleve.Filter{
				"category": {Type: bleve.FilterIn, Value: []string{"apparel", "equipment"}},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"2", "3"}, hitIDs(res))
	})

	t.Run("query combined with filter", func(t *testing.T) {
		res, err := c.Search(ctx, "products", &bleve.SearchRequest{
			Query: "red",
			SearchFields: []bleve.SearchFieldConfig{
				{Field: "name", Boost: 1},
			},
			Filters: map[string]bleve.Filter{
				"category": {Type: bleve.FilterExact, Value: "shoes"},
			},
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"1", "4"}, hitIDs(res))
	})
}

func TestSearchHighlightAndFacets(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	res, err := c.Search(context.Background(), "products", &bleve.SearchRequest{
		Query:     "tennis",
		Highlight: []string{"name"},
		Facets:    []bleve.FacetRequest{{Field: "category"}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, res.Total)

	for _, h := range res.Hits {
		require.NotEmpty(t, h.Highlight["name"], "hit %s should have a name highlight", h.ID)
	}

	facets, ok := res.Facets["category"]
	require.True(t, ok)
	counts := make(map[string]int, len(facets))
	for _, f := range facets {
		counts[f.Value] = f.Count
	}
	require.Equal(t, map[string]int{"shoes": 1, "equipment": 1}, counts)
}

func TestSearchFuzzyMatching(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	res, err := c.Search(context.Background(), "products", &bleve.SearchRequest{
		// One substituted letter: "runnimg" should still match "running"
		// via the default fuzzy search field configuration.
		Query: "runnimg",
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1", "2"}, hitIDs(res))
}

func TestDeleteDocument(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")
	ctx := context.Background()

	require.NoError(t, c.DeleteDocument(ctx, "products", "1"))

	stats, err := c.GetStats("products")
	require.NoError(t, err)
	require.EqualValues(t, 3, stats.DocCount)

	res, err := c.Search(ctx, "products", &bleve.SearchRequest{Query: "running"})
	require.NoError(t, err)
	require.Equal(t, []string{"2"}, hitIDs(res))
}

func TestIndexDocumentSingle(t *testing.T) {
	c := newTestClient(t)
	require.NoError(t, c.GetOrCreateIndex("items", nil))
	ctx := context.Background()

	require.NoError(t, c.IndexDocument(ctx, "items", "a", doc{Name: "lonely item"}))

	stats, err := c.GetStats("items")
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.DocCount)

	// Re-indexing the same ID updates rather than duplicates.
	require.NoError(t, c.IndexDocument(ctx, "items", "a", doc{Name: "updated item"}))
	stats, err = c.GetStats("items")
	require.NoError(t, err)
	require.EqualValues(t, 1, stats.DocCount)

	res, err := c.Search(ctx, "items", &bleve.SearchRequest{Query: "updated"})
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, hitIDs(res))
}

func TestBulkIndexDocumentEmptyIsNoop(t *testing.T) {
	c := newTestClient(t)

	// No index needed: empty input short-circuits before index lookup.
	require.NoError(t, c.BulkIndexDocument(context.Background(), "missing", nil))
}

func TestDeleteIndex(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	require.NoError(t, c.DeleteIndex("products"))
	require.False(t, c.IndexExists("products"))

	_, err := c.Search(context.Background(), "products", &bleve.SearchRequest{})
	require.ErrorIs(t, err, bleve.ErrIndexNotFound)

	// Deleting a missing index is not an error.
	require.NoError(t, c.DeleteIndex("products"))
}

func TestCanceledContextIsRejected(t *testing.T) {
	c := newTestClient(t)
	seedDocs(t, c, "products")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, c.IndexDocument(ctx, "products", "9", doc{Name: "x"}), context.Canceled)
	require.ErrorIs(t, c.DeleteDocument(ctx, "products", "1"), context.Canceled)

	_, err := c.Search(ctx, "products", &bleve.SearchRequest{})
	require.ErrorIs(t, err, context.Canceled)

	err = c.BulkIndexDocument(ctx, "products", map[string]any{"9": doc{Name: "x"}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestIndexPersistsOnDiskAcrossClients(t *testing.T) {
	dir := t.TempDir()

	restore := config.SetForTesting(&config.Config{
		Bleve: config.BleveConfig{IndexPath: dir},
	})
	t.Cleanup(restore)

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() { logger.Log = prev })

	first, err := bleve.NewBleveClient()
	require.NoError(t, err)
	require.NoError(t, first.GetOrCreateIndex("persist", nil))
	require.NoError(t, first.IndexDocument(context.Background(), "persist", "1", doc{Name: "durable"}))
	require.NoError(t, first.Close())

	// A new client over the same path opens the index lazily from disk.
	second, err := bleve.NewBleveClient()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Close()) })

	require.True(t, second.IndexExists("persist"))

	res, err := second.Search(context.Background(), "persist", &bleve.SearchRequest{Query: "durable"})
	require.NoError(t, err)
	require.Equal(t, []string{"1"}, hitIDs(res))
}

func TestDefaultSearchFields(t *testing.T) {
	fields := bleve.DefaultSearchFields()
	require.Len(t, fields, 3)
	for _, f := range fields {
		require.Equal(t, "name", f.Field)
	}
	require.Equal(t, bleve.BoostExactMatch, fields[0].Boost)
	require.True(t, fields[0].Phrase)
	require.Equal(t, bleve.DefaultFuzziness, fields[2].Fuzziness)
}
