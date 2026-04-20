package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepo "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	rtokenrepo "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	"github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService defines the interface for user service operations
type UserService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.User, response.Meta, error)
	Update(ctx context.Context, userId uint, req *dto.UserUpdateRequest) (*models.User, error)
	FindById(ctx context.Context, userId uint) (*models.User, error)
	AssignAdminRole(ctx context.Context, userId uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error)
	ChangePassword(ctx context.Context, userId uint, req *dto.ChangeAdminPasswordRequest) error
}

// userService implements the UserService interface
type userService struct {
	userRepository      repository.UserRepository
	adminRoleRepository adminrolerepo.AdminRoleRepository
	refreshTokenRepo    rtokenrepo.RefreshTokenRepository
	logRepository       logRepository.LogRepository
	casbinClient        casbin.Client
}

// NewUserService creates a new instance of UserService
func NewUserService(
	userRepository repository.UserRepository,
	adminRoleRepository adminrolerepo.AdminRoleRepository,
	refreshTokenRepo rtokenrepo.RefreshTokenRepository,
	logRepository logRepository.LogRepository,
	casbinClient casbin.Client,
) UserService {
	return &userService{
		userRepository:      userRepository,
		adminRoleRepository: adminRoleRepository,
		refreshTokenRepo:    refreshTokenRepo,
		logRepository:       logRepository,
		casbinClient:        casbinClient,
	}
}

// Index implements UserService.
func (s *userService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Fetching users with pagination",
		zap.String("request_id", requestID),
		zap.Int("page", pg.GetPage()),
		zap.Int("limit", pg.Limit),
		zap.Int("offset", pg.Offset),
	)

	// Add preloads
	pg.AddCustomScope(func(db *gorm.DB) *gorm.DB {
		return db.Preload("AdminRole")
	})

	users, err := s.userRepository.FindAll(ctx, pg)
	if err != nil {
		logger.Error("Failed to fetch users",
			zap.String("request_id", requestID),
			zap.Int("page", pg.GetPage()),
			zap.Error(err),
		)
		return users, response.Meta{}, err
	}

	count, err := s.userRepository.Count(ctx, pg)
	if err != nil {
		logger.Error("Failed to count users",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return users, response.Meta{}, err
	}

	logger.Info("Users fetched successfully",
		zap.String("request_id", requestID),
		zap.Int("returned_count", len(users)),
		zap.Int64("total_count", count),
	)

	return users, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Update implements UserService.
func (s *userService) Update(ctx context.Context, userId uint, req *dto.UserUpdateRequest) (*models.User, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Updating user",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
	)

	user, err := s.userRepository.FindById(ctx, userId)
	if err != nil {
		logger.Error("Failed to find user for update",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)
		return user, err
	}

	err = copier.Copy(&user, &req)
	if err != nil {
		logger.Error("Failed to copy user data",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)
		return user, err
	}

	err = s.userRepository.Update(ctx, user)
	if err != nil {
		logger.Error("Failed to update user",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)
		return user, err
	}

	logger.Info("User updated successfully",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
	)

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, user.ID, user.Name)

	return user, nil
}

// FindById implements UserService.
func (s *userService) FindById(ctx context.Context, userId uint) (*models.User, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Finding user by ID",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
	)

	user, err := s.userRepository.FindById(ctx, userId, generated.User.AdminRole)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			logger.Error("Failed to find user by ID",
				zap.String("request_id", requestID),
				zap.Uint("user_id", userId),
				zap.Error(err),
			)
		}
		return user, err
	}

	if user.AdminRole != nil {
		user.AdminRole.Permissions = s.casbinClient.GetRolePermissions(user.AdminRole.ID)
	}

	logger.Debug("Found user by ID successfully",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
	)

	return user, nil
}

// AssignAdminRole assigns an admin role to a user
func (s *userService) AssignAdminRole(ctx context.Context, userId uint, req *dto.UserAssignAdminRoleRequest) (*models.User, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Assigning admin role to user",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
		zap.Uint("admin_role_id", req.AdminRoleID),
	)

	// Find user
	user, err := s.userRepository.FindById(ctx, userId)
	if err != nil {
		logger.Error("Failed to find user",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)
		return nil, err
	}

	// Prevent modifying root users
	if user.Role == models.UserRoleRoot {
		logger.Warn("Attempted to modify root user",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
		)
		return nil, cerrors.NewForbiddenError("cannot modify root user")
	}

	// Assign admin role and set role to admin
	user.AdminRoleID = &req.AdminRoleID

	err = s.userRepository.Update(ctx, user)
	if err != nil {
		logger.Error("Failed to assign admin role",
			zap.String("request_id", requestID),
			zap.Uint("user_id", userId),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Info("Admin role assigned successfully",
		zap.String("request_id", requestID),
		zap.Uint("user_id", userId),
		zap.Uint("admin_role_id", req.AdminRoleID),
	)

	// Create audit log
	s.createLog(ctx, models.LogActionUpdate, user.ID, user.Name)

	return user, nil
}

// ChangePassword allows root to change another admin's password
func (s *userService) ChangePassword(ctx context.Context, userId uint, req *dto.ChangeAdminPasswordRequest) error {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Root changing admin password",
		zap.String("request_id", requestID),
		zap.Uint("target_user_id", userId),
	)

	user, err := s.userRepository.FindById(ctx, userId)
	if err != nil {
		logger.Error("Failed to find target user",
			zap.String("request_id", requestID),
			zap.Uint("target_user_id", userId),
			zap.Error(err),
		)
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
		logger.Error("Failed to hash password",
			zap.String("request_id", requestID),
			zap.Uint("target_user_id", userId),
			zap.Error(err),
		)
		return cerrors.NewInternalServerError("failed to process new password", err)
	}

	user.Password = string(hashedPassword)
	if err := s.userRepository.Update(ctx, user); err != nil {
		logger.Error("Failed to update password",
			zap.String("request_id", requestID),
			zap.Uint("target_user_id", userId),
			zap.Error(err),
		)
		return err
	}

	// Revoke all refresh tokens for the target user
	if err := s.refreshTokenRepo.RevokeAllByUserID(ctx, userId); err != nil {
		logger.Error("Failed to revoke tokens after password change",
			zap.String("request_id", requestID),
			zap.Uint("target_user_id", userId),
			zap.Error(err),
		)
	}

	logger.Info("Admin password changed by root",
		zap.String("request_id", requestID),
		zap.Uint("target_user_id", userId),
	)

	s.createLog(ctx, models.LogActionChangePassword, user.ID, user.Name)

	return nil
}

// createLog creates an audit log entry for user operations
func (s *userService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	var userID *uint
	if id, ok := utils.GetUserIDFromContext(ctx); ok {
		userID = &id
	}

	userName, _ := utils.GetUserNameFromContext(ctx)
	if userName == "" {
		userName = "Unknown"
	}

	var message string
	switch action {
	case models.LogActionUpdate:
		message = fmt.Sprintf("%s updated user: %s", userName, entityName)
	default:
		message = fmt.Sprintf("%s performed %s on user: %s", userName, action, entityName)
	}

	log := &models.Log{
		UserID:     userID,
		Action:     action,
		EntityType: models.LogEntityTypeUser,
		EntityID:   entityID,
		Message:    message,
	}

	go func() {
		if err := s.logRepository.Create(context.Background(), log); err != nil {
			logger.Error("Failed to create audit log",
				zap.String("entity_type", models.LogEntityTypeUser),
				zap.Uint("entity_id", entityID),
				zap.String("action", string(action)),
				zap.Error(err),
			)
		}
	}()
}
