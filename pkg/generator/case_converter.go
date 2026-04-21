// Package generator contains the code-generation helpers used by the module generator.
package generator

import (
	"regexp"
	"strings"

	"github.com/stoewer/go-strcase"
)

// CaseConverter handles case conversions for module generation
type CaseConverter struct{}

// NewCaseConverter creates a new instance of CaseConverter
func NewCaseConverter() *CaseConverter {
	return &CaseConverter{}
}

// ConvertModuleData converts a module name to different case formats
func (c *CaseConverter) ConvertModuleData(moduleName string) ModuleData {
	// Normalize the input by removing extra spaces and underscores
	normalized := c.normalizeInput(moduleName)

	// Convert to different cases
	snakeCase := strcase.SnakeCase(normalized)
	camelCase := strcase.LowerCamelCase(normalized)
	pascalCase := strcase.UpperCamelCase(normalized)
	lowerCase := strings.ToLower(strings.ReplaceAll(normalized, "_", ""))
	kebabCase := strcase.KebabCase(normalized)

	return ModuleData{
		SnakeCase:  snakeCase,
		CamelCase:  camelCase,
		PascalCase: pascalCase,
		LowerCase:  lowerCase,
		KebabCase:  kebabCase,
	}
}

// normalizeInput cleans up the input string
func (c *CaseConverter) normalizeInput(input string) string {
	// Remove extra spaces
	input = strings.TrimSpace(input)

	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	input = spaceRegex.ReplaceAllString(input, " ")

	// Replace spaces with underscores for processing
	input = strings.ReplaceAll(input, " ", "_")

	// Remove multiple underscores
	underscoreRegex := regexp.MustCompile(`_+`)
	input = underscoreRegex.ReplaceAllString(input, "_")

	// Remove leading/trailing underscores
	input = strings.Trim(input, "_")

	return input
}

// DetectInputFormat attempts to detect the format of the input string
func (c *CaseConverter) DetectInputFormat(input string) string {
	if strings.Contains(input, "_") {
		return "snake_case"
	}
	if strings.Contains(input, "-") {
		return "kebab-case"
	}
	if strings.ToLower(input) != input && strings.ToUpper(input) != input && !strings.Contains(input, "_") {
		// Check if it's camelCase or PascalCase
		if strings.ToUpper(string(input[0])) == string(input[0]) {
			return "PascalCase"
		}
		return "camelCase"
	}
	return "unknown"
}

// ValidateModuleName validates the module name
func (c *CaseConverter) ValidateModuleName(moduleName string) error {
	normalized := c.normalizeInput(moduleName)

	if normalized == "" {
		return &ValidationError{Message: "module name cannot be empty"}
	}

	// Check for invalid characters
	validRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !validRegex.MatchString(normalized) {
		return &ValidationError{Message: "module name can only contain letters, numbers, and underscores, and must start with a letter"}
	}

	// Check for reserved Go keywords
	reservedWords := []string{
		"break", "default", "func", "interface", "select",
		"case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var",
	}

	lowerNormalized := strings.ToLower(normalized)
	for _, reserved := range reservedWords {
		if lowerNormalized == reserved {
			return &ValidationError{Message: "module name cannot be a Go reserved keyword"}
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
