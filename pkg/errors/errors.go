// Package errors defines the application's structured error types.
package errors

import (
	"errors"
	"net/http"
)

// AppError is the standard application error wrapper.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped underlying error, making it compatible
// with Go's standard errors package (errors.Is, errors.As).
func (e *AppError) Unwrap() error {
	return e.Err
}

// Predefined errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrDatabase       = errors.New("database error")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInternalServer = errors.New("internal server error")
	ErrForbidden      = errors.New("forbidden")
)

// NewAppError creates an AppError with an explicit status code and wrapped cause.
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewNotFoundError creates a not-found AppError.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: message,
		Err:     ErrNotFound,
	}
}

// NewBadRequestError creates a bad-request AppError.
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     ErrInvalidInput,
	}
}

// NewUnauthorizedError creates an unauthorized AppError.
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

// NewForbiddenError creates a forbidden AppError.
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: message,
		Err:     ErrForbidden,
	}
}

// NewInternalServerError creates an internal-server-error AppError.
func NewInternalServerError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

// IsNotFound reports whether err wraps the sentinel not-found error.
func IsNotFound(err AppError) bool {
	return errors.Is(err.Err, ErrNotFound)
}
