// Package service contains the user business logic.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepo "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	logrepo "github.com/PhantomX7/athleton/internal/modules/log/repository"
	rtokenrepo "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	"github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/constants/security"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService defines the interface for user service operations
type UserService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.User, response.Meta, error)
	Create(ctx context.Context, req *dto.AdminUserCreateRequest) (*models.User, error)
	Update(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error)
	FindByID(ctx context.Context, userID uint) (*models.User, error)
	AssignAdminRole(ctx context.Context, userID uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error)
	ChangePassword(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error
}

// userService implements the UserService interface
type userService struct {
	userRepository   repository.UserRepository
	adminRoleRepo    adminrolerepo.AdminRoleRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
	logRepository    logrepo.LogRepository
	casbinClient     casbin.Client
	txManager        transaction_manager.TransactionManager
	log              *zap.Logger
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepository repository.UserRepository,
	adminRoleRepo adminrolerepo.AdminRoleRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
	logRepository logrepo.LogRepository,
	casbinClient casbin.Client,
	txManager transaction_manager.TransactionManager,
	log *zap.Logger,
) UserService {
	return &userService{
		userRepository:   userRepository,
		adminRoleRepo:    adminRoleRepo,
		refreshTokenRepo: refreshTokenRepo,
		logRepository:    logRepository,
		casbinClient:     casbinClient,
		txManager:        txManager,
		log:              log,
	}
}

// callerHasPermission reports whether the authenticated caller holds perm.
// Root bypasses; a context without auth values counts as holding nothing
// (fail closed) — the /admin group middleware guarantees values in practice.
func (s *userService) callerHasPermission(ctx context.Context, perm permissions.Permission) (bool, error) {
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		// Deliberately swallow the error: a context without auth values means
		// "no grants", not an infrastructure failure.
		return false, nil //nolint:nilerr // fail closed on missing auth context
	}
	return s.casbinClient.CheckPermissionWithRoot(values.Role, values.AdminRoleID, perm.String())
}

// requireAdminUserGrant enforces the stronger admin_user:* grant when the
// operation targets an admin-type account: managing admin accounts is more
// privileged than managing regular users, so user:* alone is not enough.
func (s *userService) requireAdminUserGrant(ctx context.Context, target *models.User, perm permissions.Permission) error {
	if !target.Role.IsAdminType() {
		return nil
	}
	allowed, err := s.callerHasPermission(ctx, perm)
	if err != nil {
		return cerrors.NewInternalServerError("failed to verify permissions", err)
	}
	if !allowed {
		logger.CtxWith(ctx, s.log, zap.Uint("target_user_id", target.ID), zap.String("permission", perm.String())).
			Warn("Denied admin-account access without admin_user grant")
		return cerrors.NewForbiddenError("insufficient permissions to manage admin accounts")
	}
	return nil
}

// Index implements UserService.
func (s *userService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
	// Callers without admin_user:read only see regular accounts: admin and
	// root rows are filtered out of the listing entirely.
	canSeeAdmins, err := s.callerHasPermission(ctx, permissions.AdminUserRead)
	if err != nil {
		return nil, response.Meta{}, cerrors.NewInternalServerError("failed to verify permissions", err)
	}
	if !canSeeAdmins {
		pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
			return db.Where("role = ?", models.UserRoleUser.ToString())
		})
	}

	// Add preloads
	pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
		return db.Preload("AdminRole")
	})

	users, err := s.userRepository.FindAll(ctx, pg)
	if err != nil {
		return users, response.Meta{}, err
	}

	count, err := s.userRepository.Count(ctx, pg)
	if err != nil {
		return users, response.Meta{}, err
	}

	return users, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements UserService: an admin (holding admin_user:create) creates
// a new admin account directly. The role is hardcoded to "admin" — no request
// field can produce a root account — and PasswordChangedAt stays nil so the
// must-change-default-password gate forces the new admin to rotate the
// creator-chosen password on first login.
func (s *userService) Create(ctx context.Context, req *dto.AdminUserCreateRequest) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), security.BcryptCost)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to process password", err)
	}

	user := &models.User{
		Username:    req.Username,
		Name:        req.Name,
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Phone:       strings.TrimSpace(req.Phone),
		IsActive:    true,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &req.AdminRoleID,
		Password:    string(hashedPassword),
	}

	err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Lock the target admin-role row: this serializes with the role-delete
		// flow (which locks the same row before its "no users assigned" check)
		// and re-verifies the role exists inside the transaction instead of
		// trusting the request-validation lookup that ran outside it.
		if _, err := s.adminRoleRepo.FindByIDForUpdate(txCtx, req.AdminRoleID); err != nil {
			return err
		}

		return s.userRepository.Create(txCtx, user)
	})
	if err != nil {
		return nil, err
	}

	s.createLog(ctx, models.LogActionCreate, user.ID, user.Name)

	return user, nil
}

