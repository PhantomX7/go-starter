package bleve

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"go.uber.org/zap"
)

// FacetRequest defines a facet request with optional size
type FacetRequest struct {
	Field string
	Size  int
}

const (
	DefaultSearchSize = 10
	DefaultFacetSize  = 10
	DefaultFuzziness  = 2

	// Default boost values for query prioritization
	BoostExactMatch  = 10.0
	BoostPhraseMatch = 5.0
	BoostTermMatch   = 3.0
	BoostFuzzyMatch  = 1.0
)

var (
	ErrIndexNotFound = errors.New("index not found")
)

// FilterType defines the type of filter
type FilterType uint8

const (
	FilterExact FilterType = iota
	FilterMatch
	FilterRange
	FilterIn
	FilterBool
)

// Filter represents a search filter
type Filter struct {
	Type  FilterType
	Value any
}

// RangeValue represents min/max range
type RangeValue struct {
	Min, Max *float64
}

// SearchFieldConfig defines how a field should be searched
type SearchFieldConfig struct {
	Field     string  // Field name in the index
	Boost     float64 // Boost value for ranking
	Fuzziness int     // 0 = no fuzziness, 1-2 = fuzzy matching
	Phrase    bool    // true = phrase match, false = term match
}

// SearchRequest defines search parameters
type SearchRequest struct {
	Query        string
	Filters      map[string]Filter
	Sort         []string
	Size         int
	From         int
	Highlight    []string
	Facets       []FacetRequest
	SearchFields []SearchFieldConfig // Configurable search fields
}

// SearchResult contains search results
type SearchResult struct {
	Total    uint64
	Hits     []Hit
	MaxScore float64
	Took     time.Duration
	Facets   map[string][]Facet
}

// Hit represents a single search result
type Hit struct {
	ID        string
	Score     float64
	Highlight map[string][]string
}

// Facet represents aggregated data
type Facet struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// Stats represents index statistics
type Stats struct {
	DocCount uint64
}

// DefaultSearchFields returns a basic search configuration for a "name" field
func DefaultSearchFields() []SearchFieldConfig {
	return []SearchFieldConfig{
		{Field: "name", Boost: BoostExactMatch, Phrase: true},                               // Exact phrase match
		{Field: "name", Boost: BoostPhraseMatch, Fuzziness: 0, Phrase: false},               // All terms must match
		{Field: "name", Boost: BoostFuzzyMatch, Fuzziness: DefaultFuzziness, Phrase: false}, // Fuzzy match
	}
}

// Client interface for Bleve operations
type Client interface {
	IndexDocument(ctx context.Context, indexName, docID string, doc any) error
	BulkIndexDocument(ctx context.Context, indexName string, docs map[string]any) error
	DeleteDocument(ctx context.Context, indexName, docID string) error
	Search(ctx context.Context, indexName string, req *SearchRequest) (*SearchResult, error)
	GetOrCreateIndex(indexName string, mapping mapping.IndexMapping) error
	DeleteIndex(indexName string) error
	IndexExists(indexName string) bool
	GetStats(indexName string) (*Stats, error)
	Close() error
}

type client struct {
	mu        sync.RWMutex
	indices   map[string]bleve.Index
	indexPath string
}

// NewBleveClient creates a new Bleve client
func NewBleveClient() (Client, error) {
	cfg := config.Get().Bleve

	if err := os.MkdirAll(cfg.IndexPath, 0755); err != nil {
		return nil, fmt.Errorf("create index directory: %w", err)
	}

	logger.Info("Bleve client initialized", zap.String("path", cfg.IndexPath))

	return &client{
		indices:   make(map[string]bleve.Index),
		indexPath: cfg.IndexPath,
	}, nil
}

func (c *client) getIndex(indexName string) (bleve.Index, error) {
	c.mu.RLock()
	if idx, ok := c.indices[indexName]; ok {
		c.mu.RUnlock()
		return idx, nil
	}
	c.mu.RUnlock()

	// Try to open from disk
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if idx, ok := c.indices[indexName]; ok {
		return idx, nil
	}

	idx, err := bleve.Open(filepath.Join(c.indexPath, indexName))
	if err != nil {
		return nil, ErrIndexNotFound
	}

	c.indices[indexName] = idx
	return idx, nil
}

