// internal/modules/admin_role/repository/repository.go
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/PhantomX7/athleton/internal/models"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdminRoleRepository defines the interface for admin role repository operations
type AdminRoleRepository interface {
	repository.IRepository[models.AdminRole]
	FindByName(ctx context.Context, name string) (*models.AdminRole, error)
	CountUsersWithRole(ctx context.Context, roleID uint) (int64, error)
}

// adminRoleRepository implements the AdminRoleRepository interface
type adminRoleRepository struct {
	repository.Repository[models.AdminRole]
}

// NewAdminRoleRepository creates a new instance of AdminRoleRepository
func NewAdminRoleRepository(db *gorm.DB) AdminRoleRepository {
	return &adminRoleRepository{
		Repository: repository.Repository[models.AdminRole]{
			DB: db,
		},
	}
}

// FindByName finds an admin role by name
func (r *adminRoleRepository) FindByName(ctx context.Context, name string) (*models.AdminRole, error) {
	var entity models.AdminRole
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Where("name = ?", name).
		Take(&entity).Error

	duration := time.Since(start)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("admin role not found")
		}
		return nil, cerrors.NewInternalServerError("failed to find admin role by name", err)
	}

	r.LogSlowQuery(ctx, "FindByName", duration, 500*time.Millisecond)
	return &entity, nil
}

// CountUsersWithRole counts users that have a specific admin role
func (r *adminRoleRepository) CountUsersWithRole(ctx context.Context, roleID uint) (int64, error) {
	var count int64
	start := time.Now()

	err := r.GetDB(ctx).WithContext(ctx).
		Model(&models.User{}).
		Where("admin_role_id = ?", roleID).
		Count(&count).Error

	duration := time.Since(start)

	if err != nil {
		logger.Error("Failed to count users with admin role",
			zap.String("request_id", utils.GetRequestIDFromContext(ctx)),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return 0, cerrors.NewInternalServerError("failed to count users with role", err)
	}

	r.LogSlowQuery(ctx, "CountUsersWithRole", duration, 500*time.Millisecond)
	return count, nil
}
