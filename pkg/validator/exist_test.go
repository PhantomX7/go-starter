package validator

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test model for exist validator testing
type ExistTestModel struct {
	ID    uint   `gorm:"primarykey"`
	Email string `gorm:"unique"`
	Name  string
	Age   int
}

// setupExistTestDB creates an in-memory SQLite database for exist testing
func setupExistTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto migrate the test model
	err = db.AutoMigrate(&ExistTestModel{})
	if err != nil {
		t.Fatalf("Failed to migrate test model: %v", err)
	}

	// Insert test data
	testData := []ExistTestModel{
		{ID: 1, Email: "user1@example.com", Name: "User One", Age: 25},
		{ID: 2, Email: "user2@example.com", Name: "User Two", Age: 30},
		{ID: 3, Email: "user3@example.com", Name: "User Three", Age: 35},
	}

	for _, data := range testData {
		db.Create(&data)
	}

	return db
}

func TestExist_ValidatorFunction(t *testing.T) {
	db := setupExistTestDB(t)
	v := validator.New()

	// Register the exist validator
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	// Test struct with exist validation
	type TestStruct struct {
		UserID    string `validate:"exist=exist_test_models.id"`
		UserEmail string `validate:"exist=exist_test_models.email"`
		UserName  string `validate:"exist=exist_test_models.name"`
		Optional  string `validate:"omitempty,exist=exist_test_models.email"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "valid existing ID",
			input: TestStruct{
				UserID:    "1",
				UserEmail: "user1@example.com",
				UserName:  "User One",
				Optional:  "",
			},
			expectValid: true,
			description: "Should be valid with existing values",
		},
		{
			name: "valid existing email",
			input: TestStruct{
				UserID:    "2",
				UserEmail: "user2@example.com",
				UserName:  "User Two",
				Optional:  "user3@example.com",
			},
			expectValid: true,
			description: "Should be valid with existing values and optional field",
		},
		{
			name: "invalid non-existing ID",
			input: TestStruct{
				UserID:    "999",
				UserEmail: "user1@example.com",
				UserName:  "User One",
				Optional:  "",
			},
			expectValid: false,
			description: "Should be invalid with non-existing ID",
		},
		{
			name: "invalid non-existing email",
			input: TestStruct{
				UserID:    "1",
				UserEmail: "nonexistent@example.com",
				UserName:  "User One",
				Optional:  "",
			},
			expectValid: false,
			description: "Should be invalid with non-existing email",
		},
		{
			name: "invalid non-existing name",
			input: TestStruct{
				UserID:    "1",
				UserEmail: "user1@example.com",
				UserName:  "Non Existent User",
				Optional:  "",
			},
			expectValid: false,
			description: "Should be invalid with non-existing name",
		},
		{
			name: "invalid optional field",
			input: TestStruct{
				UserID:    "1",
				UserEmail: "user1@example.com",
				UserName:  "User One",
				Optional:  "nonexistent@example.com",
			},
			expectValid: false,
			description: "Should be invalid with non-existing optional field",
		},
		{
			name: "valid empty optional field",
			input: TestStruct{
				UserID:    "3",
				UserEmail: "user3@example.com",
				UserName:  "User Three",
				Optional:  "",
			},
			expectValid: true,
			description: "Should be valid with empty optional field (omitempty)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestExist_InvalidTableColumnFormat(t *testing.T) {
	db := setupExistTestDB(t)
	v := validator.New()

	// Register the exist validator
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	// Malformed tags must fail CLOSED: a broken tag silently disabling the
	// check would let unverified values through.
	t.Run("malformed tags fail closed", func(t *testing.T) {
		malformedCases := []struct {
			name  string
			input any
		}{
			{
				name: "missing dot",
				input: struct {
					Value string `validate:"exist=invalid_format"`
				}{Value: "test"},
			},
			{
				name: "too many segments",
				input: struct {
					Value string `validate:"exist=table.column.extra"`
				}{Value: "test"},
			},
			{
				name: "empty parameter",
				input: struct {
					Value string `validate:"exist="`
				}{Value: "test"},
			},
			{
				name: "unsafe table identifier",
				input: struct {
					Value string `validate:"exist=bad-table.email"`
				}{Value: "test"},
			},
			{
				name: "unsafe column identifier",
				input: struct {
					Value string `validate:"exist=exist_test_models.email;--"`
				}{Value: "test"},
			},
			{
				name: "identifier starting with digit",
				input: struct {
					Value string `validate:"exist=1table.email"`
				}{Value: "test"},
			},
		}

		for _, tc := range malformedCases {
			t.Run(tc.name, func(t *testing.T) {
				if err := v.Struct(tc.input); err == nil {
					t.Errorf("expected malformed tag to fail closed, but validation passed")
				}
			})
		}
	})

	// A well-formed tag keeps working normally alongside the hardening.
	type ValidStruct struct {
		ValidFormat string `validate:"exist=exist_test_models.email"`
	}

	t.Run("valid format with existing value", func(t *testing.T) {
		if err := v.Struct(ValidStruct{ValidFormat: "user2@example.com"}); err != nil {
			t.Errorf("expected validation to pass for existing value, got: %v", err)
		}
	})

	t.Run("valid format with non-existing value", func(t *testing.T) {
		if err := v.Struct(ValidStruct{ValidFormat: "nonexistent@example.com"}); err == nil {
			t.Errorf("expected validation to fail for non-existing value")
		}
	})
}

func TestExist_DeletedAtColumnLookupIsCached(t *testing.T) {
	db := setupExistTestDB(t)

	cv, ok := New(db).(*customValidator)
	if !ok {
		t.Fatal("New should return *customValidator")
	}

	// First lookup goes to the migrator and is cached.
	if cv.hasDeletedAtColumn("exist_test_models") {
		t.Fatal("exist_test_models should not have a deleted_at column")
	}

	// Poison the cache to prove subsequent calls never hit the migrator again.
	cv.deletedAtCache.Store("exist_test_models", true)
	if !cv.hasDeletedAtColumn("exist_test_models") {
		t.Error("expected the cached value to be returned instead of a fresh migrator lookup")
	}
}

func TestExist_DatabaseError(t *testing.T) {
	// Create a validator with a closed database connection
	db := setupExistTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.Close() // Close the connection to simulate database error

	v := validator.New()
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	type TestStruct struct {
		UserEmail string `validate:"exist=exist_test_models.email"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "database error should fail validation",
			input: TestStruct{
				UserEmail: "user1@example.com",
			},
			expectValid: false, // Should fail when database error occurs
			description: "Should fail validation when database error occurs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestExist_EdgeCases(t *testing.T) {
	db := setupExistTestDB(t)
	v := validator.New()
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	type TestStruct struct {
		Value string `validate:"exist=exist_test_models.email"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "empty string",
			input: TestStruct{
				Value: "",
			},
			expectValid: false,
			description: "Should fail validation for empty string",
		},
		{
			name: "whitespace only",
			input: TestStruct{
				Value: "   ",
			},
			expectValid: false,
			description: "Should fail validation for whitespace only",
		},
		{
			name: "case sensitive check",
			input: TestStruct{
				Value: "USER1@EXAMPLE.COM",
			},
			expectValid: false,
			description: "Should fail validation for different case (case sensitive)",
		},
		{
			name: "special characters",
			input: TestStruct{
				Value: "user1+test@example.com",
			},
			expectValid: false,
			description: "Should fail validation for non-existing value with special characters",
		},
		{
			name: "unicode characters",
			input: TestStruct{
				Value: "üser1@example.com",
			},
			expectValid: false,
			description: "Should fail validation for non-existing value with unicode characters",
		},
		{
			name: "very long string",
			input: TestStruct{
				Value: "verylongemailaddressthatdoesnotexistinthedatabase@verylongdomainnamethatisunusuallylong.com",
			},
			expectValid: false,
			description: "Should fail validation for very long non-existing value",
		},
		{
			name: "sql injection attempt",
			input: TestStruct{
				Value: "user1@example.com'; DROP TABLE exist_test_models; --",
			},
			expectValid: false,
			description: "Should safely handle SQL injection attempts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestExist_NonExistentTable(t *testing.T) {
	db := setupExistTestDB(t)
	v := validator.New()
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	type TestStruct struct {
		Value string `validate:"exist=non_existent_table.email"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "non-existent table",
			input: TestStruct{
				Value: "test@example.com",
			},
			expectValid: false,
			description: "Should fail validation for non-existent table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestExist_NonExistentColumn(t *testing.T) {
	db := setupExistTestDB(t)
	v := validator.New()
	customValidator := New(db)
	v.RegisterValidation("exist", customValidator.Exist())

	type TestStruct struct {
		Value string `validate:"exist=exist_test_models.non_existent_column"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "non-existent column",
			input: TestStruct{
				Value: "test@example.com",
			},
			expectValid: false,
			description: "Should fail validation for non-existent column",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}
