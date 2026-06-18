// Package service contains the admin-role module business logic.
package service

import (
	"context"
	"fmt"
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

	"go.uber.org/zap"
)

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
	// Run the find→modify→update sequence inside a single transaction so a
	// failure at any step rolls the role row back and concurrent writers cannot
	// interleave between the read and the write.
	var adminRole *models.AdminRole
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Find existing role
		var err error
		adminRole, err = s.adminRoleRepo.FindByID(txCtx, roleID)
		if err != nil {
			return err
		}

		// Validate permissions if provided
		if req.Permissions != nil {
			invalidPerms := s.validatePermissions(req.Permissions)
			if len(invalidPerms) > 0 {
				return cerrors.NewBadRequestError("invalid permissions: " + strings.Join(invalidPerms, ", "))
			}
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
		// Find existing role
		var err error
		adminRole, err = s.adminRoleRepo.FindByID(txCtx, roleID)
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

// createLog creates an audit log entry for admin role operations
func (s *adminRoleService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	userName := audit.UserName(ctx)
	var message string
	switch action {
	case models.LogActionCreate:
		message = fmt.Sprintf("%s created admin role: %s", userName, entityName)
	case models.LogActionUpdate:
		message = fmt.Sprintf("%s updated admin role: %s", userName, entityName)
	case models.LogActionDelete:
		message = fmt.Sprintf("%s deleted admin role: %s", userName, entityName)
	default:
		message = fmt.Sprintf("%s performed %s on admin role: %s", userName, action, entityName)
	}

	audit.Record(ctx, s.logRepository, audit.Entry{
		Action:     action,
		EntityType: models.LogEntityTypeAdminRole,
		EntityID:   entityID,
		Message:    message,
	})
}
