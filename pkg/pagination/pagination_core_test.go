package pagination_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/PhantomX7/athleton/pkg/pagination"
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
	db  *gorm.DB
	now time.Time
}

func (suite *PaginationTestSuite) SetupSuite() {
	// Setup in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Migrate the schema
	err = db.AutoMigrate(&User{})
	suite.Require().NoError(err)

	suite.db = db
	suite.now = time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)

	// Seed test data
	suite.seedData()
}

func (suite *PaginationTestSuite) seedData() {
	now := suite.now
	users := []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Age: 25, Status: "active", IsActive: true, Role: "admin", CreatedAt: now.AddDate(0, 0, -10)},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Age: 30, Status: "active", IsActive: true, Role: "user", CreatedAt: now.AddDate(0, 0, -9)},
		{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Age: 35, Status: "inactive", IsActive: false, Role: "user", CreatedAt: now.AddDate(0, 0, -8)},
		{ID: 4, Name: "Alice Brown", Email: "alice@example.com", Age: 28, Status: "active", IsActive: true, Role: "moderator", CreatedAt: now.AddDate(0, 0, -7)},
		{ID: 5, Name: "Charlie Wilson", Email: "charlie@example.com", Age: 32, Status: "pending", IsActive: true, Role: "user", CreatedAt: now.AddDate(0, 0, -6)},
		{ID: 6, Name: "Diana Davis", Email: "diana@example.com", Age: 27, Status: "active", IsActive: true, Role: "user", CreatedAt: now.AddDate(0, 0, -5)},
		{ID: 7, Name: "Eve Miller", Email: "eve@example.com", Age: 29, Status: "inactive", IsActive: false, Role: "user", CreatedAt: now.AddDate(0, 0, -4)},
		{ID: 8, Name: "Frank Moore", Email: "frank@example.com", Age: 31, Status: "active", IsActive: true, Role: "user", CreatedAt: now.AddDate(0, 0, -3)},
		{ID: 9, Name: "Grace Taylor", Email: "grace@example.com", Age: 26, Status: "active", IsActive: true, Role: "admin", CreatedAt: now.AddDate(0, 0, -2)},
		{ID: 10, Name: "Henry Anderson", Email: "henry@example.com", Age: 33, Status: "pending", IsActive: true, Role: "user", CreatedAt: now.AddDate(0, 0, -1)},
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
	suite.Len(users, 2)
	for _, user := range users {
		joined := strings.ToLower(user.Name + " " + user.Email)
		suite.Contains(joined, "john")
	}
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
	// Invalid enum value must NOT widen the result set. We emit a no-match
	// scope so the response is empty rather than leaking every row.
	suite.Equal(0, len(users))
}

// Test Date Filter - Between
func (suite *PaginationTestSuite) TestDateFilterBetween() {
	startDate := suite.now.AddDate(0, 0, -7).Format("2006-01-02")
	endDate := suite.now.Format("2006-01-02")

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
	suite.Len(users, 7)
	for _, user := range users {
		suite.False(user.CreatedAt.Before(suite.now.AddDate(0, 0, -7)))
		suite.False(user.CreatedAt.After(suite.now))
	}
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
	suite.Greater(len(users), 1)
	for i := 0; i < len(users)-1; i++ {
		current := users[i]
		next := users[i+1]
		if current.Status == next.Status {
			suite.GreaterOrEqual(current.Age, next.Age)
			continue
		}
		suite.LessOrEqual(current.Status, next.Status)
	}
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

// ===============================================================================
// ADDITIONAL COMPREHENSIVE TESTS TO INCREASE COVERAGE
// ===============================================================================

// Test FilterConfig Methods
func (suite *PaginationTestSuite) TestFilterConfigGetAllowedOperators() {
	// Test with custom operators
	config := pagination.FilterConfig{
		Type:      pagination.FilterTypeString,
		Operators: []pagination.FilterOperator{pagination.OperatorEquals, pagination.OperatorLike},
	}
	operators := config.GetAllowedOperators()
	suite.Equal(2, len(operators))
	suite.Contains(operators, pagination.OperatorEquals)
	suite.Contains(operators, pagination.OperatorLike)

	// Test with default operators
	config2 := pagination.FilterConfig{
		Type: pagination.FilterTypeString,
	}
	operators2 := config2.GetAllowedOperators()
	suite.Greater(len(operators2), 0)
	suite.Contains(operators2, pagination.OperatorEquals)
	suite.Contains(operators2, pagination.OperatorLike)

	operators2[0] = pagination.OperatorBetween
	fresh := config2.GetAllowedOperators()
	suite.Contains(fresh, pagination.OperatorEquals)
	suite.NotEqual(pagination.OperatorBetween, fresh[0], "returned slice must be cloned")
}

func (suite *PaginationTestSuite) TestFilterConfigGetFields() {
	// Test with SearchFields
	config := pagination.FilterConfig{
		Field:        "name",
		SearchFields: []string{"name", "email"},
	}
	fields := config.GetFields()
	suite.Equal(2, len(fields))
	suite.Contains(fields, "name")
	suite.Contains(fields, "email")

	// Test with Field only
	config2 := pagination.FilterConfig{
		Field: "name",
	}
	fields2 := config2.GetFields()
	suite.Equal(1, len(fields2))
	suite.Contains(fields2, "name")

	// Test with TableName prefix
	config3 := pagination.FilterConfig{
		Field:     "name",
		TableName: "users",
	}
	fields3 := config3.GetFields()
	suite.Equal(1, len(fields3))
	suite.Contains(fields3, "users.name")

	// Test with TableName and SearchFields
	config4 := pagination.FilterConfig{
		SearchFields: []string{"name", "email"},
		TableName:    "users",
	}
	fields4 := config4.GetFields()
	suite.Equal(2, len(fields4))
	suite.Contains(fields4, "users.name")
	suite.Contains(fields4, "users.email")

	// Test with already qualified field names
	config5 := pagination.FilterConfig{
		SearchFields: []string{"users.name", "profiles.email"},
		TableName:    "users",
	}
	fields5 := config5.GetFields()
	suite.Equal(2, len(fields5))
	suite.Contains(fields5, "users.name")
	suite.Contains(fields5, "profiles.email")
}

// Test FilterDefinition Methods
func (suite *PaginationTestSuite) TestFilterDefinition() {
	fd := pagination.NewFilterDefinition()
	suite.NotNil(fd)

	// Test AddFilter chainable
	result := fd.AddFilter("name", pagination.FilterConfig{
		Field: "name",
		Type:  pagination.FilterTypeString,
	})
	suite.Equal(fd, result) // Should return the same instance for chaining

	// Test AddSort chainable
	result2 := fd.AddSort("age", pagination.SortConfig{
		Field:   "age",
		Allowed: true,
	})
	suite.Equal(fd, result2) // Should return the same instance for chaining

	// Verify functionality by testing the filters work
	conditions := map[string][]string{
		"name": {"John"},
		"sort": {"age desc"},
	}
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, fd, options)
	suite.NotNil(pg) // Should work without errors
}

