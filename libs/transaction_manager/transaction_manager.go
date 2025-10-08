package transaction_manager

import (
	"context"

	"github.com/PhantomX7/go-starter/pkg/utils"

	"gorm.io/gorm"
)

type TransactionManager interface {
	ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error
}

type transactionManager struct {
	DB *gorm.DB
}

func NewTransactionManager(db *gorm.DB) TransactionManager {
	return &transactionManager{DB: db}
}

func (tm *transactionManager) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx := tm.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Set transaction in context
	ctxWithTx := utils.SetTxToContext(ctx, tx)

	if err := fn(ctxWithTx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}