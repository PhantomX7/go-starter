// Package transaction_manager provides helpers for transaction-scoped work.
package transaction_manager

import (
	"context"

	"github.com/PhantomX7/athleton/pkg/utils"

	"gorm.io/gorm"
)

// TransactionManager runs closures inside a database transaction.
type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type transactionManager struct {
	DB *gorm.DB
}

// NewTransactionManager builds a TransactionManager backed by GORM.
func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{DB: db}
}

func (tm *transactionManager) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Join an ambient transaction when one is already in the context: composed
	// services (A in a tx calling B which also wants a tx) must share one
	// transaction, so B sees A's uncommitted writes and rolls back with A.
	// Opening a second independent transaction here would commit B's writes
	// even when A later fails.
	if utils.GetTxFromContext(ctx) != nil {
		return fn(ctx)
	}

	return tm.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(utils.SetTxToContext(ctx, tx))
	})
}
