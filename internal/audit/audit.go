// Package audit centralizes background audit-log writes so services don't each
// reimplement user attribution, context detachment, and error correlation.
//
// Services own the message phrasing (domain language); this package owns
// everything else.
package audit

import (
	"context"
	"sync"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
)

// LogWriter is the narrow persistence dependency audit.Record needs: the
// ability to append a single log row. Declared here — at the consumer — so
// callers hand audit a one-method contract instead of the full log-module
// repository, and so this package does not import another module's repository.
// The log module's LogRepository satisfies it structurally.
type LogWriter interface {
	Create(ctx context.Context, log *models.Log) error
}

// pending tracks in-flight background writes so graceful shutdown can wait
// for them (via Drain) before the database connection is closed.
var pending sync.WaitGroup

// Go runs fn in a background goroutine tracked by Drain. Use it for audit-ish
// writes that can't go through Record (e.g. attribution known before the user
// exists in the request context) so shutdown still waits for them.
func Go(fn func()) {
	pending.Go(fn)
}

// Drain blocks until every in-flight audit write has finished, or until ctx
// expires. Call it during shutdown after the HTTP server has stopped and
// before the DB closes.
func Drain(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		pending.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Entry is a single audit record to emit. Message is built by the caller.
type Entry struct {
	Action     models.LogAction
	EntityType string
	EntityID   uint
	Message    string
}

// UserName returns the user name from ctx, falling back to "Unknown" when
// absent, so audit-log phrasing stays consistent across services.
func UserName(ctx context.Context) string {
	name, _ := utils.GetUserNameFromContext(ctx)
	if name == "" {
		return "Unknown"
	}
	return name
}

// Record writes an audit entry in the background, attributed to the user in
// ctx. Cancellation is detached (context.WithoutCancel) so the write can
// complete after the originating request returns, but request-scoped logging
// fields are preserved so a failed write stays correlated to its request.
func Record(ctx context.Context, repo LogWriter, entry Entry) {
	var userID *uint
	if id, ok := utils.GetUserIDFromContext(ctx); ok {
		userID = &id
	}

	log := &models.Log{
		UserID:     userID,
		Action:     entry.Action,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		Message:    entry.Message,
	}

	// Detach cancellation so the write can outlive the request, and strip any
	// transaction the caller's context carries: context.WithoutCancel preserves
	// context values, so without the strip a caller invoking Record from inside
	// a transaction would hand the background goroutine a tx that is committed
	// or rolled back by the time the write runs.
	bgCtx := utils.StripTx(context.WithoutCancel(ctx))
	Go(func() {
		if err := repo.Create(bgCtx, log); err != nil {
			logger.Ctx(bgCtx).Error("Failed to create audit log",
				zap.String("entity_type", entry.EntityType),
				zap.Uint("entity_id", entry.EntityID),
				zap.String("action", string(entry.Action)),
				zap.Error(err),
			)
		}
	})
}
