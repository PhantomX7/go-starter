package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertModuleData(t *testing.T) {
	t.Parallel()

	converter := NewCaseConverter()

	tests := []struct {
		name  string
		input string
		want  ModuleData
	}{
		{
			name:  "single word",
			input: "category",
			want: ModuleData{
				SnakeCase:  "category",
				CamelCase:  "category",
				PascalCase: "Category",
				LowerCase:  "category",
				KebabCase:  "category",
				TableName:  "categories", // inflection pluralization, not naive +s
			},
		},
		{
			name:  "snake_case input",
			input: "user_profile",
			want: ModuleData{
				SnakeCase:  "user_profile",
				CamelCase:  "userProfile",
				PascalCase: "UserProfile",
				LowerCase:  "userprofile",
				KebabCase:  "user-profile",
				TableName:  "user_profiles",
			},
		},
		{
			name:  "camelCase input",
			input: "userProfile",
			want: ModuleData{
				SnakeCase:  "user_profile",
				CamelCase:  "userProfile",
				PascalCase: "UserProfile",
				LowerCase:  "userprofile",
				KebabCase:  "user-profile",
				TableName:  "user_profiles",
			},
		},
		{
			name:  "PascalCase input",
			input: "UserProfile",
			want: ModuleData{
				SnakeCase:  "user_profile",
				CamelCase:  "userProfile",
				PascalCase: "UserProfile",
				LowerCase:  "userprofile",
				KebabCase:  "user-profile",
				TableName:  "user_profiles",
			},
		},
		{
			name:  "spaced input with extra whitespace",
			input: "  product   category ",
			want: ModuleData{
				SnakeCase:  "product_category",
				CamelCase:  "productCategory",
				PascalCase: "ProductCategory",
				LowerCase:  "productcategory",
				KebabCase:  "product-category",
				TableName:  "product_categories",
			},
		},
		{
			name:  "three word PascalCase",
			input: "OrderLineItem",
			want: ModuleData{
				SnakeCase:  "order_line_item",
				CamelCase:  "orderLineItem",
				PascalCase: "OrderLineItem",
				LowerCase:  "orderlineitem",
				KebabCase:  "order-line-item",
				TableName:  "order_line_items",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, converter.ConvertModuleData(tt.input))
		})
	}
}

func TestConvertModuleDataKebabInput(t *testing.T) {
	t.Parallel()

	converter := NewCaseConverter()
	got := converter.ConvertModuleData("user-profile")

	require.Equal(t, "user_profile", got.SnakeCase)
	require.Equal(t, "userProfile", got.CamelCase)
	require.Equal(t, "UserProfile", got.PascalCase)
	require.Equal(t, "user-profile", got.KebabCase)
	require.Equal(t, "user_profiles", got.TableName)
}

func TestValidateModuleName(t *testing.T) {
	t.Parallel()

	converter := NewCaseConverter()

	valid := []string{"category", "userProfile", "UserProfile", "user_profile", "user profile", "post2"}
	for _, name := range valid {
		require.NoError(t, converter.ValidateModuleName(name), name)
	}

	t.Run("empty and whitespace-only names", func(t *testing.T) {
		t.Parallel()
		require.ErrorContains(t, converter.ValidateModuleName(""), "cannot be empty")
		require.ErrorContains(t, converter.ValidateModuleName("   "), "cannot be empty")
		require.ErrorContains(t, converter.ValidateModuleName("___"), "cannot be empty")
	})

	t.Run("invalid characters", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"user.profile", "user@name", "user!"} {
			err := converter.ValidateModuleName(name)
			require.Error(t, err, name)
			require.ErrorContains(t, err, "can only contain letters, numbers, and underscores")
		}
	})

	t.Run("leading digit", func(t *testing.T) {
		t.Parallel()
		err := converter.ValidateModuleName("9lives")
		require.Error(t, err)
		require.ErrorContains(t, err, "must start with a letter")
	})

	t.Run("go keywords", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"func", "type", "range", "select", "Import"} {
			err := converter.ValidateModuleName(name)
			require.Error(t, err, name)
			require.ErrorContains(t, err, "reserved keyword")
		}
	})
}

func TestValidationErrorMessage(t *testing.T) {
	t.Parallel()

	err := &ValidationError{Message: "boom"}
	require.Equal(t, "boom", err.Error())
}

func TestDetectInputFormat(t *testing.T) {
	t.Parallel()

	converter := NewCaseConverter()

	tests := []struct {
		input string
		want  string
	}{
		{"user_profile", "snake_case"},
		{"user-profile", "kebab-case"},
		{"userProfile", "camelCase"},
		{"UserProfile", "PascalCase"},
		{"user", "unknown"},
		{"USER", "unknown"},
	}

	for _, tt := range tests {
		require.Equal(t, tt.want, converter.DetectInputFormat(tt.input), tt.input)
	}
}
