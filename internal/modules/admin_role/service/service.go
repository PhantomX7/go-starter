// Package service contains the admin-role module business logic.
package service

import (
	"context"
	"strings"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepo "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . AdminRoleService

// AdminRoleService exposes the admin-role use cases used by handlers.
type AdminRoleService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.AdminRole, response.Meta, error)
	Create(ctx context.Context, req *dto.CreateAdminRoleRequest) (*models.AdminRole, error)
	Update(ctx context.Context, roleID uint, req *dto.UpdateAdminRoleRequest) (*models.AdminRole, error)
	Delete(ctx context.Context, roleID uint) error
	FindByID(ctx context.Context, roleID uint) (*models.AdminRole, error)
	GetAllPermissions(ctx context.Context) map[string][]map[string]string
}

type adminRoleService struct {
	adminRoleRepo adminrolerepo.AdminRoleRepository
	logRepository logRepository.LogRepository
	casbinClient  casbin.Client
	txManager     transaction_manager.TransactionManager
}

// NewAdminRoleService builds an AdminRoleService from its dependencies.
func NewAdminRoleService(
	adminRoleRepo adminrolerepo.AdminRoleRepository,
	logRepository logRepository.LogRepository,
	casbinClient casbin.Client,
	txManager transaction_manager.TransactionManager,
) AdminRoleService {
	return &adminRoleService{
		adminRoleRepo: adminRoleRepo,
		logRepository: logRepository,
		casbinClient:  casbinClient,
		txManager:     txManager,
	}
}

// Index implements AdminRoleService.
func (s *adminRoleService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
	roles, err := s.adminRoleRepo.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.adminRoleRepo.Count(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	for _, role := range roles {
		role.Permissions = s.casbinClient.GetRolePermissions(role.ID)
	}

	return roles, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements AdminRoleService.
func (s *adminRoleService) Create(ctx context.Context, req *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
	// Validate permissions
	invalidPerms := s.validatePermissions(req.Permissions)
	if len(invalidPerms) > 0 {
		return nil, cerrors.NewBadRequestError("invalid permissions: " + strings.Join(invalidPerms, ", "))
	}

	// A new role has no current permissions, so every requested permission is a
	// grant the caller must hold.
	if err := s.authorizeGrant(ctx, req.Permissions, func() []string { return nil }); err != nil {
		return nil, err
	}

	// Create admin role model. Permissions are not persisted on the row
	// (gorm:"-"); they are owned by Casbin and synced below.
	adminRole := &models.AdminRole{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
	}

	// Save to database. This is the only DB write in Create, so it is atomic on
	// its own and needs no explicit transaction.
	if err := s.adminRoleRepo.Create(ctx, adminRole); err != nil {
		return nil, err
	}

	// Add permissions to Casbin only after the role row is committed.
	//
	// Residual atomicity gap: Casbin policies are persisted through the casbin
	// client's own GORM adapter, which does not join our transaction context,
	// so the role write and the policy write can never be fully atomic. If the
	// Casbin sync fails we attempt a compensating delete of the role; if that
	// delete also fails, a role without permissions is left behind and must be
	// repaired manually (it grants no access, so it fails closed).
	if err := s.casbinClient.AddRolePermissions(adminRole.ID, req.Permissions); err != nil {
		// Compensating action: delete the created role
		if delErr := s.adminRoleRepo.Delete(ctx, adminRole); delErr != nil {
			logger.Ctx(ctx, zap.Uint("role_id", adminRole.ID)).Error(
				"CRITICAL: failed to roll back admin role after casbin sync failure; role exists without permissions",
				zap.Error(delErr),
			)
		}
		return nil, cerrors.NewInternalServerError("failed to set role permissions", err)
	}

	// Get permissions for response
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	// Create audit log
	s.createLog(ctx, models.LogActionCreate, adminRole.ID, adminRole.Name)

	return adminRole, nil
}

// Update implements AdminRoleService.
func (s *adminRoleService) Update(ctx context.Context, roleID uint, req *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
	// Validate and authorize the permission change before opening the
	// transaction: neither check needs the row lock, and rejecting here avoids
	// holding the lock (and a DB round trip) for requests that will be denied.
	if req.Permissions != nil {
		invalidPerms := s.validatePermissions(req.Permissions)
		if len(invalidPerms) > 0 {
			return nil, cerrors.NewBadRequestError("invalid permissions: " + strings.Join(invalidPerms, ", "))
		}

		// Only permissions being ADDED to the role require the caller to hold
		// them; the role's current permissions are resolved lazily so root
		// callers skip the Casbin read entirely.
		err := s.authorizeGrant(ctx, req.Permissions, func() []string {
			return s.casbinClient.GetRolePermissions(roleID)
		})
		if err != nil {
			return nil, err
		}
	}

	// Run the find→modify→update sequence inside a single transaction so a
	// failure at any step rolls the role row back and concurrent writers cannot
	// interleave between the read and the write.
	var adminRole *models.AdminRole
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Lock the row for the read→modify→save sequence: a plain SELECT takes
		// no lock under MVCC, so without FOR UPDATE a concurrent writer could
		// interleave and have its columns reverted by this full-row save.
		var err error
		adminRole, err = s.adminRoleRepo.FindByIDForUpdate(txCtx, roleID)
		if err != nil {
			return err
		}

		// Update fields
		if req.Name != nil {
			adminRole.Name = *req.Name
		}
		if req.Description != nil {
			adminRole.Description = *req.Description
		}

		// Save to database
		return s.adminRoleRepo.Update(txCtx, adminRole)
	})
	if err != nil {
		return nil, err
	}

	// Sync permissions in Casbin only after the DB transaction has committed.
	//
	// Residual atomicity gap: Casbin policies are persisted through the casbin
	// client's own GORM adapter, which does not join the transaction context,
	// so the role update and the policy sync can never be fully atomic. By
	// ordering the DB commit first, a Casbin failure leaves the role row
	// updated but its permissions unchanged — we log loudly and surface the
	// error so the caller can retry the permission sync.
	if req.Permissions != nil {
		if err := s.casbinClient.SetRolePermissions(adminRole.ID, req.Permissions); err != nil {
			logger.Ctx(ctx, zap.Uint("role_id", roleID)).Error(
				"CRITICAL: admin role updated in DB but casbin permission sync failed; permissions are stale",
				zap.Strings("requested_permissions", req.Permissions),
				zap.Error(err),
			)
			return nil, cerrors.NewInternalServerError("failed to update role permissions", err)
		}
	}

	// Get permissions for response
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, adminRole.ID, adminRole.Name)

	return adminRole, nil
}

