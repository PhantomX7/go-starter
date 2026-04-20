package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ---- test fixtures ---------------------------------------------------------

// testProduct exercises the full CRUD surface including soft delete.
type testProduct struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	Price     uint
	OwnerID   *uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// testOwner + testProduct.OwnerID drive the preload tests.
type testOwner struct {
	ID       uint          `gorm:"primarykey"`
	Name     string        `gorm:"uniqueIndex"`
	Products []testProduct `gorm:"foreignKey:OwnerID"`
}

// productRepo is the exact shape module repositories use: embed the generic
// base, then optionally add bespoke methods. Keeping it here makes failures
// read like the real call sites.
type productRepo struct {
	repository.BaseRepository[testProduct]
}

// ---- TestMain: init nop logger so LogSlowQuery doesn't nil-panic -----------

func TestMain(m *testing.M) {
	logger.Log = zap.NewNop()
	os.Exit(m.Run())
}

// ---- helpers ---------------------------------------------------------------

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testOwner{}, &testProduct{}))
	return db
}

func newProductRepo(db *gorm.DB) *productRepo {
	return &productRepo{BaseRepository: repository.NewBaseRepository[testProduct](db)}
}

func mustSeedProducts(t *testing.T, db *gorm.DB, rows ...testProduct) {
	t.Helper()
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}
}

func TestNewBaseRepository_UsesDefaults(t *testing.T) {
	db := setupDB(t)

	r := repository.NewBaseRepository[testProduct](db)

	assert.Same(t, db, r.DB)
	assert.Contains(t, r.EntityName(), "testProduct")
	assert.Equal(t, repository.DefaultSlowReadThreshold, r.SlowReadThreshold())
	assert.Equal(t, repository.DefaultSlowWriteThreshold, r.SlowWriteThreshold())
}

func TestNewBaseRepository_AppliesOptions(t *testing.T) {
	db := setupDB(t)

	r := repository.NewBaseRepository[testProduct](
		db,
		repository.WithSlowReadThreshold(2*time.Second),
		repository.WithSlowWriteThreshold(3*time.Second),
	)

	assert.Equal(t, 2*time.Second, r.SlowReadThreshold())
	assert.Equal(t, 3*time.Second, r.SlowWriteThreshold())
}

// ---- Association / Preload -------------------------------------------------

func TestPreload_WrapsLiteralName(t *testing.T) {
	a := repository.Preload("Products")
	assert.Equal(t, "Products", a.Name(), "Preload(name).Name() must round-trip the input")

	// And it must satisfy the Association interface statically so it can be
	// passed to FindById. This assignment is the assertion.
	var _ repository.Association = a
}

// ---- GetDB -----------------------------------------------------------------

func TestGetDB_FallsBackToDefault(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)

	got := r.GetDB(context.Background())
	assert.Same(t, db, got, "no tx in context → default DB is returned")
}

func TestGetDB_PrefersTxFromContext(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)

	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	ctx := utils.SetTxToContext(context.Background(), tx)
	got := r.GetDB(ctx)
	assert.Same(t, tx, got, "tx in context → tx wins over default DB")
}

// ---- Create ----------------------------------------------------------------

func TestCreate_InsertsAndPopulatesID(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	p := &testProduct{Name: "widget", Price: 10}
	require.NoError(t, r.Create(ctx, p))
	assert.NotZero(t, p.ID, "Create must populate the primary key")

	var got testProduct
	require.NoError(t, db.First(&got, p.ID).Error)
	assert.Equal(t, "widget", got.Name)
	assert.EqualValues(t, 10, got.Price)
}

func TestCreate_WrapsConstraintViolation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	ownerRepo := &struct {
		repository.BaseRepository[testOwner]
	}{BaseRepository: repository.NewBaseRepository[testOwner](db)}

	require.NoError(t, ownerRepo.Create(ctx, &testOwner{Name: "alice"}))

	err := ownerRepo.Create(ctx, &testOwner{Name: "alice"}) // UNIQUE(Name) violated
	require.Error(t, err)

	var app *cerrors.AppError
	require.ErrorAs(t, err, &app, "underlying error must be wrapped as *AppError")
	assert.Contains(t, app.Message, "failed to create", "message should identify the failed op")
}

