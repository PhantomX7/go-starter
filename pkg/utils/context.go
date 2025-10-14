package utils

import (
	"context"

	"github.com/PhantomX7/go-starter/pkg/errors"
	"gorm.io/gorm"
)

// Context key for transaction
type contextKey string

const (
	txKey     contextKey = "db_transaction"
	valuesKey contextKey = "values"
)

// ContextValues holds values extracted from the context.
type ContextValues struct {
	UserID uint
	Role   string
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
