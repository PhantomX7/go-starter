// Package pagination provides safe, typed pagination and filtering helpers for GORM queries.
package pagination

import (
	"slices"
	"strconv"
	"time"

	_ "time/tzdata" // ensures Asia/Jakarta resolves on systems without OS tzdata (e.g. Windows containers)

	"gorm.io/gorm"
)

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

// PaginationOptions configures pagination behavior.
//
//nolint:revive // PaginationOptions is kept for external API compatibility.
type PaginationOptions struct {
	DefaultLimit int
	MaxLimit     int
	MaxOffset    int
	DefaultOrder string
	Timezone     *time.Location
}

// safeDefaultOrder is the fallback used when a caller-supplied DefaultOrder
// is empty or malformed. We intentionally don't validate against the
// FilterDefinition sort registry here — DefaultOrder often references a
// primary key the caller doesn't bother to register as a user-visible sort.
const safeDefaultOrder = "id desc"

// defaultMaxOffset bounds deep-offset scans for callers using
// DefaultPaginationOptions. At the default limit of 20 this is page 50,000 —
// far past any real UI paging, while still stopping ?offset=2_000_000_000 from
// forcing the database to walk billions of rows. Callers who genuinely page
// deeper can raise (or zero out) MaxOffset explicitly.
const defaultMaxOffset = 1_000_000

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
		MaxOffset:    defaultMaxOffset,
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
	// A negative MaxOffset is meaningless; normalise it to 0 (no cap) so it
	// behaves the same as an unset field rather than clamping every request to
	// a negative offset.
	if options.MaxOffset < 0 {
		options.MaxOffset = 0
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
		Offset:     parseOffset(internal, options.MaxOffset),
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

// parseOffset reads ?offset=, clamping to maxOffset when maxOffset > 0. A
// maxOffset of 0 means "no cap" — deep-offset protection is opt-in so callers
// constructing PaginationOptions literally keep their prior behaviour.
func parseOffset(conditions map[string][]string, maxOffset int) int {
	if offsetStr, exists := conditions[QueryKeyOffset]; exists && len(offsetStr) > 0 {
		if offset, err := strconv.Atoi(offsetStr[0]); err == nil && offset >= 0 {
			if maxOffset > 0 && offset > maxOffset {
				return maxOffset
			}
			return offset
		}
	}
	return 0
}
