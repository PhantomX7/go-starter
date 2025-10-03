package pagination_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/PhantomX7/go-starter/pkg/pagination"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// Test Models
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"index"`
	Email     string    `gorm:"uniqueIndex"`
	Age       int       `gorm:"index"`
	Status    string    `gorm:"index"`
	IsActive  bool      `gorm:"index"`
	Role      string    `gorm:"index"`
	CreatedAt time.Time `gorm:"index"`
}

type PaginationTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (suite *PaginationTestSuite) SetupSuite() {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	suite.Require().NoError(err)

	suite.db = db

	// Seed test data
	suite.seedData()
}

func (suite *PaginationTestSuite) seedData() {
	users := []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Age: 25, Status: "active", IsActive: true, Role: "admin", CreatedAt: time.Now().AddDate(0, 0, -10)},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 30, Status: "active", IsActive: true, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -9)},
		{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive", IsActive: false, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -8)},
		{ID: 4, Name: "Alice Brown", Email: "alice@example.com", Age: 28, Status: "active", IsActive: true, Role: "moderator", CreatedAt: time.Now().AddDate(0, 0, -7)},
		{ID: 5, Name: "Charlie Wilson", Email: "charlie@example.com", Age: 32, Status: "pending", IsActive: true, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -6)},
		{ID: 6, Name: "Diana Davis", Email: "diana@example.com", Age: 27, Status: "active", IsActive: true, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -5)},
		{ID: 7, Name: "Eve Miller", Email: "eve@example.com", Age: 29, Status: "inactive", IsActive: false, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -4)},
		{ID: 8, Name: "Frank Moore", Email: "frank@example.com", Age: 31, Status: "active", IsActive: true, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -3)},
		{ID: 9, Name: "Grace Taylor", Email: "grace@example.com", Age: 26, Status: "active", IsActive: true, Role: "admin", CreatedAt: time.Now().AddDate(0, 0, -2)},
		{ID: 10, Name: "Henry Anderson", Email: "henry@example.com", Age: 33, Status: "pending", IsActive: true, Role: "user", CreatedAt: time.Now().AddDate(0, 0, -1)},
	}

	suite.db.Create(&users)
}

func (suite *PaginationTestSuite) TearDownTest() {
	// Clean up after each test if needed
}

// Test Basic Pagination
func (suite *PaginationTestSuite) TestBasicPagination() {
	conditions := map[string][]string{
		"limit":  {"3"},
		"offset": {"0"},
	}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{
		DefaultLimit: 10,
		MaxLimit:     100,
		DefaultOrder: "id asc",
	}

	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Equal(3, len(users))
	suite.Equal(uint(1), users[0].ID)
	suite.Equal(uint(2), users[1].ID)
	suite.Equal(uint(3), users[2].ID)
}

// Test Pagination with Offset
func (suite *PaginationTestSuite) TestPaginationWithOffset() {
	conditions := map[string][]string{
		"limit":  {"3"},
		"offset": {"3"},
	}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{
		DefaultOrder: "id asc",
	}

	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Equal(3, len(users))
	suite.Equal(uint(4), users[0].ID)
	suite.Equal(uint(5), users[1].ID)
	suite.Equal(uint(6), users[2].ID)
}

// Test Max Limit Enforcement
func (suite *PaginationTestSuite) TestMaxLimitEnforcement() {
	conditions := map[string][]string{
		"limit": {"500"}, // Exceeds max limit
	}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{
		MaxLimit: 50,
	}

	pg := pagination.NewPagination(conditions, filterDef, options)

	suite.Equal(50, pg.Limit, "Limit should be capped at MaxLimit")
}

// Test String Filter - Equals
func (suite *PaginationTestSuite) TestStringFilterEquals() {
	conditions := map[string][]string{
		"status": {"active"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field: "status",
			Type:  pagination.FilterTypeString,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Equal("active", user.Status)
	}
}

// Test String Filter - Like
func (suite *PaginationTestSuite) TestStringFilterLike() {
	conditions := map[string][]string{
		"name": {"like:John"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field: "name",
			Type:  pagination.FilterTypeString,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Contains(user.Name, "John")
	}
}

// Test String Filter - IN
func (suite *PaginationTestSuite) TestStringFilterIn() {
	conditions := map[string][]string{
		"status": {"in:active,pending"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field: "status",
			Type:  pagination.FilterTypeString,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Contains([]string{"active", "pending"}, user.Status)
	}
}

// Test Multi-Field Search (OR condition)
func (suite *PaginationTestSuite) TestMultiFieldSearch() {
	conditions := map[string][]string{
		"search": {"like:john"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchFields: []string{"name", "email"},
			Type:         pagination.FilterTypeString,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
}

// Test Number Filter - Greater Than
func (suite *PaginationTestSuite) TestNumberFilterGt() {
	conditions := map[string][]string{
		"age": {"gt:30"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("age", pagination.FilterConfig{
			Field: "age",
			Type:  pagination.FilterTypeNumber,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Greater(user.Age, 30)
	}
}

// Test Number Filter - Between
func (suite *PaginationTestSuite) TestNumberFilterBetween() {
	conditions := map[string][]string{
		"age": {"between:25,30"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("age", pagination.FilterConfig{
			Field: "age",
			Type:  pagination.FilterTypeNumber,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.GreaterOrEqual(user.Age, 25)
		suite.LessOrEqual(user.Age, 30)
	}
}

// Test Boolean Filter
func (suite *PaginationTestSuite) TestBooleanFilter() {
	conditions := map[string][]string{
		"is_active": {"true"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("is_active", pagination.FilterConfig{
			Field: "is_active",
			Type:  pagination.FilterTypeBool,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.True(user.IsActive)
	}
}

// Test Enum Filter
func (suite *PaginationTestSuite) TestEnumFilter() {
	conditions := map[string][]string{
		"role": {"in:admin,moderator"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("role", pagination.FilterConfig{
			Field:      "role",
			Type:       pagination.FilterTypeEnum,
			EnumValues: []string{"admin", "moderator", "user"},
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Contains([]string{"admin", "moderator"}, user.Role)
	}
}

// Test Invalid Enum Value
func (suite *PaginationTestSuite) TestInvalidEnumValue() {
	conditions := map[string][]string{
		"role": {"invalid_role"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("role", pagination.FilterConfig{
			Field:      "role",
			Type:       pagination.FilterTypeEnum,
			EnumValues: []string{"admin", "moderator", "user"},
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	// Should return all users since invalid filter is ignored
	suite.Equal(10, len(users))
}

// Test Date Filter - Equals
func (suite *PaginationTestSuite) TestDateFilterEquals() {
	today := time.Now().Format("2006-01-02")

	conditions := map[string][]string{
		"created_at": {today},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	// Users created today
}

// Test Date Filter - Between
func (suite *PaginationTestSuite) TestDateFilterBetween() {
	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	endDate := time.Now().Format("2006-01-02")

	conditions := map[string][]string{
		"created_at": {fmt.Sprintf("between:%s,%s", startDate, endDate)},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
}

// Test Sorting
func (suite *PaginationTestSuite) TestSorting() {
	conditions := map[string][]string{
		"sort": {"age desc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{
			Field:   "age",
			Allowed: true,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 1)

	// Verify descending order
	for i := 0; i < len(users)-1; i++ {
		suite.GreaterOrEqual(users[i].Age, users[i+1].Age)
	}
}

// Test Multiple Sort Fields
func (suite *PaginationTestSuite) TestMultipleSortFields() {
	conditions := map[string][]string{
		"sort": {"status asc, age desc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddSort("status", pagination.SortConfig{Field: "status", Allowed: true}).
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
}

// Test Invalid Sort Field
func (suite *PaginationTestSuite) TestInvalidSortField() {
	conditions := map[string][]string{
		"sort": {"invalid_field asc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	options := pagination.PaginationOptions{
		DefaultOrder: "id asc",
	}
	pg := pagination.NewPagination(conditions, filterDef, options)

	// Should fall back to default order
	suite.Equal("id asc", pg.Order)
}

// Test Combined Filters
func (suite *PaginationTestSuite) TestCombinedFilters() {
	conditions := map[string][]string{
		"status":    {"active"},
		"age":       {"gte:28"},
		"is_active": {"true"},
		"limit":     {"5"},
		"sort":      {"age desc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber}).
		AddFilter("is_active", pagination.FilterConfig{Field: "is_active", Type: pagination.FilterTypeBool}).
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	suite.LessOrEqual(len(users), 5)

	for _, user := range users {
		suite.Equal("active", user.Status)
		suite.GreaterOrEqual(user.Age, 28)
		suite.True(user.IsActive)
	}
}

// Test ApplyWithoutMeta
func (suite *PaginationTestSuite) TestApplyWithoutMeta() {
	conditions := map[string][]string{
		"status": {"active"},
		"limit":  {"2"},
		"offset": {"1"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var countWithMeta int64
	var countWithoutMeta int64

	// Count with meta (should apply limit/offset)
	pg.Apply(suite.db).Model(&User{}).Count(&countWithMeta)

	// Count without meta (should ignore limit/offset)
	pg.ApplyWithoutMeta(suite.db).Model(&User{}).Count(&countWithoutMeta)

	suite.Greater(countWithoutMeta, countWithMeta)
}

// Test Custom Scopes
func (suite *PaginationTestSuite) TestCustomScopes() {
	conditions := map[string][]string{}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	// Add custom scope
	customScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 28)
	}
	pg.AddCustomScope(customScope)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Greater(user.Age, 28)
	}
}

// Test Empty Conditions
func (suite *PaginationTestSuite) TestEmptyConditions() {
	conditions := map[string][]string{}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Equal(10, len(users)) // Should return all users with default limit
}

// Test Table Name Prefix
func (suite *PaginationTestSuite) TestTableNamePrefix() {
	conditions := map[string][]string{
		"status": {"active"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field:     "status",
			TableName: "users",
			Type:      pagination.FilterTypeString,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error

	suite.NoError(err)
	suite.Greater(len(users), 0)
}

func TestPaginationSuite(t *testing.T) {
	suite.Run(t, new(PaginationTestSuite))
}

// Benchmark Tests
func BenchmarkPaginationApply(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&User{})

	// Seed data
	for i := 1; i <= 100; i++ {
		db.Create(&User{
			Name:   fmt.Sprintf("User %d", i),
			Email:  fmt.Sprintf("user%d@example.com", i),
			Age:    20 + (i % 30),
			Status: []string{"active", "inactive", "pending"}[i%3],
		})
	}

	conditions := map[string][]string{
		"status": {"active"},
		"age":    {"gte:25"},
		"limit":  {"10"},
		"sort":   {"age desc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber}).
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	options := pagination.PaginationOptions{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pg := pagination.NewPagination(conditions, filterDef, options)
		var users []User
		pg.Apply(db).Find(&users)
	}
}
