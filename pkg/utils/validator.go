package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/stoewer/go-strcase"
)

type ValidationErrorResponse struct {
	TotalErrors int               `json:"total_errors"`
	Fields      map[string]string `json:"fields"`
}

// FormatValidationErrors creates a structured response from validator.ValidationErrors.
func FormatValidationErrors(err validator.ValidationErrors) ValidationErrorResponse {
	fieldErrors := make(map[string]string)
	for _, e := range err {
		// Use snake_case for JSON field names for consistency.
		fieldName := strcase.SnakeCase(e.Field())
		fieldErrors[fieldName] = formatSingleError(e)
	}

	return ValidationErrorResponse{
		TotalErrors: len(err),
		Fields:      fieldErrors,
	}
}

// formatSingleError creates a human-readable message for a single validation error.
func formatSingleError(e validator.FieldError) string {
	// e.Param() gets the value associated with the tag, e.g., '8' for 'min=8'.
	param := e.Param()

	switch e.Tag() {
	case "required":
		return "This field is required."
	case "email":
		return "Must be a valid email address."
	case "min":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("Must be at least %s characters long.", param)
		}
		return fmt.Sprintf("Must be at least %s.", param)
	case "max":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("Must be no more than %s characters long.", param)
		}
		return fmt.Sprintf("Must be no more than %s.", param)
	case "len":
		return fmt.Sprintf("Length must be exactly %s.", param)
	case "oneof":
		return fmt.Sprintf("Must be one of the following: %s.", strings.ReplaceAll(param, " ", ", "))
	case "numeric":
		return "Must be a numeric value."
	case "alphanum":
		return "Must contain only letters and numbers."
	default:
		return "This field is invalid."
	}
}
