// Package repository contains data-access code for the admin-role module.
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
	"gorm.io/gorm/clause"
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . AdminRoleRepository

// AdminRoleRepository defines the interface for admin role repository operations.
type AdminRoleRepository interface {
	repository.Repository[models.AdminRole]
	FindByName(ctx context.Context, name string) (*models.AdminRole, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*models.AdminRole, error)
	CountUsersWithRole(ctx context.Context, roleID uint) (int64, error)
}

type adminRoleRepository struct {
	repository.BaseRepository[models.AdminRole]
}

// NewAdminRoleRepository builds an AdminRoleRepository backed by GORM.
func NewAdminRoleRepository(db *gorm.DB) AdminRoleRepository {
	return &adminRoleRepository{
		BaseRepository: repository.NewBaseRepository[models.AdminRole](db),
	}
}

// FindByName returns the admin role whose name matches exactly (the column is
// uniquely indexed so at most one row comes back).
func (r *adminRoleRepository) FindByName(ctx context.Context, name string) (*models.AdminRole, error) {
	start := time.Now()

	entity, err := gorm.G[models.AdminRole](r.GetDB(ctx)).
		Where(generated.AdminRole.Name.Eq(name)).
		First(ctx)

	r.LogSlowRead(ctx, "FindByName", time.Since(start))

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("admin role not found")
		}
		return nil, cerrors.NewInternalServerError("failed to find admin role by name", err)
	}

	return &entity, nil
}

// FindByIDForUpdate loads the admin-role row under a pessimistic SELECT ...
// FOR UPDATE lock. Check-then-act flows on the role (delete-if-unassigned,
// assign-if-exists) lock the row first so they serialize against each other.
// Call it only inside a transaction — the lock is held until commit/rollback.
// On sqlite (unit tests via glebarez) the locking clause is a no-op, which is
// fine: sqlite serializes writers at the database level anyway.
func (r *adminRoleRepository) FindByIDForUpdate(ctx context.Context, id uint) (*models.AdminRole, error) {
	start := time.Now()

	var role models.AdminRole
	err := r.GetDB(ctx).WithContext(ctx).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		First(&role, "id = ?", id).Error

	r.LogSlowRead(ctx, "FindByIDForUpdate", time.Since(start))

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cerrors.NewNotFoundError("admin role not found")
		}
		return nil, cerrors.NewInternalServerError("failed to find admin role by id", err)
	}

	return &role, nil
}

// CountUsersWithRole counts non-deleted users assigned to the given admin role.
// GORM automatically applies `deleted_at IS NULL` because User embeds
// gorm.DeletedAt via the Timestamp mixin, so we do not need to add it manually.
func (r *adminRoleRepository) CountUsersWithRole(ctx context.Context, roleID uint) (int64, error) {
	start := time.Now()

	count, err := gorm.G[models.User](r.GetDB(ctx)).
		Where(generated.User.AdminRoleID.Eq(roleID)).
		Count(ctx, "*")

	r.LogSlowRead(ctx, "CountUsersWithRole", time.Since(start))

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
