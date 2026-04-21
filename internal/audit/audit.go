// Package audit centralizes background audit-log writes so services don't each
// reimplement user attribution, context detachment, and error correlation.
//
// Services own the message phrasing (domain language); this package owns
// everything else.
package audit

import (
	"context"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
)

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
func Record(ctx context.Context, repo repository.LogRepository, entry Entry) {
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

	bgCtx := context.WithoutCancel(ctx)
	go func() {
		if err := repo.Create(bgCtx, log); err != nil {
			logger.Ctx(bgCtx).Error("Failed to create audit log",
				zap.String("entity_type", entry.EntityType),
				zap.Uint("entity_id", entry.EntityID),
				zap.String("action", string(entry.Action)),
				zap.Error(err),
			)
		}
	}()
}
