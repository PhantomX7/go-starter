package pagination_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/PhantomX7/athleton/pkg/pagination"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// dryRunSQL exposes the SQL a Pagination would emit without executing it.
// Shared helper for the hardening tests below so we can assert fragment
// shape across DB dialects without spinning up a real database.
func dryRunSQL(t *testing.T, pg *pagination.Pagination) string {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)
	var sink []struct {
		ID uint
	}
	stmt := pg.Apply(db.Table("users")).Find(&sink).Statement
	return stmt.SQL.String()
}

// ---- B1: datetime half-open semantics ---------------------------------------

func TestDateTimeEquals_IsHalfOpen(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("ts", pagination.FilterConfig{Field: "ts", Type: pagination.FilterTypeDateTime})

	pg := pagination.NewPagination(
		map[string][]string{"ts": {"2024-01-01 12:00:00"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "ts >= ?")
	assert.Contains(t, sql, "ts < ?", "datetime Equals must upper-bound with < (next second), not =")
	assert.NotContains(t, sql, "ts = ?", "closed = would miss sub-second rows")
}

func TestDateTimeBetween_IsHalfOpen(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("ts", pagination.FilterConfig{Field: "ts", Type: pagination.FilterTypeDateTime})

	pg := pagination.NewPagination(
		map[string][]string{"ts": {"between:2024-01-01 00:00:00,2024-01-01 23:59:59"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "ts >= ?")
	assert.Contains(t, sql, "ts < ?")
	assert.NotContains(t, sql, "BETWEEN",
		"closed BETWEEN re-introduces the sub-second precision bug — see buildDateTimeScope doc")
}

func TestDateTimeLte_IsHalfOpen(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("ts", pagination.FilterConfig{Field: "ts", Type: pagination.FilterTypeDateTime})

	pg := pagination.NewPagination(
		map[string][]string{"ts": {"lte:2024-01-01 23:59:59"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "ts < ?", "datetime Lte must upper-bound with < (next second)")
	assert.NotContains(t, sql, "ts <= ?", "closed <= would miss rows stored with fractional seconds")
}

// ---- B2: LIKE escape character (portable across MySQL/Postgres/SQLite) ------

func TestLikeEscape_UsesBangChar_NotBackslash(t *testing.T) {
	// Backslash ESCAPE is a MySQL-default-mode hazard: the server reads
	// "ESCAPE '\'" as an unterminated literal. '!' is non-special in every
	// DB in go.mod.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString})

	pg := pagination.NewPagination(
		map[string][]string{"name": {"like:foo"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "ESCAPE '!'", "LIKE must use '!' escape for cross-DB portability")
	assert.NotContains(t, sql, `ESCAPE '\`, "backslash ESCAPE breaks on MySQL default SQL mode")
}

func TestLikeEscape_MultiField_UsesBangChar(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchFields: []string{"name", "email"},
			Type:         pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"search": {"like:foo"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	// Both field fragments carry the '!' escape clause.
	assert.Equal(t, 2, strings.Count(sql, "ESCAPE '!'"),
		"multi-field LIKE must carry ESCAPE '!' on every field fragment")
}

func TestLikeEscape_DoublesLiteralBang(t *testing.T) {
	// '!' in user input must survive escaping as '!!' so the LIKE pattern
	// matches a literal '!' in the target column rather than treating the
	// input as a control char.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	type Row struct {
		ID   uint `gorm:"primaryKey"`
		Name string
	}
	require.NoError(t, db.AutoMigrate(&Row{}))
	require.NoError(t, db.Create([]Row{
		{ID: 1, Name: "Hello!"},
		{ID: 2, Name: "Hello World"},
		{ID: 3, Name: "!important"},
	}).Error)

	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString})

	pg := pagination.NewPagination(
		map[string][]string{"name": {"like:!"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	var got []Row
	require.NoError(t, pg.Apply(db.Model(&Row{})).Find(&got).Error)
	assert.Len(t, got, 2, "only rows containing a literal '!' should match")
}

// ---- BP1: GetAllowedOperators returns a fresh slice -------------------------

func TestGetAllowedOperators_IsClone_DefaultsPath(t *testing.T) {
	cfg := pagination.FilterConfig{Type: pagination.FilterTypeString}

	// First read: cache-typical defaults.
	first := cfg.GetAllowedOperators()
	original := make([]pagination.FilterOperator, len(first))
	copy(original, first)

	// Poison attempt — this used to clobber the package-level
	// operatorsByType[FilterTypeString] slice in-place.
	for i := range first {
		first[i] = "evil"
	}

	// Second read must still reflect the true defaults, not "evil".
	second := cfg.GetAllowedOperators()
	assert.Equal(t, original, second, "mutating the returned slice must not affect subsequent reads")
}

func TestGetAllowedOperators_IsClone_CustomPath(t *testing.T) {
	// Same guarantee when the config carries its own Operators list: the
	// caller's slice must not be handed back by reference.
	seed := []pagination.FilterOperator{pagination.OperatorEquals, pagination.OperatorLike}
	cfg := pagination.FilterConfig{
		Type:      pagination.FilterTypeString,
		Operators: seed,
	}

	got := cfg.GetAllowedOperators()
	got[0] = "evil"

	assert.Equal(t, pagination.OperatorEquals, seed[0],
		"mutation on the returned slice must not bleed into the caller's Operators slice")
}

// ---- P2-1: NewPagination deep-clones input conditions -----------------------

func TestNewPagination_DeepClonesInputConditions(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString})

	conditions := map[string][]string{"status": {"active"}}
	pg := pagination.NewPagination(conditions, filterDef, pagination.DefaultPaginationOptions())

	// Mutate both the outer map and the inner slice — neither should affect
	// queries issued through pg afterwards.
	conditions["status"][0] = "inactive"
	conditions["injected"] = []string{"evil"}
	delete(conditions, "status")

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "status = ?", "original 'status' filter must still fire after caller mutation")
	assert.NotContains(t, sql, "injected", "filters added to the caller's map post-construction must not appear")
}

// ---- P2-2: GetConditions deep-clones on egress ------------------------------

func TestGetConditions_DeepClonesInnerSlices(t *testing.T) {
	// The existing TestGetConditionsReturnsClone covered the outer map;
	// this one asserts the inner []string values are independent too.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString})

	pg := pagination.NewPagination(
		map[string][]string{"status": {"active", "pending"}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	got := pg.GetConditions()
	require.Len(t, got["status"], 2)
	got["status"][0] = "evil"

	fresh := pg.GetConditions()
	assert.Equal(t, "active", fresh["status"][0],
		"mutating an inner slice of the returned map must not affect pagination state")
}

// ---- P2-3: DefaultLimit is clamped by MaxLimit ------------------------------

func TestDefaultLimit_IsClampedByMaxLimit(t *testing.T) {
	// Caller misconfigured: DefaultLimit > MaxLimit. The hard cap must win
	// even for requests that omit ?limit=.
	pg := pagination.NewPagination(
		nil, nil,
		pagination.PaginationOptions{DefaultLimit: 500, MaxLimit: 100},
	)

	assert.Equal(t, 100, pg.Limit, "DefaultLimit exceeding MaxLimit must be clamped at construction")
}

func TestDefaultLimit_BelowMaxLimit_Preserved(t *testing.T) {
	// Sanity check: a valid DefaultLimit is untouched by the clamp.
	pg := pagination.NewPagination(
		nil, nil,
		pagination.PaginationOptions{DefaultLimit: 25, MaxLimit: 100},
	)

	assert.Equal(t, 25, pg.Limit)
}

// ---- A1: over-cap IN list fails closed instead of widening -------------------

func TestInList_OverCap_FailsClosed(t *testing.T) {
	// An `in:` list longer than maxFilterValues (256) must NOT silently drop
	// the filter — dropping it returns every row, violating the package's
	// "malformed input never widens the result set" invariant. It must fail
	// closed (WHERE 1 = 0), mirroring the all-unparseable IN handling.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{Field: "id", Type: pagination.FilterTypeID})

	vals := make([]string, 257) // one over the 256 cap
	for i := range vals {
		vals[i] = strconv.Itoa(i + 1)
	}

	pg := pagination.NewPagination(
		map[string][]string{"id": {"in:" + strings.Join(vals, ",")}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.Contains(t, sql, "1 = 0",
		"an over-cap IN list must fail closed, not drop the filter and return all rows")
}

func TestInList_OverCap_DisallowedOperator_Dropped(t *testing.T) {
	// When the over-cap operator isn't even valid for the type, the filter is
	// dropped just as a small invalid filter would be — no spurious 1 = 0.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString})

	vals := make([]string, 257)
	for i := range vals {
		vals[i] = strconv.Itoa(i + 1)
	}

	// `between` is not a valid String operator, so this must be dropped, not 1=0.
	pg := pagination.NewPagination(
		map[string][]string{"name": {"between:" + strings.Join(vals, ",")}},
		filterDef, pagination.DefaultPaginationOptions(),
	)

	sql := dryRunSQL(t, pg)
	assert.NotContains(t, sql, "1 = 0",
		"an over-cap list on a disallowed operator is dropped, not failed closed")
}

// ---- A2: offset is clamped to MaxOffset -------------------------------------

func TestOffset_ClampedToMaxOffset(t *testing.T) {
	// Symmetric to limit clamping: a request offset beyond MaxOffset is capped
	// rather than handed to the DB verbatim (deep-offset scan protection).
	pg := pagination.NewPagination(
		map[string][]string{"offset": {"999999999"}},
		nil,
		pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, MaxOffset: 10000},
	)

	assert.Equal(t, 10000, pg.Offset, "offset beyond MaxOffset must be clamped")
}

func TestOffset_Unlimited_WhenMaxOffsetZero(t *testing.T) {
	// MaxOffset == 0 means "no cap" so existing callers that construct
	// PaginationOptions literally keep their current behaviour.
	pg := pagination.NewPagination(
		map[string][]string{"offset": {"999999999"}},
		nil,
		pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, MaxOffset: 0},
	)

	assert.Equal(t, 999999999, pg.Offset, "MaxOffset=0 means no cap (backward compatible)")
}
