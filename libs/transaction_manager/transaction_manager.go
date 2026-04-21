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
	err := tm.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctxWithTx := utils.SetTxToContext(ctx, tx)
		return fn(ctxWithTx)
	})

	return err
}
