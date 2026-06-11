package transaction_manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type txRecord struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&txRecord{}))

	return db
}

func TestExecuteInTransactionCommitsOnSuccess(t *testing.T) {
	db := setupDB(t)
	tm := transaction_manager.NewTransactionManager(db)

	err := tm.ExecuteInTransaction(context.Background(), func(ctx context.Context) error {
		tx := utils.GetTxFromContext(ctx)
		require.NotNil(t, tx)
		return tx.Create(&txRecord{Name: "committed"}).Error
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&txRecord{}).Where("name = ?", "committed").Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestExecuteInTransactionRollsBackOnError(t *testing.T) {
	db := setupDB(t)
	tm := transaction_manager.NewTransactionManager(db)

	sentinel := errors.New("boom")
	err := tm.ExecuteInTransaction(context.Background(), func(ctx context.Context) error {
		tx := utils.GetTxFromContext(ctx)
		require.NotNil(t, tx)
		require.NoError(t, tx.Create(&txRecord{Name: "rolled back"}).Error)

		// The write is visible inside the transaction before rollback.
		var inTxCount int64
		require.NoError(t, tx.Model(&txRecord{}).Count(&inTxCount).Error)
		require.EqualValues(t, 1, inTxCount)

		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	var count int64
	require.NoError(t, db.Model(&txRecord{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestExecuteInTransactionContextCarriesTx(t *testing.T) {
	db := setupDB(t)
	tm := transaction_manager.NewTransactionManager(db)

	type ctxKey struct{}
	baseCtx := context.WithValue(context.Background(), ctxKey{}, "kept")

	var called bool
	err := tm.ExecuteInTransaction(baseCtx, func(ctx context.Context) error {
		called = true

		// The tx in the context is a transaction handle, not the base DB.
		tx := utils.GetTxFromContext(ctx)
		require.NotNil(t, tx)
		require.NotSame(t, db, tx)

		// Values from the caller's context are preserved.
		require.Equal(t, "kept", ctx.Value(ctxKey{}))

		// The tx is usable for queries.
		var one int
		require.NoError(t, tx.Raw("SELECT 1").Scan(&one).Error)
		require.Equal(t, 1, one)

		return nil
	})
	require.NoError(t, err)
	require.True(t, called)

	// The base context is untouched.
	require.Nil(t, utils.GetTxFromContext(baseCtx))
}
