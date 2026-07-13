package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/models"
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// newMockLogRepository returns a mock log repository that signals every Create
// call on the returned channel so tests can wait for the background goroutine
// spawned by audit.Record.
func newMockLogRepository(createErr error) (*logmocks.LogRepositoryMock, chan *models.Log) {
	created := make(chan *models.Log, 1)
	return &logmocks.LogRepositoryMock{
		CreateFunc: func(_ context.Context, entity *models.Log) error {
			created <- entity
			return createErr
		},
	}, created
}

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

// waitForCreate blocks until the mock repository receives a Create call or the
// timeout elapses. The timeout only fires when the code under test is broken.
func waitForCreate(t *testing.T, created chan *models.Log) *models.Log {
	t.Helper()

	select {
	case entity := <-created:
		return entity
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for repository Create call")
		return nil
	}
}

func TestRecordAttributesUserFromContext(t *testing.T) {
	setupLogger(t)

	repo, created := newMockLogRepository(nil)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID:   42,
		UserName: "kenichi",
	})

	audit.Record(ctx, repo, audit.Entry{
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeConfig,
		EntityID:   7,
		Message:    "kenichi created config",
	})

	got := waitForCreate(t, created)
	require.NotNil(t, got)
	require.NotNil(t, got.UserID)
	require.Equal(t, uint(42), *got.UserID)
	require.Equal(t, models.LogActionCreate, got.Action)
	require.Equal(t, models.LogEntityTypeConfig, got.EntityType)
	require.Equal(t, uint(7), got.EntityID)
	require.Equal(t, "kenichi created config", got.Message)
}

func TestRecordWithoutUserLeavesUserIDNil(t *testing.T) {
	setupLogger(t)

	repo, created := newMockLogRepository(nil)

	audit.Record(context.Background(), repo, audit.Entry{
		Action:     models.LogActionDelete,
		EntityType: models.LogEntityTypeUser,
		EntityID:   9,
		Message:    "user deleted",
	})

	got := waitForCreate(t, created)
	require.NotNil(t, got)
	require.Nil(t, got.UserID)
	require.Equal(t, models.LogActionDelete, got.Action)
	require.Equal(t, models.LogEntityTypeUser, got.EntityType)
	require.Equal(t, uint(9), got.EntityID)
	require.Equal(t, "user deleted", got.Message)
}

func TestRecordDoesNotUseCallerTransaction(t *testing.T) {
	setupLogger(t)

	// Simulate a caller invoking Record from inside a transaction: the tx is
	// set on the context exactly as libs/transaction_manager does. By the time
	// the background goroutine runs, that tx may be committed or rolled back,
	// so the audit write must never see it.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	// Capture the transaction (if any) carried by the context each Create call
	// receives, so the test can prove the background audit write does not
	// inherit the caller's transaction.
	created := make(chan *models.Log, 1)
	txSeen := make(chan *gorm.DB, 1)
	repo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entity *models.Log) error {
			txSeen <- utils.GetTxFromContext(ctx)
			created <- entity
			return nil
		},
	}
	ctx := utils.SetTxToContext(context.Background(), tx)

	audit.Record(ctx, repo, audit.Entry{
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeUser,
		EntityID:   4,
		Message:    "created inside tx",
	})

	waitForCreate(t, created)
	select {
	case seen := <-txSeen:
		require.Nil(t, seen, "background audit write must not reuse the caller's transaction")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for tx capture")
	}
}

func TestRecordSurvivesCanceledRequestContext(t *testing.T) {
	setupLogger(t)

	repo, created := newMockLogRepository(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancellation must be detached via context.WithoutCancel

	audit.Record(ctx, repo, audit.Entry{
		Action:     models.LogActionUpdate,
		EntityType: models.LogEntityTypeAdminRole,
		EntityID:   3,
		Message:    "role updated",
	})

	got := waitForCreate(t, created)
	require.NotNil(t, got)
	require.Equal(t, models.LogActionUpdate, got.Action)
}

func TestRecordRepoErrorDoesNotPanic(t *testing.T) {
	// Install an observed logger that signals once the goroutine's error log is
	// written, so the test waits for the full error path before restoring the
	// previous logger.
	core, observed := observer.New(zapcore.ErrorLevel)
	logged := make(chan struct{}, 1)
	prev := logger.Log
	logger.Log = zap.New(core, zap.Hooks(func(zapcore.Entry) error {
		select {
		case logged <- struct{}{}:
		default:
		}
		return nil
	}))

	repo, created := newMockLogRepository(errors.New("db down"))

	require.NotPanics(t, func() {
		audit.Record(context.Background(), repo, audit.Entry{
			Action:     models.LogActionLogin,
			EntityType: models.LogEntityTypeUser,
			EntityID:   1,
			Message:    "login",
		})
		waitForCreate(t, created)

		select {
		case <-logged:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for error log")
		}
	})

	logger.Log = prev

	entries := observed.All()
	require.Len(t, entries, 1)
	require.Equal(t, "Failed to create audit log", entries[0].Message)
}

func TestRecordSurvivesPanickingWriter(t *testing.T) {
	// A panic inside the background write runs on a detached goroutine, so it
	// bypasses the HTTP recovery middleware — unrecovered it would crash the
	// whole process. Record must contain it and log the panic instead.
	core, observed := observer.New(zapcore.ErrorLevel)
	logged := make(chan struct{}, 1)
	prev := logger.Log
	logger.Log = zap.New(core, zap.Hooks(func(zapcore.Entry) error {
		select {
		case logged <- struct{}{}:
		default:
		}
		return nil
	}))
	t.Cleanup(func() { logger.Log = prev })

	repo := &logmocks.LogRepositoryMock{
		CreateFunc: func(context.Context, *models.Log) error {
			panic("audit writer exploded")
		},
	}

	audit.Record(context.Background(), repo, audit.Entry{
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeUser,
		EntityID:   1,
		Message:    "created",
	})

	// The panicking goroutine must still complete from Drain's point of view.
	drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, audit.Drain(drainCtx))

	select {
	case <-logged:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for panic to be logged")
	}
	entries := observed.All()
	require.NotEmpty(t, entries)
	require.Contains(t, entries[0].Message, "panic")
}

func TestGoSurvivesPanickingFunc(t *testing.T) {
	setupLogger(t)

	require.NotPanics(t, func() {
		audit.Go(func() {
			panic("background job exploded")
		})
		drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, audit.Drain(drainCtx))
	})
}

func TestUserNameReturnsNameFromContext(t *testing.T) {
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID:   1,
		UserName: "kenichi",
	})

	require.Equal(t, "kenichi", audit.UserName(ctx))
}

func TestUserNameFallsBackToUnknown(t *testing.T) {
	// No values in context at all.
	require.Equal(t, "Unknown", audit.UserName(context.Background()))

	// Values present but name empty.
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1})
	require.Equal(t, "Unknown", audit.UserName(ctx))
}