func (c *client) IndexExists(indexName string) bool {
	c.mu.RLock()
	if _, ok := c.indices[indexName]; ok {
		c.mu.RUnlock()
		return true
	}
	c.mu.RUnlock()

	_, err := os.Stat(filepath.Join(c.indexPath, indexName))
	return err == nil
}

func (c *client) GetOrCreateIndex(indexName string, indexMapping mapping.IndexMapping) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.indices[indexName]; ok {
		return nil
	}

	path := filepath.Join(c.indexPath, indexName)

	// Try open existing
	if idx, err := bleve.Open(path); err == nil {
		c.indices[indexName] = idx
		logger.Info("Opened existing index", zap.String("name", indexName))
		return nil
	}

	// Create new
	if indexMapping == nil {
		indexMapping = bleve.NewIndexMapping()
	}

	idx, err := bleve.New(path, indexMapping)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	c.indices[indexName] = idx
	logger.Info("Created new index", zap.String("name", indexName))
	return nil
}

func (c *client) IndexDocument(ctx context.Context, indexName, docID string, doc any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	idx, err := c.getIndex(indexName)
	if err != nil {
		return err
	}

	return idx.Index(docID, doc)
}

func (c *client) BulkIndexDocument(ctx context.Context, indexName string, docs map[string]any) error {
	if len(docs) == 0 {
		return nil
	}

	idx, err := c.getIndex(indexName)
	if err != nil {
		return err
	}

	batch := idx.NewBatch()
	for id, doc := range docs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := batch.Index(id, doc); err != nil {
			return fmt.Errorf("batch index %s: %w", id, err)
		}
	}

	return idx.Batch(batch)
}

func (c *client) DeleteDocument(ctx context.Context, indexName, docID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	idx, err := c.getIndex(indexName)
	if err != nil {
		return err
	}

	return idx.Delete(docID)
}

func (c *client) DeleteIndex(indexName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if idx, ok := c.indices[indexName]; ok {
		_ = idx.Close()
		delete(c.indices, indexName)
	}

	path := filepath.Join(c.indexPath, indexName)
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete index: %w", err)
	}

	logger.Info("Index deleted", zap.String("name", indexName))
	return nil
}

func (c *client) Search(ctx context.Context, indexName string, req *SearchRequest) (*SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	idx, err := c.getIndex(indexName)
	if err != nil {
		return nil, err
	}

	searchReq := c.buildSearchRequest(req)
	start := time.Now()

	result, err := idx.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return c.toSearchResult(result, time.Since(start)), nil
}

func (c *client) buildSearchRequest(req *SearchRequest) *bleve.SearchRequest {
	if req == nil {
		req = &SearchRequest{}
	}

	searchReq := bleve.NewSearchRequest(c.buildQuery(req))

	// Pagination
	searchReq.Size = req.Size
	if searchReq.Size == 0 {
		searchReq.Size = DefaultSearchSize
	}
	searchReq.From = req.From

	// Sorting
	if len(req.Sort) > 0 {
		searchReq.SortBy(req.Sort)
	}

	// Highlighting
	if len(req.Highlight) > 0 {
		hl := bleve.NewHighlight()
		for _, f := range req.Highlight {
			hl.AddField(f)
		}
		searchReq.Highlight = hl
	}

	// Facets
	for _, f := range req.Facets {
		size := f.Size
		if size <= 0 {
			size = DefaultFacetSize
		}
		searchReq.AddFacet(f.Field, bleve.NewFacetRequest(f.Field, size))
	}

	return searchReq
}

func (c *client) buildQuery(req *SearchRequest) query.Query {
	var queries []query.Query

	// Text query with configurable fields
	if req.Query != "" {
		textQuery := c.buildTextQuery(req.Query, req.SearchFields)
		queries = append(queries, textQuery)
	}

	// Filters
	for field, filter := range req.Filters {
		if q := c.buildFilterQuery(field, filter); q != nil {
			queries = append(queries, q)
		}
	}

	switch len(queries) {
	case 0:
		return bleve.NewMatchAllQuery()
	case 1:
		return queries[0]
	default:
		return bleve.NewConjunctionQuery(queries...)
	}
}

