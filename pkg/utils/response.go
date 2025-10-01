package utils

import "github.com/go-playground/validator/v10"

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Error   any    `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

func BuildResponseSuccess(message string, data any) Response {
	res := Response{
		Status:  true,
		Message: message,
		Data:    data,
	}
	return res
}

func BuildResponseFailed(message string, err string) Response {
	res := Response{
		Status:  false,
		Message: message,
		Error:   err,
	}
	return res
}

func BuildResponseValidationError(err validator.ValidationErrors) Response {
	res := Response{
		Status:  false,
		Message: "Validation failed",
		Error:   FormatValidationErrors(err),
	}
	return res
}
