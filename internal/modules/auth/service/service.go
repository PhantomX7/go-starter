// Package service contains the auth module business logic.
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
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	logRepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	userrepo "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/constants/security"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

//go:generate go tool moq -out mocks/mock.go -pkg mocks -fmt goimports . AuthService

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
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, values.UserID, generated.User.AdminRole)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		logger.Ctx(ctx, zap.Uint("user_id", values.UserID)).Warn("Inactive user access attempt")
		return nil, cerrors.NewForbiddenError("user account is inactive")
	}

	// AdminRole can be nil even when AdminRoleID is set (soft-deleted role,
	// seed drift); degrade to a role-less profile instead of panicking.
	if user.AdminRoleID != nil && user.AdminRole != nil {
		user.AdminRole.Permissions = s.casbinClient.GetRolePermissions(*user.AdminRoleID)
	}

	return &dto.MeResponse{UserResponse: *user.ToResponse()}, nil
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Normalize inputs
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	// Create user model. Username mirrors the (already normalized) email; the
	// password is set from the hash below, never from the raw request.
	user := &models.User{
		Username:     req.Email,
		Name:         req.Name,
		BusinessName: req.BusinessName,
		Email:        req.Email,
		Phone:        req.Phone,
		Role:         models.UserRoleUser,
		IsActive:     true,
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), security.BcryptCost)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to process password", err)
	}
	user.Password = string(hashedPassword)
	// A self-chosen password at registration counts as changed, so the
	// must-change-default-password gate never fires for self-registered users.
	now := time.Now()
	user.PasswordChangedAt = &now

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
		return nil, err
	}

	logger.Ctx(ctx, zap.Uint("user_id", user.ID)).Info("User registered successfully")
	return authResponse, nil
}

// Refresh validates a refresh token and issues new tokens
func (s *authService) Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
	return s.authJWT.ValidateAndRotateRefreshToken(ctx, req.RefreshToken)
}

// ChangePassword updates the user's password
func (s *authService) ChangePassword(ctx context.Context, req *dto.ChangePasswordRequest) error {
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByID(ctx, values.UserID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		logger.Ctx(ctx, zap.Uint("user_id", values.UserID)).Warn("Password change failed - incorrect current password")
		return cerrors.NewBadRequestError("current password is incorrect")
	}

	// Ensure new password is different
	if req.OldPassword == req.NewPassword {
		return cerrors.NewBadRequestError("new password must be different from current password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), security.BcryptCost)
	if err != nil {
		return cerrors.NewInternalServerError("failed to process new password", err)
	}

	// Update password and revoke tokens in transaction
	err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		user.Password = string(hashedPassword)
		// Clears the must-change-default-password gate for seeded accounts.
		now := time.Now()
		user.PasswordChangedAt = &now
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return err
		}
		return s.authJWT.RevokeAllUserTokensExcept(txCtx, user.ID, req.ExceptToken)
	})
	if err != nil {
		return err
	}

	// Audit every privileged password rotation — root included.
	if user.Role.IsAdminType() {
		s.createLog(ctx, models.LogActionChangePassword, user.ID, user.Name)
	}

	return nil
}

// Logout revokes a specific refresh token
func (s *authService) Logout(ctx context.Context, req *dto.LogoutRequest) error {
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		return err
	}

	return s.authJWT.RevokeRefreshToken(ctx, req.RefreshToken, values.UserID)
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