// Test PaginationOptions
func (suite *PaginationTestSuite) TestDefaultPaginationOptions() {
	options := pagination.DefaultPaginationOptions()
	suite.Equal(20, options.DefaultLimit)
	suite.Equal(100, options.MaxLimit)
	suite.Equal("id desc", options.DefaultOrder)
	suite.NotNil(options.Timezone)
	suite.Equal("Asia/Jakarta", options.Timezone.String())
}

// Test Helper Functions
func (suite *PaginationTestSuite) TestParseLimit() {
	// Test valid limit
	conditions := map[string][]string{"limit": {"50"}}
	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100}
	pg := pagination.NewPagination(conditions, filterDef, options)
	suite.Equal(50, pg.Limit)

	// Test limit exceeding max
	conditions2 := map[string][]string{"limit": {"150"}}
	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	suite.Equal(100, pg2.Limit) // Should be capped at MaxLimit

	// Test invalid limit
	conditions3 := map[string][]string{"limit": {"invalid"}}
	pg3 := pagination.NewPagination(conditions3, filterDef, options)
	suite.Equal(20, pg3.Limit) // Should use default

	// Test negative limit
	conditions4 := map[string][]string{"limit": {"-5"}}
	pg4 := pagination.NewPagination(conditions4, filterDef, options)
	suite.Equal(20, pg4.Limit) // Should use default

	// Test zero limit
	conditions5 := map[string][]string{"limit": {"0"}}
	pg5 := pagination.NewPagination(conditions5, filterDef, options)
	suite.Equal(20, pg5.Limit) // Should use default

	// Test no limit provided
	conditions6 := map[string][]string{}
	pg6 := pagination.NewPagination(conditions6, filterDef, options)
	suite.Equal(20, pg6.Limit) // Should use default
}

func (suite *PaginationTestSuite) TestParseOffset() {
	// Test valid offset
	conditions := map[string][]string{"offset": {"5"}}
	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)
	suite.Equal(5, pg.Offset)

	// Test zero offset
	conditions2 := map[string][]string{"offset": {"0"}}
	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	suite.Equal(0, pg2.Offset)

	// Test invalid offset
	conditions3 := map[string][]string{"offset": {"invalid"}}
	pg3 := pagination.NewPagination(conditions3, filterDef, options)
	suite.Equal(0, pg3.Offset) // Should use default

	// Test negative offset
	conditions4 := map[string][]string{"offset": {"-5"}}
	pg4 := pagination.NewPagination(conditions4, filterDef, options)
	suite.Equal(0, pg4.Offset) // Should use default

	// Test no offset provided
	conditions5 := map[string][]string{}
	pg5 := pagination.NewPagination(conditions5, filterDef, options)
	suite.Equal(0, pg5.Offset) // Should use default
}

func (suite *PaginationTestSuite) TestParseOrder() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true})

	// Test valid order
	conditions := map[string][]string{"sort": {"age desc"}}
	options := pagination.PaginationOptions{DefaultOrder: "id asc"}
	pg := pagination.NewPagination(conditions, filterDef, options)
	suite.Equal("age desc", pg.Order)

	// Test invalid order (should use default)
	conditions2 := map[string][]string{"sort": {"invalid_field desc"}}
	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	suite.Equal("id asc", pg2.Order) // Should use default

	// Test no order provided
	conditions3 := map[string][]string{}
	pg3 := pagination.NewPagination(conditions3, filterDef, options)
	suite.Equal("id asc", pg3.Order) // Should use default
}

func (suite *PaginationTestSuite) TestValidateOrder() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true, TableName: "users"}).
		AddSort("email", pagination.SortConfig{Field: "email", Allowed: false}) // Not allowed

	// Test valid single field
	conditions := map[string][]string{"sort": {"age desc"}}
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)
	suite.Equal("age desc", pg.Order)

	// Test valid multiple fields
	conditions2 := map[string][]string{"sort": {"age desc, name asc"}}
	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	suite.Equal("age desc, name asc", pg2.Order)

	// Test invalid field (not allowed)
	conditions3 := map[string][]string{"sort": {"email asc"}}
	pg3 := pagination.NewPagination(conditions3, filterDef, options)
	suite.NotEqual("email asc", pg3.Order) // Should not use invalid field

	// Test valid table prefix
	conditions4 := map[string][]string{"sort": {"users.name asc"}}
	pg4 := pagination.NewPagination(conditions4, filterDef, options)
	suite.Equal("users.name asc", pg4.Order)

	// Test invalid table prefix
	conditions5 := map[string][]string{"sort": {"profiles.name asc"}}
	pg5 := pagination.NewPagination(conditions5, filterDef, options)
	suite.NotEqual("profiles.name asc", pg5.Order) // Should not use invalid table prefix

	// Test empty order — falls back to the resolved default ("id desc"), which
	// NewPagination fills in when options.DefaultOrder is left empty.
	conditions6 := map[string][]string{"sort": {""}}
	pg6 := pagination.NewPagination(conditions6, filterDef, options)
	suite.Equal("id desc", pg6.Order)
}

