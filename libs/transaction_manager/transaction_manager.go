package transaction_manager

import (
	"context"

	"github.com/PhantomX7/go-starter/pkg/utils"

	"gorm.io/gorm"
)

type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type transactionManager struct {
	DB *gorm.DB
}

func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{DB: db}
}

func (tm *transactionManager) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	err := tm.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctxWithTx := utils.SetTxToContext(ctx, tx)
		if err := fn(ctxWithTx); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}