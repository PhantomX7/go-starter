// internal/modules/admin_role/repository/repository.go
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdminRoleRepository defines the interface for admin role repository operations.
type AdminRoleRepository interface {
	repository.IRepository[models.AdminRole]
	FindByName(ctx context.Context, name string) (*models.AdminRole, error)
	CountUsersWithRole(ctx context.Context, roleID uint) (int64, error)
}

type adminRoleRepository struct {
	repository.Repository[models.AdminRole]
}

func NewAdminRoleRepository(db *gorm.DB) AdminRoleRepository {
	return &adminRoleRepository{
		Repository: repository.Repository[models.AdminRole]{DB: db},
	}
}

// FindByName returns the admin role whose name matches exactly (the column is
// uniquely indexed so at most one row comes back).
func (r *adminRoleRepository) FindByName(ctx context.Context, name string) (*models.AdminRole, error) {
	start := time.Now()

	entity, err := gorm.G[models.AdminRole](r.GetDB(ctx)).
		Where(generated.AdminRole.Name.Eq(name)).
		First(ctx)

	r.LogSlowQuery(ctx, "FindByName", time.Since(start), 500*time.Millisecond)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("admin role not found")
		}
		return nil, cerrors.NewInternalServerError("failed to find admin role by name", err)
	}

	return &entity, nil
}

// CountUsersWithRole counts non-deleted users assigned to the given admin role.
// GORM automatically applies `deleted_at IS NULL` because User embeds
// gorm.DeletedAt via the Timestamp mixin, so we do not need to add it manually.
func (r *adminRoleRepository) CountUsersWithRole(ctx context.Context, roleID uint) (int64, error) {
	start := time.Now()

	count, err := gorm.G[models.User](r.GetDB(ctx)).
		Where(generated.User.AdminRoleID.Eq(roleID)).
		Count(ctx, "*")

	r.LogSlowQuery(ctx, "CountUsersWithRole", time.Since(start), 500*time.Millisecond)

	if err != nil {
		logger.Error("Failed to count users with admin role",
			zap.String("request_id", utils.GetRequestIDFromContext(ctx)),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return 0, cerrors.NewInternalServerError("failed to count users with role", err)
	}

	return count, nil
}