// ---- FindById --------------------------------------------------------------

func TestFindById_HappyPath(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	// Create directly (not via mustSeedProducts) so we capture the populated ID.
	seed := &testProduct{Name: "widget", Price: 10}
	require.NoError(t, db.Create(seed).Error)

	got, err := r.FindById(ctx, seed.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, seed.ID, got.ID)
	assert.Equal(t, "widget", got.Name)
}

func TestFindById_NotFoundIsTypedError(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)

	_, err := r.FindById(context.Background(), 9999)
	require.Error(t, err)

	// Must unwrap both to *AppError (callers switch on this) AND to
	// ErrNotFound (services call errors.Is(err, cerrors.ErrNotFound)).
	var app *cerrors.AppError
	require.ErrorAs(t, err, &app)
	assert.True(t, errors.Is(err, cerrors.ErrNotFound), "error chain must include ErrNotFound")
}

func TestFindById_PreloadsAssociation(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	// Seed a product with an owner and two sibling products so the preload
	// actually has rows to materialize.
	owner := testOwner{Name: "acme"}
	require.NoError(t, db.Create(&owner).Error)
	mustSeedProducts(t, db,
		testProduct{Name: "widget", OwnerID: &owner.ID},
		testProduct{Name: "gizmo", OwnerID: &owner.ID},
	)

	// Find the owner and verify the has-many Products preload fires.
	ownerRepo := &struct {
		repository.BaseRepository[testOwner]
	}{BaseRepository: repository.NewBaseRepository[testOwner](db)}

	// Both typed and string-escape-hatch paths should work. Test the escape
	// hatch here because we don't have generator output for testOwner.
	got, err := ownerRepo.FindById(ctx, owner.ID, repository.Preload("Products"))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got.Products, 2, "preloaded Products should contain both seeded rows")
}

// ---- Update (Save semantics: writes all columns, including zeros) ----------

func TestUpdate_WritesAllFieldsIncludingZeros(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	p := testProduct{Name: "widget", Price: 10}
	require.NoError(t, r.Create(ctx, &p))

	// Zero out Price — Save persists the zero value (unlike Updates with
	// struct, which would skip it).
	p.Name = "widget-v2"
	p.Price = 0
	require.NoError(t, r.Update(ctx, &p))

	var after testProduct
	require.NoError(t, db.First(&after, p.ID).Error)
	assert.Equal(t, "widget-v2", after.Name)
	assert.EqualValues(t, 0, after.Price, "Save must write zero-valued fields")
}

// ---- Delete (soft delete via gorm.DeletedAt) -------------------------------

func TestDelete_SoftDeletesWhenDeletedAtPresent(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	p := testProduct{Name: "widget"}
	require.NoError(t, r.Create(ctx, &p))
	require.NoError(t, r.Delete(ctx, &p))

	// Default scope hides the row.
	_, err := r.FindById(ctx, p.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, cerrors.ErrNotFound))

	// Unscoped sees it — the row exists, it's just soft-deleted.
	var raw testProduct
	require.NoError(t, db.Unscoped().First(&raw, p.ID).Error)
	assert.True(t, raw.DeletedAt.Valid, "DeletedAt should be stamped")
}

// ---- FindAll (pagination bridge: classic Apply → generics Find) ------------

func TestFindAll_AppliesLimitOffsetAndOrder(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	mustSeedProducts(t, db,
		testProduct{Name: "a"},
		testProduct{Name: "b"},
		testProduct{Name: "c"},
		testProduct{Name: "d"},
		testProduct{Name: "e"},
	)

	// limit=2, offset=1, default order "id desc" → expect rows 4 and 3.
	pg := pagination.NewPagination(
		map[string][]string{"limit": {"2"}, "offset": {"1"}},
		nil,
		pagination.DefaultPaginationOptions(),
	)

	got, err := r.FindAll(ctx, pg)
	require.NoError(t, err)
	require.Len(t, got, 2, "pagination.Limit must bound the result")

	assert.EqualValues(t, 4, got[0].ID, "first row after offset=1 with id desc")
	assert.EqualValues(t, 3, got[1].ID)
}