// Test Edge Cases and Error Conditions
func (suite *PaginationTestSuite) TestInvalidDateFormats() {
	conditions := map[string][]string{
		"created_at": {"invalid-date"},
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
	suite.NoError(err)          // Should not error, just ignore invalid filter
	suite.Equal(10, len(users)) // Should return all users
}

func (suite *PaginationTestSuite) TestInvalidDateTimeFormats() {
	conditions := map[string][]string{
		"created_at": {"invalid-datetime"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDateTime,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)          // Should not error, just ignore invalid filter
	suite.Equal(10, len(users)) // Should return all users
}

func (suite *PaginationTestSuite) TestInvalidOperatorCombinations() {
	// Test Between operator with single value
	conditions := map[string][]string{
		"age": {"between:25"}, // Should have 2 values
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
	suite.NoError(err)          // Should ignore invalid filter
	suite.Equal(10, len(users)) // Should return all users (invalid filter ignored)

	// Test In operator with no values
	conditions2 := map[string][]string{
		"status": {"in:"}, // Should have at least 1 value
	}

	filterDef2 := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field: "status",
			Type:  pagination.FilterTypeString,
		})

	pg2 := pagination.NewPagination(conditions2, filterDef2, options)
	var users2 []User
	err2 := pg2.Apply(suite.db).Find(&users2).Error
	suite.NoError(err2)
	// The second test case actually behaves differently - it returns 0 users
	// This is because the filter is invalid but the behavior depends on the specific implementation
	suite.Equal(0, len(users2)) // Should return no users due to invalid filter
}

func (suite *PaginationTestSuite) TestCustomOperators() {
	// Test filter with custom operators
	conditions := map[string][]string{
		"status": {"like:active"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{
			Field:     "status",
			Type:      pagination.FilterTypeString,
			Operators: []pagination.FilterOperator{pagination.OperatorLike}, // Only allow LIKE
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.Greater(len(users), 0)

	// Test with disallowed operator
	conditions2 := map[string][]string{
		"status": {"gt:active"}, // GT not allowed for strings
	}

	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	var users2 []User
	err2 := pg2.Apply(suite.db).Find(&users2).Error
	suite.NoError(err2)
	suite.Equal(10, len(users2)) // Should return all users (filter ignored)
}

func (suite *PaginationTestSuite) TestBooleanFilterEdgeCases() {
	// Boolean parsing now follows strconv.ParseBool semantics:
	// "1","t","T","TRUE","true","True"  -> true
	// "0","f","F","FALSE","false","False" -> false
	// Anything else -> filter is dropped (returns all rows unfiltered)
	type tc struct {
		value      string
		valid      bool
		expectBool bool
	}
	cases := []tc{
		{"true", true, true},
		{"false", true, false},
		{"TRUE", true, true},
		{"FALSE", true, false},
		{"True", true, true},
		{"False", true, false},
		{"1", true, true},
		{"0", true, false},
		{"t", true, true},
		{"f", true, false},
		{"yes", false, false},
		{"no", false, false},
		{"banana", false, false},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("is_active", pagination.FilterConfig{
			Field: "is_active",
			Type:  pagination.FilterTypeBool,
		})

	options := pagination.PaginationOptions{}

	for _, c := range cases {
		conditions := map[string][]string{"is_active": {c.value}}
		pg := pagination.NewPagination(conditions, filterDef, options)

		var users []User
		err := pg.Apply(suite.db).Find(&users).Error
		suite.NoError(err, "value=%s", c.value)

		if c.valid {
			suite.Greater(len(users), 0, "value=%s should match at least one row", c.value)
			for _, u := range users {
				suite.Equal(c.expectBool, u.IsActive, "value=%s", c.value)
			}
		} else {
			// Invalid bool literal: filter is dropped, so every row is returned.
			suite.Equal(10, len(users), "invalid bool %q must drop the filter, not coerce to false", c.value)
		}
	}
}

func (suite *PaginationTestSuite) TestIDFilterType() {
	conditions := map[string][]string{
		"id": {"1"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{
			Field: "id",
			Type:  pagination.FilterTypeID,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.Equal(1, len(users))
	suite.Equal(uint(1), users[0].ID)
}

func (suite *PaginationTestSuite) TestIDFilterIn() {
	conditions := map[string][]string{
		"id": {"in:1,2,3"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{
			Field: "id",
			Type:  pagination.FilterTypeID,
		})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.Equal(3, len(users))
}

func (suite *PaginationTestSuite) TestStringFilterNotEquals() {
	conditions := map[string][]string{
		"status": {"neq:inactive"},
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
		suite.NotEqual("inactive", user.Status)
	}
}

func (suite *PaginationTestSuite) TestStringFilterNotIn() {
	conditions := map[string][]string{
		"status": {"not_in:inactive"},
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
		suite.NotEqual("inactive", user.Status)
	}
}

func (suite *PaginationTestSuite) TestNumberFilterNotEquals() {
	conditions := map[string][]string{
		"age": {"neq:30"},
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
		suite.NotEqual(30, user.Age)
	}
}

func (suite *PaginationTestSuite) TestNumberFilterNotIn() {
	conditions := map[string][]string{
		"age": {"not_in:30,35"},
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
		suite.NotEqual(30, user.Age)
		suite.NotEqual(35, user.Age)
	}
}

func (suite *PaginationTestSuite) TestNumberFilterLt() {
	conditions := map[string][]string{
		"age": {"lt:30"},
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
		suite.Less(user.Age, 30)
	}
}

func (suite *PaginationTestSuite) TestNumberFilterLte() {
	conditions := map[string][]string{
		"age": {"lte:30"},
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
		suite.LessOrEqual(user.Age, 30)
	}
}

func (suite *PaginationTestSuite) TestEnumFilterEquals() {
	conditions := map[string][]string{
		"role": {"admin"},
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
		suite.Equal("admin", user.Role)
	}
}

func (suite *PaginationTestSuite) TestEnumFilterWithInvalidValues() {
	// Mixed valid + invalid enum values: the invalid ones are dropped
	// individually and the filter is applied on the remaining valid set.
	// Previously this returned ALL rows on a single typo.
	conditions := map[string][]string{
		"role": {"in:admin,invalid_role,user"},
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
	for _, u := range users {
		suite.Contains([]string{"admin", "user"}, u.Role)
	}
}

func (suite *PaginationTestSuite) TestMultiFieldSearchWithDifferentOperators() {
	// Test multi-field search with equals
	conditions := map[string][]string{
		"search": {"john@example.com"},
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

	// Test multi-field search with IN
	conditions2 := map[string][]string{
		"search": {"in:john@example.com,jane@example.com"},
	}

	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	var users2 []User
	err2 := pg2.Apply(suite.db).Find(&users2).Error
	suite.NoError(err2)
	suite.Greater(len(users2), 0)
}

func (suite *PaginationTestSuite) TestComplexFilterCombinations() {
	// Test multiple filters of different types
	conditions := map[string][]string{
		"status":    {"in:active,pending"},
		"age":       {"between:25,32"},
		"is_active": {"true"},
		"role":      {"neq:moderator"},
		"name":      {"like:john"},
		"limit":     {"3"},
		"offset":    {"0"},
		"sort":      {"age desc, name asc"},
	}

	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber}).
		AddFilter("is_active", pagination.FilterConfig{Field: "is_active", Type: pagination.FilterTypeBool}).
		AddFilter("role", pagination.FilterConfig{Field: "role", Type: pagination.FilterTypeString}).
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString}).
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true})

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.LessOrEqual(len(users), 3) // Should respect limit

	for _, user := range users {
		suite.Contains([]string{"active", "pending"}, user.Status)
		suite.GreaterOrEqual(user.Age, 25)
		suite.LessOrEqual(user.Age, 32)
		suite.True(user.IsActive)
		suite.NotEqual("moderator", user.Role)
		suite.Contains(strings.ToLower(user.Name), "john")
	}
}

func (suite *PaginationTestSuite) TestMultipleCustomScopes() {
	conditions := map[string][]string{}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	// Add multiple custom scopes
	scope1 := func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 25)
	}
	scope2 := func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", "active")
	}
	pg.AddCustomScope(scope1, scope2)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.Greater(len(users), 0)
	for _, user := range users {
		suite.Greater(user.Age, 25)
		suite.Equal("active", user.Status)
	}
}

func (suite *PaginationTestSuite) TestNonExistentFilter() {
	conditions := map[string][]string{
		"non_existent": {"value"},
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
	suite.Equal(10, len(users)) // Should return all users (non-existent filter ignored)
}

func (suite *PaginationTestSuite) TestFilterWithEmptyConfig() {
	conditions := map[string][]string{
		"status": {"active"},
	}

	filterDef := pagination.NewFilterDefinition()
	// Don't add any filter config

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)

	var users []User
	err := pg.Apply(suite.db).Find(&users).Error
	suite.NoError(err)
	suite.Equal(10, len(users)) // Should return all users (no filter config)
}

func (suite *PaginationTestSuite) TestZeroValuesInOptions() {
	conditions := map[string][]string{}

	filterDef := pagination.NewFilterDefinition()
	options := pagination.PaginationOptions{
		DefaultLimit: 0,   // Should be set to 20
		MaxLimit:     0,   // Should be set to 100
		DefaultOrder: "",  // Should be set to "id desc"
		Timezone:     nil, // Should be set to Asia/Jakarta
	}

	pg := pagination.NewPagination(conditions, filterDef, options)

	suite.Equal(20, pg.Limit)        // Should use default
	suite.Equal("id desc", pg.Order) // Should use default
	// Timezone is applied internally but not accessible for direct testing
}

func (suite *PaginationTestSuite) TestBooleanFilterInvalidOperator() {
	conditions := map[string][]string{
		"is_active": {"like:true"}, // LIKE not allowed for boolean
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
	suite.Equal(10, len(users)) // Should return all users (invalid operator ignored)
}

func (suite *PaginationTestSuite) TestSortWithTablePrefixValidation() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("name", pagination.SortConfig{
			Field:     "name",
			TableName: "users",
			Allowed:   true,
		})

	// Test correct table prefix
	conditions := map[string][]string{
		"sort": {"users.name asc"},
	}

	options := pagination.PaginationOptions{}
	pg := pagination.NewPagination(conditions, filterDef, options)
	suite.Equal("users.name asc", pg.Order)

	// Test incorrect table prefix
	conditions2 := map[string][]string{
		"sort": {"profiles.name asc"},
	}

	pg2 := pagination.NewPagination(conditions2, filterDef, options)
	suite.NotEqual("profiles.name asc", pg2.Order) // Should use default order
}

// ===============================================================================
// SECURITY & CORRECTNESS TESTS (regression coverage for the audit fixes)
// ===============================================================================

// TestSortSQLInjection — a malicious sort string must NOT be passed through
// to GORM's Order() verbatim. Before the fix, parseOrder validated only the
// field name and reflected the rest of the user input directly.
func (suite *PaginationTestSuite) TestSortSQLInjection() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true})

	options := pagination.PaginationOptions{DefaultOrder: "id desc"}

	cases := []string{
		"id desc; DROP TABLE users--",
		"id desc/*comment*/",
		"id desc UNION SELECT 1",
		"id desc, (SELECT 1)",
		"id sideways", // invalid direction
		"id; --",
		"1; DELETE FROM users", // numeric literal, not an identifier
		"users.id desc",        // table prefix not declared on this SortConfig
	}

	for _, raw := range cases {
		conditions := map[string][]string{"sort": {raw}}
		pg := pagination.NewPagination(conditions, filterDef, options)

		suite.Equal("id desc", pg.Order, "must reject malicious sort %q", raw)

		// And the query itself must still execute — proving the fallback was
		// applied at parse time, not surfaced to the DB.
		var users []User
		err := pg.Apply(suite.db).Find(&users).Error
		suite.NoError(err, "sort=%q", raw)
		suite.Equal(10, len(users), "sort=%q", raw)
	}
}

