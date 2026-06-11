package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/models"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// mockLogRepository signals every Create call on a channel so tests can wait
// for the background goroutine spawned by audit.Record.
type mockLogRepository struct {
	createErr error
	created   chan *models.Log
}

func newMockLogRepository(createErr error) *mockLogRepository {
	return &mockLogRepository{
		createErr: createErr,
		created:   make(chan *models.Log, 1),
	}
}

func (m *mockLogRepository) Create(_ context.Context, entity *models.Log) error {
	m.created <- entity
	return m.createErr
}

func (m *mockLogRepository) Update(context.Context, *models.Log) error {
	panic("unexpected Update call")
}

func (m *mockLogRepository) Delete(context.Context, *models.Log) error {
	panic("unexpected Delete call")
}

func (m *mockLogRepository) FindByID(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindByID call")
}

func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}

func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

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
func waitForCreate(t *testing.T, repo *mockLogRepository) *models.Log {
	t.Helper()

	select {
	case entity := <-repo.created:
		return entity
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for repository Create call")
		return nil
	}
}

func TestRecordAttributesUserFromContext(t *testing.T) {
	setupLogger(t)

	repo := newMockLogRepository(nil)
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

	got := waitForCreate(t, repo)
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

	repo := newMockLogRepository(nil)

	audit.Record(context.Background(), repo, audit.Entry{
		Action:     models.LogActionDelete,
		EntityType: models.LogEntityTypeUser,
		EntityID:   9,
		Message:    "user deleted",
	})

	got := waitForCreate(t, repo)
	require.NotNil(t, got)
	require.Nil(t, got.UserID)
	require.Equal(t, models.LogActionDelete, got.Action)
	require.Equal(t, models.LogEntityTypeUser, got.EntityType)
	require.Equal(t, uint(9), got.EntityID)
	require.Equal(t, "user deleted", got.Message)
}

func TestRecordSurvivesCanceledRequestContext(t *testing.T) {
	setupLogger(t)

	repo := newMockLogRepository(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancellation must be detached via context.WithoutCancel

	audit.Record(ctx, repo, audit.Entry{
		Action:     models.LogActionUpdate,
		EntityType: models.LogEntityTypeAdminRole,
		EntityID:   3,
		Message:    "role updated",
	})

	got := waitForCreate(t, repo)
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

	repo := newMockLogRepository(errors.New("db down"))

	require.NotPanics(t, func() {
		audit.Record(context.Background(), repo, audit.Entry{
			Action:     models.LogActionLogin,
			EntityType: models.LogEntityTypeUser,
			EntityID:   1,
			Message:    "login",
		})
		waitForCreate(t, repo)

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
