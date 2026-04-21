// Package service contains the auth module business logic.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	userrepo "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost used for password hashing.
	BcryptCost = 12
)

// AuthService defines the interface for auth service operations
type AuthService interface {
	GetMe(ctx context.Context) (*dto.MeResponse, error)
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error)
	ChangePassword(ctx context.Context, req *dto.ChangePasswordRequest) error
	Logout(ctx context.Context, req *dto.LogoutRequest) error
}

type authService struct {
	userRepo      userrepo.UserRepository
	logRepository logRepository.LogRepository
	authJWT       *authjwt.AuthJWT
	casbinClient  casbin.Client
	txManager     transaction_manager.TransactionManager
}

// NewAuthService builds the auth service from its dependencies.
func NewAuthService(
	userRepo userrepo.UserRepository,
	logRepository logRepository.LogRepository,
	authJWT *authjwt.AuthJWT,
	casbinClient casbin.Client,
	txManager transaction_manager.TransactionManager,
) AuthService {
	return &authService{
		userRepo:      userRepo,
		logRepository: logRepository,
		authJWT:       authJWT,
		casbinClient:  casbinClient,
		txManager:     txManager,
	}
}

// GetMe retrieves the authenticated user's profile
func (s *authService) GetMe(ctx context.Context) (*dto.MeResponse, error) {
	requestID := utils.GetRequestIDFromContext(ctx)
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		logger.Error("Failed to get user from context", zap.String("request_id", requestID), zap.Error(err))
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, values.UserID, generated.User.AdminRole)
	if err != nil {
		logger.Error("Failed to find user", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID), zap.Error(err))
		return nil, err
	}

	if !user.IsActive {
		logger.Warn("Inactive user access attempt", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))
		return nil, cerrors.NewForbiddenError("user account is inactive")
	}

	if user.AdminRoleID != nil {
		// Get permissions for response
		rolePermissions := s.casbinClient.GetRolePermissions(*user.AdminRoleID)
		user.AdminRole.Permissions = rolePermissions
	}

	return &dto.MeResponse{UserResponse: *user.ToResponse()}, nil
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	requestID := utils.GetRequestIDFromContext(ctx)
	logger.Info("User registration initiated", zap.String("request_id", requestID))

	// Normalize inputs
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	// Create user model
	user := &models.User{Role: models.UserRoleUser, IsActive: true, Username: req.Email}
	if err := copier.Copy(&user, &req); err != nil {
		logger.Error("Failed to copy user data", zap.String("request_id", requestID), zap.Error(err))
		return nil, cerrors.NewInternalServerError("failed to process user data", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), BcryptCost)
	if err != nil {
		logger.Error("Failed to hash password", zap.String("request_id", requestID), zap.Error(err))
		return nil, cerrors.NewInternalServerError("failed to process password", err)
	}
	user.Password = string(hashedPassword)

	// Create user and tokens in transaction
	var authResponse *dto.AuthResponse
	err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Create(txCtx, user); err != nil {
			return err
		}

		// Generate tokens using AuthJWT
		authResponse, err = s.authJWT.GenerateTokensForUser(txCtx, user)
		return err
	})
	if err != nil {
		logger.Error("Failed to register user", zap.String("request_id", requestID), zap.Error(err))
		return nil, err
	}

	logger.Info("User registered successfully", zap.String("request_id", requestID), zap.Uint("user_id", user.ID))
	return authResponse, nil
}

// Refresh validates a refresh token and issues new tokens
func (s *authService) Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
	requestID := utils.GetRequestIDFromContext(ctx)
	logger.Info("Token refresh initiated", zap.String("request_id", requestID))

	authResponse, err := s.authJWT.ValidateAndRotateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		logger.Warn("Token refresh failed", zap.String("request_id", requestID), zap.Error(err))
		return nil, err
	}

	logger.Info("Token refresh successful", zap.String("request_id", requestID))
	return authResponse, nil
}

// ChangePassword updates the user's password
func (s *authService) ChangePassword(ctx context.Context, req *dto.ChangePasswordRequest) error {
	requestID := utils.GetRequestIDFromContext(ctx)
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		logger.Error("Failed to get user from context", zap.String("request_id", requestID), zap.Error(err))
		return err
	}

	logger.Info("Password change initiated", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))

	user, err := s.userRepo.FindByID(ctx, values.UserID)
	if err != nil {
		logger.Error("Failed to find user", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID), zap.Error(err))
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		logger.Warn("Password change failed - incorrect password", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))
		return cerrors.NewBadRequestError("current password is incorrect")
	}

	// Ensure new password is different
	if req.OldPassword == req.NewPassword {
		logger.Warn("Password change failed - same password", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))
		return cerrors.NewBadRequestError("new password must be different from current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), BcryptCost)
	if err != nil {
		logger.Error("Failed to hash password", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID), zap.Error(err))
		return cerrors.NewInternalServerError("failed to process new password", err)
	}

	// Update password and revoke tokens in transaction
	err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		user.Password = string(hashedPassword)
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return err
		}
		return s.authJWT.RevokeAllUserTokensExcept(txCtx, user.ID, req.ExceptToken)
	})
	if err != nil {
		logger.Error("Failed to change password", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID), zap.Error(err))
		return err
	}

	logger.Info("Password changed successfully", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))

	// Create audit log for admin users only
	if user.Role == models.UserRoleAdmin {
		s.createLog(ctx, models.LogActionChangePassword, user.ID, user.Name)
	}

	return nil
}

// Logout revokes a specific refresh token
func (s *authService) Logout(ctx context.Context, req *dto.LogoutRequest) error {
	requestID := utils.GetRequestIDFromContext(ctx)
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		logger.Error("Failed to get user from context", zap.String("request_id", requestID), zap.Error(err))
		return err
	}

	logger.Info("Logout initiated", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))

	err = s.authJWT.RevokeRefreshToken(ctx, req.RefreshToken, values.UserID)
	if err != nil {
		logger.Error("Failed to revoke token", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID), zap.Error(err))
		return err
	}

	logger.Info("Logout successful", zap.String("request_id", requestID), zap.Uint("user_id", values.UserID))
	return nil
}

// createLog creates an audit log entry for auth operations (admin only)
func (s *authService) createLog(ctx context.Context, action models.LogAction, entityID uint, entityName string) {
	userName := audit.UserName(ctx)
	var message string
	switch action {
	case models.LogActionChangePassword:
		message = fmt.Sprintf("%s changed password for: %s", userName, entityName)
	case models.LogActionLogin:
		message = fmt.Sprintf("%s logged in", entityName)
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