// TestSortDirectionWhitelist — only asc/desc are accepted, anything else
// falls back to the default order.
func (suite *PaginationTestSuite) TestSortDirectionWhitelist() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	options := pagination.PaginationOptions{DefaultOrder: "id desc"}

	for _, dir := range []string{"asc", "ASC", "Asc", "desc", "DESC", "Desc"} {
		pg := pagination.NewPagination(
			map[string][]string{"sort": {"age " + dir}},
			filterDef, options,
		)
		expected := "age " + strings.ToLower(dir)
		suite.Equal(expected, pg.Order, "dir=%s", dir)
	}

	for _, dir := range []string{"random", "ascending", "descending", "1", ""} {
		pg := pagination.NewPagination(
			map[string][]string{"sort": {"age " + dir}},
			filterDef, options,
		)
		if dir == "" {
			// "age " trims to "age" — that's a valid 1-token sort, defaults asc.
			suite.Equal("age asc", pg.Order)
		} else {
			suite.Equal("id desc", pg.Order, "dir=%q must be rejected", dir)
		}
	}
}

// TestLikeWildcardEscaping — '%' and '_' in user input must be treated as
// literals, not wildcards. Before the fix `?name=like:%` matched everything.
func (suite *PaginationTestSuite) TestLikeWildcardEscaping() {
	// Seed two extra rows: one containing a literal '%' and one a literal '_'.
	suite.db.Create(&User{ID: 100, Name: "Discount 50% Off", Email: "discount@x.com", Status: "active", IsActive: true, Role: "user"})
	suite.db.Create(&User{ID: 101, Name: "Snake_Case Name", Email: "snake@x.com", Status: "active", IsActive: true, Role: "user"})
	defer suite.db.Unscoped().Delete(&User{}, []uint{100, 101})

	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString})

	options := pagination.PaginationOptions{}

	// '%' in input must match only rows containing literal '%'
	pg := pagination.NewPagination(
		map[string][]string{"name": {"like:%"}},
		filterDef, options,
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(1, len(users), "like:%% must match only the literal-%% row, not all rows")
	suite.Equal(uint(100), users[0].ID)

	// '_' in input must match only rows containing literal '_', not any single char
	pg2 := pagination.NewPagination(
		map[string][]string{"name": {"like:_"}},
		filterDef, options,
	)
	var users2 []User
	suite.NoError(pg2.Apply(suite.db).Find(&users2).Error)
	suite.Equal(1, len(users2), "like:_ must match only the literal-_ row")
	suite.Equal(uint(101), users2[0].ID)
}