// Update implements UserService.
func (s *userService) Update(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error) {
	// Run the find→guard→modify→update sequence inside a single transaction
	// with the user row locked: the repository persists with a full-row Save,
	// so without the lock a concurrent user-mutating flow (AssignAdminRole,
	// ChangePassword) could interleave between the read and the write and have
	// its columns silently reverted.
	var user *models.User
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		var err error
		user, err = s.userRepository.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return err
		}

		// Prevent modifying root users — same guard as AssignAdminRole and
		// ChangePassword, so Update cannot be used to demote or rename root.
		if user.Role == models.UserRoleRoot {
			logger.CtxWith(ctx, s.log, zap.Uint("user_id", userID)).Warn("Attempted to modify root user")
			return cerrors.NewForbiddenError("cannot modify root user")
		}

		// Modifying another admin account requires the stronger grant.
		if err := s.requireAdminUserGrant(txCtx, user, permissions.AdminUserUpdate); err != nil {
			return err
		}

		// Pointer fields: an omitted field (nil) keeps its current value — PATCH semantics.
		if req.Name != nil {
			user.Name = *req.Name
		}
		if req.Role != nil {
			user.Role = models.UserRole(*req.Role)
			// Demoting away from an admin-type role must clear the admin-role
			// assignment in the same write so no dangling AdminRoleID remains.
			if !user.Role.IsAdminType() {
				user.AdminRoleID = nil
			}
		}

		return s.userRepository.Update(txCtx, user)
	})
	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, user.ID, user.Name)

	return user, nil
}

// FindByID implements UserService.
func (s *userService) FindByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepository.FindByID(ctx, userID, generated.User.AdminRole)
	if err != nil {
		return nil, err
	}

	// Admin and root accounts are only visible with the stronger grant.
	if err := s.requireAdminUserGrant(ctx, user, permissions.AdminUserRead); err != nil {
		return nil, err
	}

	if user.AdminRole != nil {
		user.AdminRole.Permissions = s.casbinClient.GetRolePermissions(user.AdminRole.ID)
	}

	return user, nil
}

// AssignAdminRole assigns an admin role to a user and promotes the account to
// the "admin" role in the same write — Casbin's CheckPermissionWithRoot only
// consults AdminRoleID when Role == "admin", so setting only one of the two
// fields would grant nothing. There is no separate unassign endpoint: demotion
// goes through Update (role "user"), which clears AdminRoleID in the same
// write so both fields always change together.
func (s *userService) AssignAdminRole(ctx context.Context, userID uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error) {
	// Run the find→check→assign→update sequence inside a single transaction so
	// a failure at any step rolls everything back and concurrent writers cannot
	// interleave between the read and the role assignment.
	var user *models.User
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Lock the target admin-role row first. This serializes with
		// adminRoleService.Delete, which locks the same row before its
		// "no users assigned" check: either this assignment commits first (and
		// the delete sees the user), or the delete commits first (and this
		// locked read finds the role gone and fails). It also re-verifies the
		// role's existence inside the transaction instead of trusting the
		// request-validation lookup that ran outside it.
		if _, err := s.adminRoleRepo.FindByIDForUpdate(txCtx, req.AdminRoleID); err != nil {
			return err
		}

		// Find and lock the user row so the full-row Save below cannot race
		// other user-mutating flows.
		var err error
		user, err = s.userRepository.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return err
		}

		// Prevent modifying root users
		if user.Role == models.UserRoleRoot {
			logger.CtxWith(ctx, s.log, zap.Uint("user_id", userID)).Warn("Attempted to modify root user")
			return cerrors.NewForbiddenError("cannot modify root user")
		}

		// Assign admin role and set role to admin
		user.AdminRoleID = &req.AdminRoleID
		user.Role = models.UserRoleAdmin

		return s.userRepository.Update(txCtx, user)
	})
	if err != nil {
		return nil, err
	}

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, user.ID, user.Name)

	return user, nil
}

// ChangePassword allows root to change another admin's password
func (s *userService) ChangePassword(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error {
	// Run the find→guard→hash→update→revoke sequence inside a single
	// transaction with the user row locked: the repository persists with a
	// full-row Save, so reading outside the transaction would let a concurrent
	// Update/AssignAdminRole interleave and have its columns silently
	// reverted. The same transaction also revokes all refresh tokens, so a
	// partial failure cannot leave the password changed while stale sessions
	// remain valid (or vice versa).
	var user *models.User
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		var err error
		user, err = s.userRepository.FindByIDForUpdate(txCtx, userID)
		if err != nil {
			return err
		}

		// Only allow changing password of admin users
		if !user.Role.IsAdminType() {
			return cerrors.NewBadRequestError("can only change password of admin users")
		}

		// Prevent changing another root user's password
		if user.Role == models.UserRoleRoot {
			return cerrors.NewForbiddenError("cannot change root user password")
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), security.BcryptCost)
		if err != nil {
			return cerrors.NewInternalServerError("failed to process new password", err)
		}

		user.Password = string(hashedPassword)
		// Clears the must-change-default-password gate for the target account.
		now := time.Now()
		user.PasswordChangedAt = &now

		if err := s.userRepository.Update(txCtx, user); err != nil {
			return err
		}

		// Revoke all refresh tokens for the target user
		return s.refreshTokenRepo.RevokeAllByUserID(txCtx, userID)
	})
	if err != nil {
		return err
	}

	s.createLog(ctx, models.LogActionChangePassword, user.ID, user.Name)

	return nil
}

// createLog creates an audit log entry for user operations
func (s *userService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	userName := audit.UserName(ctx)
	var message string
	switch action {
	case models.LogActionUpdate:
		message = fmt.Sprintf("%s updated user: %s", userName, entityName)
	default:
		message = fmt.Sprintf("%s performed %s on user: %s", userName, action, entityName)
	}

	audit.Record(ctx, s.logRepository, audit.Entry{
		Action:     action,
		EntityType: models.LogEntityTypeUser,
		EntityID:   entityID,
		Message:    message,
	})
}
