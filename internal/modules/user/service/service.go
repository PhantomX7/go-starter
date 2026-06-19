// Package service contains the user business logic.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	logrepo "github.com/PhantomX7/athleton/internal/modules/log/repository"
	rtokenrepo "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	"github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService defines the interface for user service operations
type UserService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.User, response.Meta, error)
	Update(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error)
	FindByID(ctx context.Context, userID uint) (*models.User, error)
	AssignAdminRole(ctx context.Context, userID uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error)
	ChangePassword(ctx context.Context, userID uint, req *dto.ChangeAdminPasswordRequest) error
}

// userService implements the UserService interface
type userService struct {
	userRepository   repository.UserRepository
	refreshTokenRepo rtokenrepo.RefreshTokenRepository
	logRepository    logrepo.LogRepository
	casbinClient     casbin.Client
	txManager        transaction_manager.TransactionManager
	log              *zap.Logger
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepository repository.UserRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
	logRepository logrepo.LogRepository,
	casbinClient casbin.Client,
	txManager transaction_manager.TransactionManager,
	log *zap.Logger,
) UserService {
	return &userService{
		userRepository:   userRepository,
		refreshTokenRepo: refreshTokenRepo,
		logRepository:    logRepository,
		casbinClient:     casbinClient,
		txManager:        txManager,
		log:              log,
	}
}

// Index implements UserService.
func (s *userService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
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

// Update implements UserService.
func (s *userService) Update(ctx context.Context, userID uint, req *dto.UserUpdateRequest) (*models.User, error) {
	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return user, err
	}

	// Pointer fields: an omitted field (nil) keeps its current value — PATCH semantics.
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Role != nil {
		user.Role = models.UserRole(*req.Role)
	}

	if err := s.userRepository.Update(ctx, user); err != nil {
		return user, err
	}

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, user.ID, user.Name)

	return user, nil
}

// FindByID implements UserService.
func (s *userService) FindByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepository.FindByID(ctx, userID, generated.User.AdminRole)
	if err != nil {
		return user, err
	}

	if user.AdminRole != nil {
		user.AdminRole.Permissions = s.casbinClient.GetRolePermissions(user.AdminRole.ID)
	}

	return user, nil
}

// AssignAdminRole assigns an admin role to a user
func (s *userService) AssignAdminRole(ctx context.Context, userID uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error) {
	// Run the find→check→assign→update sequence inside a single transaction so
	// a failure at any step rolls everything back and concurrent writers cannot
	// interleave between the read and the role assignment.
	var user *models.User
	err := s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		// Find user
		var err error
		user, err = s.userRepository.FindByID(txCtx, userID)
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
	user, err := s.userRepository.FindByID(ctx, userID)
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		return cerrors.NewInternalServerError("failed to process new password", err)
	}

	// Update the password and revoke all refresh tokens in one transaction so a
	// partial failure cannot leave the password changed while stale sessions
	// remain valid (or vice versa).
	user.Password = string(hashedPassword)
	// Clears the must-change-default-password gate for the target account.
	now := time.Now()
	user.PasswordChangedAt = &now
	err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
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
