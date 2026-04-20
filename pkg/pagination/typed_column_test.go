package pagination_test

import (
	"strings"
	"testing"

	"github.com/PhantomX7/athleton/pkg/pagination"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Hand-rolled typed columns — we don't want this test to depend on whatever
// internal/generated happens to emit. The goal is to exercise the
// TypedColumn contract: anything satisfying `Column() clause.Column` must be
// accepted by AddFilter / AddSort and have its Name / Table promoted into the
// string-based engine.
type fakeColumn struct {
	name  string
	table string
}

func (f fakeColumn) Column() clause.Column {
	return clause.Column{Name: f.name, Table: f.table}
}

// emittedSQL extracts the SQL that a Pagination would generate, so we can
// assert the typed column's Name / Table actually made it into the WHERE /
// ORDER BY clauses. DryRun skips execution and just builds the statement.
func emittedSQL(t *testing.T, pg *pagination.Pagination) string {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)

	var sink []struct {
		ID uint
	}
	stmt := pg.Apply(db.Table("users")).Find(&sink).Statement
	return stmt.SQL.String()
}

func TestFilterConfig_TypedColumn_PromotesFieldIntoWHERE(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("email", pagination.FilterConfig{
			Column: fakeColumn{name: "email"},
			Type:   pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"email": {"eq:admin@aimo.xyz"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.Contains(t, sql, "email = ?", "typed Column.Name must land in the WHERE clause")
}

func TestFilterConfig_TypedColumn_InheritsTableFromWithTable(t *testing.T) {
	// Simulates `generated.User.Email.WithTable("users")` — i.e. the
	// generator emits a qualified column.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("email", pagination.FilterConfig{
			Column: fakeColumn{name: "email", table: "users"},
			Type:   pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"email": {"eq:admin@aimo.xyz"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	// The column comes out as `users.email` when the typed column carries a
	// Table. No explicit TableName was set on the FilterConfig — the
	// normalization path inherited it.
	assert.Contains(t, sql, "users.email", "typed column's Table must promote to TableName when none is set explicitly")
}

func TestFilterConfig_TypedColumn_BeatsStringField(t *testing.T) {
	// When both are set, typed wins — callers migrating away from the string
	// API shouldn't accidentally leave a stale Field in place.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("email", pagination.FilterConfig{
			Column: fakeColumn{name: "typed_email"},
			Field:  "string_email",
			Type:   pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"email": {"eq:x@y"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.Contains(t, sql, "typed_email", "typed Column must win over string Field")
	assert.NotContains(t, sql, "string_email", "string Field must be shadowed when Column is set")
}

func TestFilterConfig_SearchColumns_BuildsOR(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchColumns: []pagination.TypedColumn{
				fakeColumn{name: "name"},
				fakeColumn{name: "email"},
			},
			Type: pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"search": {"like:foo"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	// Multi-field LIKE search over the two typed columns.
	assert.Contains(t, sql, "name")
	assert.Contains(t, sql, "email")
	assert.Contains(t, strings.ToUpper(sql), " OR ", "SearchColumns must combine with OR for matching ops")
}

func TestFilterConfig_SearchColumns_QualifiesWithTable(t *testing.T) {
	// Mixed tables in a SearchColumns list — each column carries its own
	// Table, and they all land pre-qualified in the SQL.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchColumns: []pagination.TypedColumn{
				fakeColumn{name: "name", table: "users"},
				fakeColumn{name: "title", table: "posts"},
			},
			Type: pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"search": {"like:foo"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.Contains(t, sql, "users.name", "SearchColumns must qualify per-entry when Table is set")
	assert.Contains(t, sql, "posts.title")
}

func TestFilterConfig_StringField_StillWorks(t *testing.T) {
	// Backwards compatibility: existing Field-based registrations must keep
	// emitting the same SQL they did before the TypedColumn addition.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field: "status",
			Type:  pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"status": {"eq:active"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.Contains(t, sql, "status = ?")
}

func TestFilterConfig_RejectsUnsafeTypedColumnName(t *testing.T) {
	// Defense in depth: the isQualifiedIdent check runs on resolved fields
	// regardless of source. A typed column with a malicious Name must be
	// dropped at registration time, same as a bad string Field would be.
	filterDef := pagination.NewFilterDefinition().
		AddFilter("bad", pagination.FilterConfig{
			Column: fakeColumn{name: "name; DROP TABLE users--"},
			Type:   pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"bad": {"eq:x"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.NotContains(t, sql, "DROP TABLE", "unsafe typed column Name must be dropped at registration")
}

func TestSortConfig_TypedColumn_PromotesIntoORDERBY(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddSort("email", pagination.SortConfig{
			Column:  fakeColumn{name: "email"},
			Allowed: true,
		})

	pg := pagination.NewPagination(
		map[string][]string{"sort": {"email desc"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	// Default order is "id desc" if sort parsing fails; a successful typed
	// sort replaces it with the requested column.
	assert.Contains(t, strings.ToLower(sql), "order by email desc", "typed SortConfig.Column must emit ORDER BY Name")
}

func TestSortConfig_TypedColumn_WithTableQualifies(t *testing.T) {
	filterDef := pagination.NewFilterDefinition().
		AddSort("email", pagination.SortConfig{
			Column:  fakeColumn{name: "email", table: "users"},
			Allowed: true,
		})

	pg := pagination.NewPagination(
		map[string][]string{"sort": {"users.email desc"}},
		filterDef,
		pagination.DefaultPaginationOptions(),
	)

	sql := emittedSQL(t, pg)
	assert.Contains(t, strings.ToLower(sql), "order by users.email desc")
}

func TestFilterConfig_NilColumn_FallsBackToStringField(t *testing.T) {
	// A nil TypedColumn must not panic — the engine should quietly fall
	// through to the string path.
	require.NotPanics(t, func() {
		filterDef := pagination.NewFilterDefinition().
			AddFilter("status", pagination.FilterConfig{
				Column: nil,
				Field:  "status",
				Type:   pagination.FilterTypeString,
			})

		pg := pagination.NewPagination(
			map[string][]string{"status": {"eq:active"}},
			filterDef,
			pagination.DefaultPaginationOptions(),
		)

		sql := emittedSQL(t, pg)
		assert.Contains(t, sql, "status = ?")
	})
}
