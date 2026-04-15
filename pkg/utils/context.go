// pkg/utils/context.go
package utils

import (
	"context"

	"github.com/PhantomX7/athleton/pkg/errors"

	"gorm.io/gorm"
)

// Context key for transaction and values
type contextKey string

const (
	txKey        contextKey = "db_transaction"
	valuesKey    contextKey = "values"
	requestIDKey contextKey = "request_id"
)

// ContextValues holds values extracted from the context.
type ContextValues struct {
	UserID      uint
	UserName    string
	Role        string
	AdminRoleID *uint
	RequestID   string
}

// NewContextWithValues creates a new context with the provided ContextValues.
func NewContextWithValues(ctx context.Context, values ContextValues) context.Context {
	return context.WithValue(ctx, valuesKey, values)
}

// ValuesFromContext retrieves ContextValues from the given context.
// It returns an error if the values are not found or are of an unexpected type.
func ValuesFromContext(ctx context.Context) (*ContextValues, error) {
	values, ok := ctx.Value(valuesKey).(ContextValues)
	if !ok {
		return nil, errors.NewBadRequestError("invalid authentication context")
	}
	return &values, nil
}

// GetUserIDFromContext retrieves user ID from context.
// Returns 0 and false if not found.
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	values, ok := ctx.Value(valuesKey).(ContextValues)
	if !ok {
		return 0, false
	}
	return values.UserID, true
}

// GetUserNameFromContext retrieves user name from context.
// Returns empty string and false if not found.
func GetUserNameFromContext(ctx context.Context) (string, bool) {
	values, ok := ctx.Value(valuesKey).(ContextValues)
	if !ok {
		return "", false
	}
	return values.UserName, true
}

// GetRoleFromContext retrieves role from context.
// Returns empty string and false if not found.
func GetRoleFromContext(ctx context.Context) (string, bool) {
	values, ok := ctx.Value(valuesKey).(ContextValues)
	if !ok {
		return "", false
	}
	return values.Role, true
}

// GetAdminRoleIDFromContext retrieves admin role ID from context.
// Returns nil if not found or not set.
func GetAdminRoleIDFromContext(ctx context.Context) *uint {
	values, ok := ctx.Value(valuesKey).(ContextValues)
	if !ok {
		return nil
	}
	return values.AdminRoleID
}

// SetRequestIDToContext sets the request ID to the context.
func SetRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestIDFromContext retrieves the request ID from the context.
// It returns an empty string if no request ID is found.
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// SetTxToContext sets the transaction to the context for database operations.
func SetTxToContext(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTxFromContext retrieves the transaction from the context.
// It returns nil if no transaction is found.
func GetTxFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return nil
}
