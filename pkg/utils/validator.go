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
	errorField := strcase.SnakeCase(e.Field())

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", errorField)
	case "exists":
		return fmt.Sprintf("%s is required", errorField)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", errorField)
	case "min":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters long", errorField, param)
		}
		return fmt.Sprintf("%s must be at least %s", errorField, param)
	case "max":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s must be no more than %s characters long", errorField, param)
		}
		return fmt.Sprintf("%s must be no more than %s", errorField, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", errorField, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of the following: %s", errorField, strings.ReplaceAll(param, " ", ", "))
	case "numeric":
		return fmt.Sprintf("%s must be a numeric value", errorField)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", errorField)
	case "unique":
		return fmt.Sprintf("%s must be unique", errorField)
	case "exist":
		return fmt.Sprintf("%s does not exist", errorField)
	default:
		return fmt.Sprintf("%s is invalid", errorField)
	}
}