// Delete implements AdminRoleService.
func (s *adminRoleService) Delete(ctx context.Context, roleID uint) error {
	// Run the find→guard→delete sequence inside a single transaction so a
	// failure at any step rolls everything back and the assigned-users check
	// cannot race with the delete.
	var adminRole *models.AdminRole
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Find and lock the role row. userService.AssignAdminRole locks the
		// same row before inserting an assignment, so the check-then-act pair
		// below (count users → delete) cannot race a concurrent assignment:
		// either the assignment commits first (and the count sees it), or this
		// delete commits first (and the assignment's locked read finds the
		// role gone).
		var err error
		adminRole, err = s.adminRoleRepo.FindByIDForUpdate(txCtx, roleID)
		if err != nil {
			return err
		}

		// Check if any users have this role
		userCount, err := s.adminRoleRepo.CountUsersWithRole(txCtx, roleID)
		if err != nil {
			return err
		}
		if userCount > 0 {
			return cerrors.NewBadRequestError("cannot delete role that is assigned to users")
		}

		// Delete from database
		return s.adminRoleRepo.Delete(txCtx, adminRole)
	})
	if err != nil {
		return err
	}

	// Delete permissions from Casbin only after the DB transaction has
	// committed, so a Casbin failure can never wipe policies for a role that
	// still exists.
	//
	// Residual atomicity gap: Casbin policies are persisted through the casbin
	// client's own GORM adapter, which does not join the transaction context,
	// so the role delete and the policy cleanup can never be fully atomic. If
	// the cleanup fails, orphaned policies remain for the deleted role ID —
	// they are inert (no user can hold the deleted role, the assigned-users
	// guard ran inside the transaction), so we log loudly and still report
	// success rather than fail an operation whose primary effect is committed.
	if err := s.casbinClient.DeleteRole(roleID); err != nil {
		logger.Ctx(ctx, zap.Uint("role_id", roleID)).Error(
			"CRITICAL: admin role deleted from DB but casbin policy cleanup failed; orphaned policies remain",
			zap.Error(err),
		)
	}

	// Create audit log
	s.createLog(ctx, models.LogActionDelete, roleID, adminRole.Name)

	return nil
}

// FindByID implements AdminRoleService.
func (s *adminRoleService) FindByID(ctx context.Context, roleID uint) (*models.AdminRole, error) {
	adminRole, err := s.adminRoleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	// Get permissions from Casbin
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	return adminRole, nil
}

// GetAllPermissions returns all available permissions for frontend
func (s *adminRoleService) GetAllPermissions(_ context.Context) map[string][]map[string]string {
	return permissions.GetPermissionsForFrontend()
}

// validatePermissions checks if all provided permissions are valid
func (s *adminRoleService) validatePermissions(perms []string) []string {
	var invalidPerms []string
	for _, perm := range perms {
		if !permissions.IsValidPermission(perm) {
			invalidPerms = append(invalidPerms, perm)
		}
	}
	return invalidPerms
}

// authorizeGrant rejects a role change unless the caller holds every
// permission being newly granted to the target role. Only additions relative
// to the role's current permissions require authority: keeping or removing
// existing grants stays allowed, so a limited admin can still maintain a role
// broader than their own. Without this check, anyone with admin_role:create /
// admin_role:update could mint roles carrying permissions they were never
// granted and escalate within the admin tier. Root bypasses the check
// (mirroring CheckPermissionWithRoot); a missing caller identity fails closed.
// current is resolved lazily so root callers skip the Casbin read.
func (s *adminRoleService) authorizeGrant(ctx context.Context, requested []string, current func() []string) error {
	if len(requested) == 0 {
		return nil
	}

	role, ok := utils.GetRoleFromContext(ctx)
	if !ok {
		return cerrors.NewForbiddenError("cannot grant permissions without an authenticated caller")
	}
	if role == models.UserRoleRoot.ToString() {
		return nil
	}

	currentSet := make(map[string]struct{})
	for _, perm := range current() {
		currentSet[perm] = struct{}{}
	}

	adminRoleID := utils.GetAdminRoleIDFromContext(ctx)
	var denied []string
	for _, perm := range requested {
		if _, held := currentSet[perm]; held {
			continue
		}
		allowed, err := s.casbinClient.CheckPermissionWithRoot(role, adminRoleID, perm)
		if err != nil {
			return cerrors.NewInternalServerError("failed to verify caller permissions", err)
		}
		if !allowed {
			denied = append(denied, perm)
		}
	}
	if len(denied) > 0 {
		return cerrors.NewForbiddenError("cannot grant permissions you do not hold: " + strings.Join(denied, ", "))
	}
	return nil
}

// createLog creates an audit log entry for admin role operations
func (s *adminRoleService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	audit.RecordAction(ctx, s.logRepository, action, models.LogEntityTypeAdminRole, entityID, "admin role", entityName)
}
