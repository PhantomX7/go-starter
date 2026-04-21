// internal/modules/admin_role/service/service.go
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepo "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type AdminRoleService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.AdminRole, response.Meta, error)
	Create(ctx context.Context, req *dto.CreateAdminRoleRequest) (*models.AdminRole, error)
	Update(ctx context.Context, roleID uint, req *dto.UpdateAdminRoleRequest) (*models.AdminRole, error)
	Delete(ctx context.Context, roleID uint) error
	FindById(ctx context.Context, roleID uint) (*models.AdminRole, error)
	GetAllPermissions(ctx context.Context) map[string][]map[string]string
}

type adminRoleService struct {
	adminRoleRepo adminrolerepo.AdminRoleRepository
	logRepository logRepository.LogRepository
	casbinClient  casbin.Client
}

func NewAdminRoleService(
	adminRoleRepo adminrolerepo.AdminRoleRepository,
	logRepository logRepository.LogRepository,
	casbinClient casbin.Client,
) AdminRoleService {
	return &adminRoleService{
		adminRoleRepo: adminRoleRepo,
		logRepository: logRepository,
		casbinClient:  casbinClient,
	}
}

// Index implements AdminRoleService.
func (s *adminRoleService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, response.Meta, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Fetching admin roles with pagination",
		zap.String("request_id", requestID),
		zap.Int("page", pg.GetPage()),
		zap.Int("limit", pg.Limit),
		zap.Int("offset", pg.Offset),
	)

	roles, err := s.adminRoleRepo.FindAll(ctx, pg)
	if err != nil {
		logger.Error("Failed to fetch admin roles",
			zap.String("request_id", requestID),
			zap.Int("page", pg.GetPage()),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	count, err := s.adminRoleRepo.Count(ctx, pg)
	if err != nil {
		logger.Error("Failed to count admin roles",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	for _, role := range roles {
		role.Permissions = s.casbinClient.GetRolePermissions(role.ID)
	}

	logger.Info("Admin roles fetched successfully",
		zap.String("request_id", requestID),
		zap.Int("returned_count", len(roles)),
		zap.Int64("total_count", count),
	)

	return roles, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements AdminRoleService.
func (s *adminRoleService) Create(ctx context.Context, req *dto.CreateAdminRoleRequest) (*models.AdminRole, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Creating admin role",
		zap.String("request_id", requestID),
		zap.String("name", req.Name),
		zap.Int("permissions_count", len(req.Permissions)),
	)

	// Validate permissions
	invalidPerms := s.validatePermissions(req.Permissions)
	if len(invalidPerms) > 0 {
		logger.Warn("Invalid permissions provided",
			zap.String("request_id", requestID),
			zap.Strings("invalid_permissions", invalidPerms),
		)
		return nil, cerrors.NewBadRequestError("invalid permissions: " + joinStrings(invalidPerms))
	}

	// Create admin role model
	adminRole := &models.AdminRole{
		IsActive: true,
	}
	if err := copier.Copy(adminRole, req); err != nil {
		logger.Error("Failed to copy admin role data",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, cerrors.NewInternalServerError("failed to process admin role data", err)
	}

	// Save to database
	if err := s.adminRoleRepo.Create(ctx, adminRole); err != nil {
		logger.Error("Failed to create admin role",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, err
	}

	// Add permissions to Casbin
	if err := s.casbinClient.AddRolePermissions(adminRole.ID, req.Permissions); err != nil {
		logger.Error("Failed to add permissions to casbin",
			zap.String("request_id", requestID),
			zap.Uint("role_id", adminRole.ID),
			zap.Error(err),
		)
		// Rollback: delete the created role
		_ = s.adminRoleRepo.Delete(ctx, adminRole)
		return nil, cerrors.NewInternalServerError("failed to set role permissions", err)
	}

	logger.Info("Admin role created successfully",
		zap.String("request_id", requestID),
		zap.Uint("role_id", adminRole.ID),
		zap.String("name", adminRole.Name),
	)

	// Get permissions for response
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	// Create audit log
	s.createLog(ctx, models.LogActionCreate, adminRole.ID, adminRole.Name)

	return adminRole, nil
}

// Update implements AdminRoleService.
func (s *adminRoleService) Update(ctx context.Context, roleID uint, req *dto.UpdateAdminRoleRequest) (*models.AdminRole, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Updating admin role",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	// Find existing role
	adminRole, err := s.adminRoleRepo.FindById(ctx, roleID)
	if err != nil {
		logger.Error("Failed to find admin role for update",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return nil, err
	}

	// Validate permissions if provided
	if req.Permissions != nil {
		invalidPerms := s.validatePermissions(req.Permissions)
		if len(invalidPerms) > 0 {
			logger.Warn("Invalid permissions provided",
				zap.String("request_id", requestID),
				zap.Strings("invalid_permissions", invalidPerms),
			)
			return nil, cerrors.NewBadRequestError("invalid permissions: " + joinStrings(invalidPerms))
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
	if err := s.adminRoleRepo.Update(ctx, adminRole); err != nil {
		logger.Error("Failed to update admin role",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return nil, err
	}

	// Update permissions in Casbin if provided
	if req.Permissions != nil {
		if err := s.casbinClient.SetRolePermissions(adminRole.ID, req.Permissions); err != nil {
			logger.Error("Failed to update permissions in casbin",
				zap.String("request_id", requestID),
				zap.Uint("role_id", roleID),
				zap.Error(err),
			)
			return nil, cerrors.NewInternalServerError("failed to update role permissions", err)
		}
	}

	logger.Info("Admin role updated successfully",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	// Get permissions for response
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, adminRole.ID, adminRole.Name)

	return adminRole, nil
}

// Delete implements AdminRoleService.
func (s *adminRoleService) Delete(ctx context.Context, roleID uint) error {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Deleting admin role",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	// Find existing role
	adminRole, err := s.adminRoleRepo.FindById(ctx, roleID)
	if err != nil {
		logger.Error("Failed to find admin role for deletion",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return err
	}

	// Check if any users have this role
	userCount, err := s.adminRoleRepo.CountUsersWithRole(ctx, roleID)
	if err != nil {
		logger.Error("Failed to count users with role",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return err
	}
	if userCount > 0 {
		return cerrors.NewBadRequestError("cannot delete role that is assigned to users")
	}

	// Delete permissions from Casbin
	if err := s.casbinClient.DeleteRole(roleID); err != nil {
		logger.Error("Failed to delete permissions from casbin",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return cerrors.NewInternalServerError("failed to delete role permissions", err)
	}

	// Delete from database
	if err := s.adminRoleRepo.Delete(ctx, adminRole); err != nil {
		logger.Error("Failed to delete admin role",
			zap.String("request_id", requestID),
			zap.Uint("role_id", roleID),
			zap.Error(err),
		)
		return err
	}

	logger.Info("Admin role deleted successfully",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	// Create audit log
	s.createLog(ctx, models.LogActionDelete, roleID, adminRole.Name)

	return nil
}

// FindById implements AdminRoleService.
func (s *adminRoleService) FindById(ctx context.Context, roleID uint) (*models.AdminRole, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Finding admin role by ID",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	adminRole, err := s.adminRoleRepo.FindById(ctx, roleID)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			logger.Error("Failed to find admin role by ID",
				zap.String("request_id", requestID),
				zap.Uint("role_id", roleID),
				zap.Error(err),
			)
		}
		return nil, err
	}

	// Get permissions from Casbin
	rolePermissions := s.casbinClient.GetRolePermissions(adminRole.ID)

	adminRole.Permissions = rolePermissions

	logger.Debug("Found admin role by ID successfully",
		zap.String("request_id", requestID),
		zap.Uint("role_id", roleID),
	)

	return adminRole, nil
}

// GetAllPermissions returns all available permissions for frontend
func (s *adminRoleService) GetAllPermissions(ctx context.Context) map[string][]map[string]string {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Getting all available permissions",
		zap.String("request_id", requestID),
	)

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

// joinStrings joins string slice with comma
func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
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