// buildTextQuery creates a boosted query across configured fields
func (c *client) buildTextQuery(searchText string, fields []SearchFieldConfig) query.Query {
	// Use defaults if no fields configured
	if len(fields) == 0 {
		fields = DefaultSearchFields()
	}

	shouldQueries := make([]query.Query, 0, len(fields))

	for _, f := range fields {
		var q query.Query

		if f.Phrase {
			// Phrase query - matches exact phrase in order
			pq := bleve.NewMatchPhraseQuery(searchText)
			pq.SetField(f.Field)
			pq.SetBoost(f.Boost)
			q = pq
		} else if f.Fuzziness > 0 {
			// Fuzzy match query
			mq := bleve.NewMatchQuery(searchText)
			mq.SetField(f.Field)
			mq.SetFuzziness(f.Fuzziness)
			mq.SetBoost(f.Boost)
			q = mq
		} else {
			// Standard match query with AND operator (all terms must match)
			mq := bleve.NewMatchQuery(searchText)
			mq.SetField(f.Field)
			mq.SetBoost(f.Boost)
			mq.SetOperator(query.MatchQueryOperatorAnd)
			q = mq
		}

		shouldQueries = append(shouldQueries, q)
	}

	// Single field doesn't need disjunction
	if len(shouldQueries) == 1 {
		return shouldQueries[0]
	}

	// Combine with DisjunctionQuery (OR) - any can match, but boosts determine ranking
	disjunction := bleve.NewDisjunctionQuery(shouldQueries...)
	disjunction.SetMin(1) // At least one should match

	return disjunction
}

func (c *client) buildFilterQuery(field string, filter Filter) query.Query {
	switch filter.Type {
	case FilterExact:
		q := bleve.NewTermQuery(fmt.Sprint(filter.Value))
		q.SetField(field)
		return q

	case FilterMatch:
		q := bleve.NewMatchQuery(fmt.Sprint(filter.Value))
		q.SetField(field)
		return q

	case FilterBool:
		term := "F"
		switch v := filter.Value.(type) {
		case bool:
			if v {
				term = "T"
			}
		case string:
			if strings.EqualFold(v, "true") {
				term = "T"
			}
		}
		q := bleve.NewTermQuery(term)
		q.SetField(field)
		return q

	case FilterRange:
		rv, ok := filter.Value.(RangeValue)
		if !ok {
			return nil
		}
		q := bleve.NewNumericRangeQuery(rv.Min, rv.Max)
		q.SetField(field)
		return q

	case FilterIn:
		values, ok := filter.Value.([]string)
		if !ok || len(values) == 0 {
			return nil
		}
		if len(values) == 1 {
			q := bleve.NewTermQuery(values[0])
			q.SetField(field)
			return q
		}
		queries := make([]query.Query, len(values))
		for i, v := range values {
			q := bleve.NewTermQuery(v)
			q.SetField(field)
			queries[i] = q
		}
		return bleve.NewDisjunctionQuery(queries...)

	default:
		return nil
	}
}

func (c *client) toSearchResult(result *bleve.SearchResult, took time.Duration) *SearchResult {
	hits := make([]Hit, len(result.Hits))
	for i, h := range result.Hits {
		hits[i] = Hit{
			ID:        h.ID,
			Score:     h.Score,
			Highlight: h.Fragments,
		}
	}

	facets := make(map[string][]Facet, len(result.Facets))
	for name, fr := range result.Facets {
		if fr == nil || fr.Terms == nil {
			continue
		}
		terms := fr.Terms.Terms()
		facetList := make([]Facet, len(terms))
		for i, t := range terms {
			facetList[i] = Facet{Value: t.Term, Count: t.Count}
		}
		facets[name] = facetList
	}

	return &SearchResult{
		Total:    result.Total,
		Hits:     hits,
		MaxScore: result.MaxScore,
		Took:     took,
		Facets:   facets,
	}
}

func (c *client) GetStats(indexName string) (*Stats, error) {
	idx, err := c.getIndex(indexName)
	if err != nil {
		return nil, err
	}

	count, err := idx.DocCount()
	if err != nil {
		return nil, err
	}

	return &Stats{DocCount: count}, nil
}

func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for name, idx := range c.indices {
		if err := idx.Close(); err != nil {
			lastErr = err
		} else {
			logger.Info("Index closed", zap.String("name", name))
		}
	}

	c.indices = make(map[string]bleve.Index)
	return lastErr
}
