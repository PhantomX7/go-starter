package transaction_manager

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

const txKey contextKey = "db_transaction"

type Client interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	GetDB(ctx context.Context) *gorm.DB
}

type client struct {
	db *gorm.DB
}

func New(db *gorm.DB) Client {
	return &client{db: db}
}

// WithTransaction executes function within a transaction
// Automatically handles commit/rollback
func (c *client) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}

// GetDB returns transaction from context if exists, otherwise returns default DB
func (c *client) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return c.db
}