func TestFindAll_CustomScopeIsApplied(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	mustSeedProducts(t, db,
		testProduct{Name: "cheap", Price: 5},
		testProduct{Name: "mid", Price: 50},
		testProduct{Name: "spendy", Price: 500},
	)

	pg := pagination.NewPagination(nil, nil, pagination.DefaultPaginationOptions())
	pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
		return db.Where("price >= ?", 50)
	})

	got, err := r.FindAll(ctx, pg)
	require.NoError(t, err)
	require.Len(t, got, 2, "only the two rows meeting the scope filter should return")
	names := []string{got[0].Name, got[1].Name}
	assert.ElementsMatch(t, []string{"mid", "spendy"}, names)
}

// ---- Count -----------------------------------------------------------------

func TestCount_MatchesFindAll(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	mustSeedProducts(t, db,
		testProduct{Name: "a"},
		testProduct{Name: "b"},
		testProduct{Name: "c"},
	)

	pg := pagination.NewPagination(nil, nil, pagination.DefaultPaginationOptions())
	count, err := r.Count(ctx, pg)
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
}

func TestCount_HonoursCustomScope(t *testing.T) {
	db := setupDB(t)
	r := newProductRepo(db)
	ctx := context.Background()

	mustSeedProducts(t, db,
		testProduct{Name: "a", Price: 5},
		testProduct{Name: "b", Price: 50},
		testProduct{Name: "c", Price: 500},
	)

	pg := pagination.NewPagination(nil, nil, pagination.DefaultPaginationOptions())
	pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
		return db.Where("price >= ?", 50)
	})

	count, err := r.Count(ctx, pg)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count, "Count must apply the same scopes as FindAll")
}

// ---- LogSlowQuery ----------------------------------------------------------

func TestLogSlowQuery_EmitsWhenOverThreshold(t *testing.T) {
	// Swap in an observable logger for this test, restore the nop afterwards.
	core, recorded := observer.New(zapcore.WarnLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	r := newProductRepo(setupDB(t))

	r.LogSlowQuery(context.Background(), "Scanning", 2*time.Second, 500*time.Millisecond)
	entries := recorded.All()
	require.Len(t, entries, 1, "exactly one warning should be recorded")
	assert.Equal(t, "Slow query detected", entries[0].Message)

	// Sanity-check the structured fields so regressions in the log shape
	// don't silently hide from dashboards.
	fields := entries[0].ContextMap()
	assert.Equal(t, "Scanning", fields["operation"])
	assert.Contains(t, fields["entity_type"], "testProduct")
}

func TestLogSlowQuery_SilentUnderThreshold(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	r := newProductRepo(setupDB(t))
	r.LogSlowQuery(context.Background(), "Quick", 10*time.Millisecond, 500*time.Millisecond)

	assert.Empty(t, recorded.All(), "fast calls must not emit slow-query warnings")
}

func TestLogSlowRead_UsesConfiguredThreshold(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	r := repository.NewBaseRepository[testProduct](
		setupDB(t),
		repository.WithSlowReadThreshold(100*time.Millisecond),
	)

	r.LogSlowRead(context.Background(), "FindAll", 150*time.Millisecond)

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "FindAll", entries[0].ContextMap()["operation"])
	assert.Equal(t, 100*time.Millisecond, entries[0].ContextMap()["threshold"])
}

func TestLogSlowWrite_UsesConfiguredThreshold(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	r := repository.NewBaseRepository[testProduct](
		setupDB(t),
		repository.WithSlowWriteThreshold(200*time.Millisecond),
	)

	r.LogSlowWrite(context.Background(), "Update", 250*time.Millisecond)

	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "Update", entries[0].ContextMap()["operation"])
	assert.Equal(t, 200*time.Millisecond, entries[0].ContextMap()["threshold"])
}
