package utils

import (
	"context"

	"gorm.io/gorm"
)

// Context key for transaction
type contextKey string

const txKey contextKey = "db_transaction"

// Context helpers
func SetTxToContext(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

func GetTxFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return nil
}