// TestLikeWildcardEscapingMultiField — same guarantee on the OR variant.
func (suite *PaginationTestSuite) TestLikeWildcardEscapingMultiField() {
	suite.db.Create(&User{ID: 102, Name: "100% Cotton", Email: "cotton@x.com", Status: "active", IsActive: true, Role: "user"})
	defer suite.db.Unscoped().Delete(&User{}, 102)

	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchFields: []string{"name", "email"},
			Type:         pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"search": {"like:%"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(1, len(users))
	suite.Equal(uint(102), users[0].ID)
}

// TestColonInValueIsNotMisparsedAsOperator — values that legitimately contain
// a colon (URLs, namespaced keys) must not have the prefix stripped as if it
// were an operator.
func (suite *PaginationTestSuite) TestColonInValueIsNotMisparsedAsOperator() {
	suite.db.Create(&User{ID: 103, Name: "url", Email: "https://example.com", Status: "active", IsActive: true, Role: "user"})
	defer suite.db.Unscoped().Delete(&User{}, 103)

	filterDef := pagination.NewFilterDefinition().
		AddFilter("email", pagination.FilterConfig{Field: "email", Type: pagination.FilterTypeString})

	// "https://example.com" was previously parsed as operator="https",
	// silently dropping the filter. Now it's an equality match on the full value.
	pg := pagination.NewPagination(
		map[string][]string{"email": {"https://example.com"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(1, len(users))
	suite.Equal(uint(103), users[0].ID)
}

// TestMultiValueConditionsApplyAsAnd — repeated query keys (?role=admin&role=user)
// must each contribute a condition rather than only the first being honored.
func (suite *PaginationTestSuite) TestMultiValueConditionsApplyAsAnd() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("role", pagination.FilterConfig{Field: "role", Type: pagination.FilterTypeString}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber})

	// age >= 25 AND age <= 30 expressed as repeated keys.
	pg := pagination.NewPagination(
		map[string][]string{"age": {"gte:25", "lte:30"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Greater(len(users), 0)
	for _, u := range users {
		suite.GreaterOrEqual(u.Age, 25)
		suite.LessOrEqual(u.Age, 30)
	}
}

// TestNilSafety — nil filter definition and nil conditions must not panic.
func (suite *PaginationTestSuite) TestNilSafety() {
	suite.NotPanics(func() {
		pg := pagination.NewPagination(nil, nil, pagination.PaginationOptions{})
		var users []User
		_ = pg.Apply(suite.db).Find(&users).Error
	})
}

// TestGetTotalPages — basic correctness, edge cases, and that we don't return
// 1 when the result set is empty (a common UI bug).
func (suite *PaginationTestSuite) TestGetTotalPages() {
	pg := pagination.NewPagination(
		map[string][]string{"limit": {"10"}},
		pagination.NewFilterDefinition(),
		pagination.PaginationOptions{},
	)
	suite.Equal(0, pg.GetTotalPages(0), "empty result set has zero pages")
	suite.Equal(1, pg.GetTotalPages(1))
	suite.Equal(1, pg.GetTotalPages(10))
	suite.Equal(2, pg.GetTotalPages(11))
	suite.Equal(10, pg.GetTotalPages(100))
}

// TestGetPage — verifies 1-indexed pages computed from offset/limit.
func (suite *PaginationTestSuite) TestGetPage() {
	mk := func(limit, offset int) *pagination.Pagination {
		return pagination.NewPagination(
			map[string][]string{
				"limit":  {strconv.Itoa(limit)},
				"offset": {strconv.Itoa(offset)},
			},
			pagination.NewFilterDefinition(),
			pagination.PaginationOptions{},
		)
	}
	suite.Equal(1, mk(10, 0).GetPage())
	suite.Equal(2, mk(10, 10).GetPage())
	suite.Equal(3, mk(10, 20).GetPage())
}

// TestSpecialCharactersInFilterApostrophe — apostrophes (single quotes) must
// be safely parameterized; this is a smoke test against accidental
// concatenation regressions.
func (suite *PaginationTestSuite) TestSpecialCharactersInFilterApostrophe() {
	suite.db.Create(&User{ID: 104, Name: "O'Reilly", Email: "oreilly@x.com", Status: "active", IsActive: true, Role: "user"})
	defer suite.db.Unscoped().Delete(&User{}, 104)

	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString})

	pg := pagination.NewPagination(
		map[string][]string{"name": {"like:O'Reilly"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(1, len(users))
}

// TestTimezoneFallback — when no Timezone is supplied and tzdata is present
// (we embed time/tzdata), Asia/Jakarta resolves; otherwise UTC. Either way it
// must be non-nil so date parsing does not panic.
func (suite *PaginationTestSuite) TestTimezoneFallback() {
	opts := pagination.DefaultPaginationOptions()
	suite.NotNil(opts.Timezone, "DefaultPaginationOptions must always return a non-nil Timezone")

	pg := pagination.NewPagination(
		map[string][]string{"created_at": {"2026-01-01"}},
		pagination.NewFilterDefinition().AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at", Type: pagination.FilterTypeDate,
		}),
		pagination.PaginationOptions{Timezone: nil},
	)
	suite.NotPanics(func() {
		var users []User
		_ = pg.Apply(suite.db).Find(&users).Error
	})
}

// TestEnumTypoDoesNotWidenResults — regression for the silent-widening bug.
// A single typo must not return rows the user didn't ask for.
func (suite *PaginationTestSuite) TestEnumTypoDoesNotWidenResults() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("role", pagination.FilterConfig{
			Field:      "role",
			Type:       pagination.FilterTypeEnum,
			EnumValues: []string{"admin", "moderator", "user"},
		})

	pg := pagination.NewPagination(
		map[string][]string{"role": {"typo_only"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(0, len(users), "all-invalid enum filter must yield zero rows, not the full table")
}

// TestBoolIsNotNullOperator — BOOL columns advertise IS NULL / IS NOT NULL in
// operatorsByType; before the fix buildBoolScope only handled equality so
// these silently no-opped despite passing validation.
func (suite *PaginationTestSuite) TestBoolIsNotNullOperator() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("is_active", pagination.FilterConfig{
			Field: "is_active",
			Type:  pagination.FilterTypeBool,
		})

	// IS NOT NULL — every seeded row has is_active set, so all 10 match.
	pg := pagination.NewPagination(
		map[string][]string{"is_active": {"is_not_null"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(10, len(users))

	// IS NULL — no rows have a NULL is_active, so none match.
	pg2 := pagination.NewPagination(
		map[string][]string{"is_active": {"is_null"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users2 []User
	suite.NoError(pg2.Apply(suite.db).Find(&users2).Error)
	suite.Equal(0, len(users2))
}

// TestNullOperatorWithTrailingColon — `is_null:` and `is_null` must produce
// the same FilterOperation (no spurious empty-string in Values).
func (suite *PaginationTestSuite) TestNullOperatorWithTrailingColon() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("is_active", pagination.FilterConfig{
			Field: "is_active",
			Type:  pagination.FilterTypeBool,
		})

	for _, form := range []string{"is_not_null", "is_not_null:"} {
		pg := pagination.NewPagination(
			map[string][]string{"is_active": {form}},
			filterDef, pagination.PaginationOptions{},
		)
		var users []User
		suite.NoError(pg.Apply(suite.db).Find(&users).Error, "form=%q", form)
		suite.Equal(10, len(users), "form=%q", form)
	}
}

// TestDeterministicWhereClauseOrder — building the same Pagination twice on
// the same conditions must produce byte-identical SQL. Map iteration order
// is randomised, so without sorting the keys the WHERE clause shuffles each
// run, breaking prepared-statement plan caching on Postgres.
func (suite *PaginationTestSuite) TestDeterministicWhereClauseOrder() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber}).
		AddFilter("role", pagination.FilterConfig{Field: "role", Type: pagination.FilterTypeString}).
		AddFilter("name", pagination.FilterConfig{Field: "name", Type: pagination.FilterTypeString}).
		AddFilter("email", pagination.FilterConfig{Field: "email", Type: pagination.FilterTypeString})

	conditions := map[string][]string{
		"status": {"active"},
		"age":    {"gte:25"},
		"role":   {"user"},
		"name":   {"like:a"},
		"email":  {"like:com"},
	}

	render := func() string {
		pg := pagination.NewPagination(conditions, filterDef, pagination.PaginationOptions{})
		stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
		return suite.db.Explain(stmt.SQL.String(), stmt.Vars...)
	}

	// 30 runs gives us astronomically high confidence: if WHERE-clause order
	// were random across N=5 keys, the odds of 30 matching renderings by
	// chance is roughly 1 in 120^29.
	first := render()
	for i := 0; i < 30; i++ {
		suite.Equal(first, render(), "WHERE clause order must be stable across runs (iter=%d)", i)
	}
}

// TestNumericInWithMixedValidity — invalid numeric values are dropped from
// IN lists individually; the remaining valid values still apply.
func (suite *PaginationTestSuite) TestNumericInWithMixedValidity() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber})

	pg := pagination.NewPagination(
		map[string][]string{"age": {"in:25,not_a_number,30"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Greater(len(users), 0)
	for _, u := range users {
		suite.Contains([]int{25, 30}, u.Age)
	}
}

// TestNumericIsTypedNotString — int IN lists must serialize as integers, not
// strings, otherwise Postgres rejects them with an "operator does not exist"
// error. We assert this by inspecting the bound parameters.
func (suite *PaginationTestSuite) TestNumericIsTypedNotString() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber})

	pg := pagination.NewPagination(
		map[string][]string{"age": {"in:25,30"}},
		filterDef, pagination.PaginationOptions{},
	)
	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	suite.Require().NotEmpty(stmt.Vars)

	// The IN values get bound as the last positional vars. Walk from the
	// back and assert every numeric arg is a typed int64, not a string.
	foundInt := 0
	for _, v := range stmt.Vars {
		if i, ok := v.(int64); ok {
			suite.Contains([]int64{25, 30}, i)
			foundInt++
		}
		_, isStr := v.(string)
		suite.False(isStr && (v == "25" || v == "30"), "numeric IN value %v leaked as string", v)
	}
	suite.Equal(2, foundInt, "expected both IN values to be bound as int64")
}

// TestInvalidFieldIdentifierIsRejected — even if a developer accidentally
// constructs a FilterConfig with a non-identifier Field (e.g. populated from
// a YAML file someday), the splice point refuses to build the scope.
func (suite *PaginationTestSuite) TestInvalidFieldIdentifierIsRejected() {
	bad := []string{
		"name; DROP TABLE users",
		"name OR 1=1",
		"users.name; --",
		"users..name",
		"",
		"1name",
	}

	for _, badField := range bad {
		filterDef := pagination.NewFilterDefinition().
			AddFilter("x", pagination.FilterConfig{Field: badField, Type: pagination.FilterTypeString})

		pg := pagination.NewPagination(
			map[string][]string{"x": {"value"}},
			filterDef, pagination.PaginationOptions{},
		)
		var users []User
		err := pg.Apply(suite.db).Find(&users).Error
		suite.NoError(err, "field=%q must be rejected at splice time, not crash the query", badField)
		suite.Equal(10, len(users), "field=%q must drop the filter without applying", badField)
	}
}

// TestFilterValueCountCap — a pathological IN list must be rejected wholesale
// rather than allocated and validated one-by-one. The exact cap is an
// implementation detail; we assert "much larger than any real request".
func (suite *PaginationTestSuite) TestFilterValueCountCap() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber})

	values := make([]string, 10_000)
	for i := range values {
		values[i] = strconv.Itoa(i)
	}

	pg := pagination.NewPagination(
		map[string][]string{"age": {"in:" + strings.Join(values, ",")}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	// Cap exceeded → filter is dropped, so every seeded row comes back.
	suite.Equal(10, len(users))
}

// TestSortPartsCap — similar bound on ?sort=.
func (suite *PaginationTestSuite) TestSortPartsCap() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	parts := make([]string, 1_000)
	for i := range parts {
		parts[i] = "age asc"
	}
	pg := pagination.NewPagination(
		map[string][]string{"sort": {strings.Join(parts, ",")}},
		filterDef, pagination.PaginationOptions{DefaultOrder: "id desc"},
	)
	suite.Equal("id desc", pg.Order, "oversized sort must fall back to default")
}

// TestGetConditionsReturnsClone — mutating the returned map must not leak
// into the pagination's internal state.
func (suite *PaginationTestSuite) TestGetConditionsReturnsClone() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("status", pagination.FilterConfig{Field: "status", Type: pagination.FilterTypeString})

	pg := pagination.NewPagination(
		map[string][]string{"status": {"active"}},
		filterDef, pagination.PaginationOptions{},
	)

	got := pg.GetConditions()
	got["status"][0] = "inactive"
	got["injected"] = []string{"evil"}

	// A fresh read must still show the original values.
	fresh := pg.GetConditions()
	suite.Equal([]string{"active"}, fresh["status"])
	_, leaked := fresh["injected"]
	suite.False(leaked, "mutations on the returned map must not persist")
}

// TestAddFilterDropsInvalidIdentifier — registration-time validation: a
// FilterConfig with an unsafe Field never enters the registry, so the
// matching query param is treated as unknown.
func (suite *PaginationTestSuite) TestAddFilterDropsInvalidIdentifier() {
	cases := []pagination.FilterConfig{
		{Field: "name; DROP TABLE x", Type: pagination.FilterTypeString},
		{Field: "name OR 1=1", Type: pagination.FilterTypeString},
		{Field: "", Type: pagination.FilterTypeString},
		{Field: "1starts_with_digit", Type: pagination.FilterTypeString},
		{Field: "ok", TableName: "bad table", Type: pagination.FilterTypeString},
		{SearchFields: []string{"ok", "bad;field"}, Type: pagination.FilterTypeString},
	}
	for i, cfg := range cases {
		fd := pagination.NewFilterDefinition().AddFilter("x", cfg)
		pg := pagination.NewPagination(
			map[string][]string{"x": {"anything"}},
			fd, pagination.PaginationOptions{},
		)
		var users []User
		err := pg.Apply(suite.db).Find(&users).Error
		suite.NoError(err, "case %d: %+v", i, cfg)
		suite.Equal(10, len(users), "case %d: unregistered filter must no-op", i)
	}
}

// TestAddSortDropsInvalidIdentifier — same guarantee for SortConfig.
func (suite *PaginationTestSuite) TestAddSortDropsInvalidIdentifier() {
	fd := pagination.NewFilterDefinition().
		AddSort("bad name", pagination.SortConfig{Field: "age", Allowed: true}).
		AddSort("ok", pagination.SortConfig{Field: "age; --", Allowed: true}).
		AddSort("ok2", pagination.SortConfig{Field: "age", TableName: "users;--", Allowed: true})

	for _, sortVal := range []string{"bad name asc", "ok asc", "ok2 asc"} {
		pg := pagination.NewPagination(
			map[string][]string{"sort": {sortVal}},
			fd, pagination.PaginationOptions{DefaultOrder: "id desc"},
		)
		suite.Equal("id desc", pg.Order, "unregistered sort %q must fall back to default", sortVal)
	}
}

// TestIsIdentHandRolled — matches the previous regex behaviour on a grid of
// tricky inputs so the hand-coded check can't silently drift.
func TestIsIdentHandRolled(t *testing.T) {
	// We test through AddSort which is the user-facing gate. A registered
	// sort key must be usable in ?sort=; a rejected one must not.
	cases := []struct {
		name  string
		valid bool
	}{
		{"a", true},
		{"_a", true},
		{"a1", true},
		{"A_B_C", true},
		{"abc_123", true},
		{"", false},
		{"1a", false},
		{"a b", false},
		{"a-b", false},
		{"a.b", false}, // dot not allowed in bare ident
		{"a;b", false},
		{"a'", false},
		{"a\x00", false},
	}
	for _, c := range cases {
		fd := pagination.NewFilterDefinition().
			AddSort(c.name, pagination.SortConfig{Field: "age", Allowed: true})

		// If the name was accepted, a corresponding ?sort= must be honored.
		// Build a query with a safe field name to probe registration.
		fdProbe := pagination.NewFilterDefinition().
			AddSort(c.name, pagination.SortConfig{Field: "age", Allowed: true})
		_ = fd

		pg := pagination.NewPagination(
			map[string][]string{"sort": {c.name + " asc"}},
			fdProbe, pagination.PaginationOptions{DefaultOrder: "id desc"},
		)
		if c.valid {
			if pg.Order == "id desc" {
				t.Errorf("isIdent(%q) should have accepted — sort was rejected", c.name)
			}
		} else {
			if pg.Order != "id desc" {
				t.Errorf("isIdent(%q) should have rejected — got order=%q", c.name, pg.Order)
			}
		}
	}
}

// TestIDFilterRejectsFloat — FilterTypeID must not accept a decimal value.
// `id=1.5` against an integer column is nonsensical and, more importantly,
// above 2^53 a float fallback silently collides distinct IDs onto the same
// rounded value, matching the wrong row.
func (suite *PaginationTestSuite) TestIDFilterRejectsFloat() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{Field: "id", Type: pagination.FilterTypeID})

	pg := pagination.NewPagination(
		map[string][]string{"id": {"1.5"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	// Float value must be rejected → filter dropped → all rows returned.
	suite.Equal(10, len(users))

	// Same filter with an integer works.
	pg2 := pagination.NewPagination(
		map[string][]string{"id": {"3"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users2 []User
	suite.NoError(pg2.Apply(suite.db).Find(&users2).Error)
	suite.Equal(1, len(users2))
	suite.Equal(uint(3), users2[0].ID)
}

// TestIDFilterBindsAsInt64 — bound parameters for FilterTypeID must be
// int64, not string or float64. Matters on strictly-typed drivers (Postgres).
func (suite *PaginationTestSuite) TestIDFilterBindsAsInt64() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{Field: "id", Type: pagination.FilterTypeID})

	pg := pagination.NewPagination(
		map[string][]string{"id": {"in:1,2,3"}},
		filterDef, pagination.PaginationOptions{},
	)
	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	intCount := 0
	for _, v := range stmt.Vars {
		if _, ok := v.(int64); ok {
			intCount++
		}
		if _, ok := v.(float64); ok {
			suite.Failf("ID var leaked as float64", "value=%v", v)
		}
	}
	suite.Equal(3, intCount)
}

// TestDateEqualsHalfOpen — a day-equality filter uses a half-open range so
// micro/nanosecond precision mismatches between Go and the DB cannot exclude
// rows stored in the last microsecond of the day. Smoke-test via dry-run SQL.
func (suite *PaginationTestSuite) TestDateEqualsHalfOpen() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("created_at", pagination.FilterConfig{Field: "created_at", Type: pagination.FilterTypeDate})

	pg := pagination.NewPagination(
		map[string][]string{"created_at": {"2026-04-15"}},
		filterDef, pagination.PaginationOptions{},
	)
	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	sql := stmt.SQL.String()
	suite.Contains(sql, ">= ?", "must use a >= lower bound")
	suite.Contains(sql, "< ?", "must use a strict < upper bound (half-open)")
	suite.NotContains(sql, "BETWEEN", "closed BETWEEN range re-introduces the precision bug")
}

// TestDefaultOrderIsValidated — a malformed PaginationOptions.DefaultOrder
// must never reach GORM's Order(). Falls back to a hardcoded safe literal.
func (suite *PaginationTestSuite) TestDefaultOrderIsValidated() {
	cases := []string{
		"id desc; DROP TABLE users",
		"(SELECT 1)",
		"id /* comment */",
		"id sideways",
		"1id asc",
		"",
	}
	for _, bad := range cases {
		pg := pagination.NewPagination(
			nil, nil,
			pagination.PaginationOptions{DefaultOrder: bad},
		)
		suite.Equal("id desc", pg.Order, "unsafe DefaultOrder %q must fall back", bad)
	}

	// A safe literal passes through unchanged.
	pg := pagination.NewPagination(
		nil, nil,
		pagination.PaginationOptions{DefaultOrder: "created_at asc"},
	)
	suite.Equal("created_at asc", pg.Order)
}

// TestMultiFieldNegationUsesAnd — "value doesn't appear in any of these
// fields" requires AND, not OR. With OR, `name != x OR email != x` is
// trivially true for essentially every row.
func (suite *PaginationTestSuite) TestMultiFieldNegationUsesAnd() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("search", pagination.FilterConfig{
			SearchFields: []string{"name", "email"},
			Type:         pagination.FilterTypeString,
		})

	// Pick a seeded row and try to exclude it by one of its fields. Before
	// the fix, OR-negation would still return that row (because the other
	// field satisfies `!= "john@example.com"`).
	pg := pagination.NewPagination(
		map[string][]string{"search": {"neq:john@example.com"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	for _, u := range users {
		suite.NotEqual("john@example.com", u.Email, "row should have been excluded")
		suite.NotEqual("john@example.com", u.Name, "row should have been excluded")
	}
	// And the SQL should join with AND.
	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	sql := stmt.SQL.String()
	suite.Contains(sql, "AND", "multi-field != must combine with AND, not OR")
}

// TestEnumCustomNotEqualsOperator — if a caller opts into NotEquals/NotIn
// via custom Operators on an enum, the scope must honor it instead of
// silently no-opping.
func (suite *PaginationTestSuite) TestEnumCustomNotEqualsOperator() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("role", pagination.FilterConfig{
			Field:      "role",
			Type:       pagination.FilterTypeEnum,
			EnumValues: []string{"admin", "moderator", "user"},
			Operators:  []pagination.FilterOperator{pagination.OperatorNotEquals, pagination.OperatorNotIn},
		})

	pg := pagination.NewPagination(
		map[string][]string{"role": {"neq:admin"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Greater(len(users), 0)
	for _, u := range users {
		suite.NotEqual("admin", u.Role)
	}

	pg2 := pagination.NewPagination(
		map[string][]string{"role": {"not_in:admin,moderator"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users2 []User
	suite.NoError(pg2.Apply(suite.db).Find(&users2).Error)
	suite.Greater(len(users2), 0)
	for _, u := range users2 {
		suite.NotContains([]string{"admin", "moderator"}, u.Role)
	}
}

// TestDateRangeIsCalendarCorrectAcrossDST — t.Add(24h) is wall-clock
// arithmetic and lands an hour off on DST-transition days. The day-equality
// filter must emit a half-open range whose upper bound is midnight of the
// next calendar day in the configured timezone, regardless of DST.
func (suite *PaginationTestSuite) TestDateRangeIsCalendarCorrectAcrossDST() {
	ny, err := time.LoadLocation("America/New_York")
	suite.Require().NoError(err)

	filterDef := pagination.NewFilterDefinition().
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at", Type: pagination.FilterTypeDate,
		})

	// 2024-03-10 is the US spring-forward day: 02:00 local → 03:00 local,
	// so the wall-clock day is only 23 hours long. Add(24h) would produce
	// 2024-03-11 01:00 instead of 2024-03-11 00:00.
	pg := pagination.NewPagination(
		map[string][]string{"created_at": {"2024-03-10"}},
		filterDef,
		pagination.PaginationOptions{Timezone: ny},
	)

	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	suite.Require().Len(stmt.Vars, 2, "expected lower and upper bound params")

	lower, ok := stmt.Vars[0].(time.Time)
	suite.Require().True(ok, "lower bound must be time.Time, got %T", stmt.Vars[0])
	upper, ok := stmt.Vars[1].(time.Time)
	suite.Require().True(ok, "upper bound must be time.Time, got %T", stmt.Vars[1])

	expectedLower := time.Date(2024, 3, 10, 0, 0, 0, 0, ny)
	expectedUpper := time.Date(2024, 3, 11, 0, 0, 0, 0, ny)
	suite.True(lower.Equal(expectedLower), "lower=%s expected=%s", lower, expectedLower)
	suite.True(upper.Equal(expectedUpper),
		"upper=%s expected=%s — Add(24h) would land at 01:00 on DST day",
		upper, expectedUpper)
}

// TestSortAliasesToConfigField — SortConfig.Field is the actual DB column;
// the sort key is what the API surface exposes. Previously the emitted
// ORDER BY used the request key verbatim, so aliasing was broken.
func (suite *PaginationTestSuite) TestSortAliasesToConfigField() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("created", pagination.SortConfig{
			Field: "created_at", Allowed: true,
		})

	pg := pagination.NewPagination(
		map[string][]string{"sort": {"created desc"}},
		filterDef, pagination.PaginationOptions{},
	)
	suite.Equal("created_at desc", pg.Order,
		"request key 'created' must resolve to cfg.Field='created_at'")

	// With a qualified table prefix the alias still resolves.
	filterDef2 := pagination.NewFilterDefinition().
		AddSort("created", pagination.SortConfig{
			Field: "created_at", TableName: "orders", Allowed: true,
		})
	pg2 := pagination.NewPagination(
		map[string][]string{"sort": {"orders.created asc"}},
		filterDef2, pagination.PaginationOptions{},
	)
	suite.Equal("orders.created_at asc", pg2.Order)
}

// TestSortKeyEqualsFieldStillWorks — common case where the request key
// matches the column name must keep working (regression guard for the
// alias change above).
func (suite *PaginationTestSuite) TestSortKeyEqualsFieldStillWorks() {
	filterDef := pagination.NewFilterDefinition().
		AddSort("age", pagination.SortConfig{Field: "age", Allowed: true})

	pg := pagination.NewPagination(
		map[string][]string{"sort": {"age desc"}},
		filterDef, pagination.PaginationOptions{},
	)
	suite.Equal("age desc", pg.Order)
}

// TestAllInvalidInReturnsZeroRows — ?id=in:abc,def against a numeric column
// used to drop the filter entirely, silently returning every row. It must
// now return zero rows, matching the enum filter's fail-closed behaviour.
func (suite *PaginationTestSuite) TestAllInvalidInReturnsZeroRows() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{Field: "id", Type: pagination.FilterTypeID}).
		AddFilter("age", pagination.FilterConfig{Field: "age", Type: pagination.FilterTypeNumber})

	cases := []struct {
		name   string
		params map[string][]string
	}{
		{"id_all_invalid", map[string][]string{"id": {"in:abc,def,ghi"}}},
		{"number_all_invalid", map[string][]string{"age": {"in:x,y,z"}}},
	}
	for _, tc := range cases {
		pg := pagination.NewPagination(tc.params, filterDef, pagination.PaginationOptions{})
		var users []User
		suite.NoError(pg.Apply(suite.db).Find(&users).Error, tc.name)
		suite.Equal(0, len(users),
			"%s: IN with all-invalid values must return zero rows, not leak all", tc.name)
	}
}

// TestAllInvalidNotInPreservesAllRows — symmetric counterpoint: "exclude
// this set" where the set is empty means "exclude nothing", so every row
// must be returned. Distinct from IN semantics above.
func (suite *PaginationTestSuite) TestAllInvalidNotInPreservesAllRows() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("id", pagination.FilterConfig{Field: "id", Type: pagination.FilterTypeID})

	pg := pagination.NewPagination(
		map[string][]string{"id": {"not_in:abc,def"}},
		filterDef, pagination.PaginationOptions{},
	)
	var users []User
	suite.NoError(pg.Apply(suite.db).Find(&users).Error)
	suite.Equal(10, len(users),
		"NOT IN with no valid exclusions must match every row")
}

// TestGetFieldsCachedAfterRegistration — regression guard for the cached
// path. Behaviour must be identical to the un-cached path: a TableName-
// prefixed filter still resolves to the right qualified column.
func (suite *PaginationTestSuite) TestGetFieldsCachedAfterRegistration() {
	filterDef := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field: "name", TableName: "users", Type: pagination.FilterTypeString,
		})

	pg := pagination.NewPagination(
		map[string][]string{"name": {"John Doe"}},
		filterDef, pagination.PaginationOptions{},
	)
	stmt := pg.Apply(suite.db.Session(&gorm.Session{DryRun: true})).Find(&[]User{}).Statement
	sql := suite.db.Explain(stmt.SQL.String(), stmt.Vars...)
	suite.Contains(sql, "users.name", "cached fields must still emit the qualified column")
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
