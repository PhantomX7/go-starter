package utils

import "net/http"

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Ensure APIError implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

func NewNotFoundError(message string) *APIError {
	return &APIError{
		Code:    http.StatusNotFound,
		Message: message,
	}
}
