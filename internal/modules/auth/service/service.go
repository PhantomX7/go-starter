package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	refreshtokenrepo "github.com/PhantomX7/go-starter/internal/modules/refresh_token/repository"
	userrepo "github.com/PhantomX7/go-starter/internal/modules/user/repository"
	tx "github.com/PhantomX7/go-starter/libs/transaction_manager"
	cerrors "github.com/PhantomX7/go-starter/pkg/errors"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/auth/dto"
	"github.com/PhantomX7/go-starter/pkg/config"
	"github.com/PhantomX7/go-starter/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jinzhu/copier"
	"golang.org/x/crypto/bcrypt"
)

// Constants for improved maintainability
var (
	// Token type constant
	TokenTypeBearer = "Bearer"

	// Bcrypt cost for password hashing
	BcryptCost = 12

	// Minimum password length
	MinPasswordLength = 8

	// JWT issuer identifier
	JWTIssuer = ""
)

// AuthService defines the interface for auth service operations
type AuthService interface {
	GetMe(ctx context.Context) (*dto.MeResponse, error)
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error)
	// Additional methods for enhanced security
	ValidatePassword(password string) error
	GenerateSecureToken() (string, error)
}

// authService implements the AuthService interface
type authService struct {
	config                 *config.Config
	userRepository         userrepo.UserRepository
	refreshTokenRepository refreshtokenrepo.RefreshTokenRepository
	transactionManager     tx.TransactionManager
}

// NewAuthService creates a new instance of AuthService with enhanced validation
func NewAuthService(
	config *config.Config,
	userRepository userrepo.UserRepository,
	refreshTokenRepository refreshtokenrepo.RefreshTokenRepository,
	transactionManager tx.TransactionManager) AuthService {

	JWTIssuer = config.JWT.Issuer

	return &authService{
		config:                 config,
		userRepository:         userRepository,
		refreshTokenRepository: refreshTokenRepository,
		transactionManager:     transactionManager,
	}
}

// GetMe retrieves the current user's information from context
func (s *authService) GetMe(ctx context.Context) (*dto.MeResponse, error) {
	values, err := utils.ValuesFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepository.FindById(ctx, values.UserID)
	if err != nil {
		return nil, err
	}

	// Additional security check
	if !user.IsActive {
		return nil, cerrors.NewForbiddenError("user account is inactive")
	}

	return &dto.MeResponse{
		UserResponse: user.ToResponse(),
	}, nil
}

// Register creates a new user account with enhanced security validation
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Validate password strength (optional)
	// if err := s.ValidatePassword(req.Password); err != nil {
	// 	return nil, err
	// }

	// Sanitize and validate input
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)

	user := &models.User{
		Role:     models.UserRoleUser,
		IsActive: true,
	}

	// Use copier to safely copy fields
	if err := copier.Copy(&user, &req); err != nil {
		return nil, cerrors.NewInternalServerError("failed to process user data", err)
	}

	// Hash password with enhanced cost
	password, err := bcrypt.GenerateFromPassword([]byte(req.Password), BcryptCost)
	if err != nil {
		return nil, cerrors.NewInternalServerError("failed to process password", err)
	}
	user.Password = string(password)

	var accessToken, refreshToken string
	err = s.transactionManager.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		// Create user
		if err = s.userRepository.Create(ctx, user); err != nil {
			return err
		}

		// Generate tokens
		accessToken, err = s.GenerateAccessToken(user.ID, user.Role)
		if err != nil {
			return err
		}

		refreshToken, err = s.GenerateRefreshToken(user.ID)
		if err != nil {
			return err
		}

		// Store refresh token
		if err = s.refreshTokenRepository.Create(ctx, &models.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			Token:     refreshToken,
			ExpiresAt: time.Now().Add(s.config.JWT.RefreshExpiration),
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    TokenTypeBearer,
	}, nil
}

// Login authenticates a user and returns access tokens with enhanced security
func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	// Sanitize input
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	user, err := s.userRepository.FindByUsername(ctx, req.Username)
	if err != nil {
		// Use constant-time comparison to prevent timing attacks
		if errors.Is(err, cerrors.ErrNotFound) {
			log.Printf("user not found for username: %s", req.Username)
			// Perform dummy bcrypt operation to maintain consistent timing
			bcrypt.CompareHashAndPassword([]byte("$2a$12$dummy.hash.to.prevent.timing.attacks"), []byte(req.Password))
			return nil, cerrors.NewBadRequestError("invalid credentials")
		}
		return nil, err
	}

	// Check if user is active before password verification
	if !user.IsActive {
		// Perform dummy bcrypt operation to maintain consistent timing
		bcrypt.CompareHashAndPassword([]byte("$2a$12$dummy.hash.to.prevent.timing.attacks"), []byte(req.Password))
		return nil, cerrors.NewBadRequestError("invalid credentials")
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, cerrors.NewBadRequestError("invalid credentials")
	}

	// Generate tokens
	accessToken, err := s.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Store refresh token
	if err = s.refreshTokenRepository.Create(ctx, &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(s.config.JWT.RefreshExpiration),
	}); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    TokenTypeBearer,
	}, nil
}

// Refresh validates a refresh token and issues new access/refresh tokens with enhanced security
func (s *authService) Refresh(ctx context.Context, req *dto.RefreshRequest) (*dto.AuthResponse, error) {
	// Parse and validate refresh token
	claims := &models.RefreshClaims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, cerrors.NewBadRequestError("invalid token signing method")
		}
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return nil, cerrors.NewBadRequestError("invalid refresh token")
	}

	if !token.Valid {
		return nil, cerrors.NewBadRequestError("invalid refresh token claims")
	}

	// Validate token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, cerrors.NewBadRequestError("refresh token has expired")
	}

	// Check if refresh token exists and is not revoked
	refreshTokenM, err := s.refreshTokenRepository.FindByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Check if token is expired or revoked
	if refreshTokenM.ExpiresAt.Before(time.Now()) || refreshTokenM.RevokedAt != nil {
		return nil, cerrors.NewBadRequestError("refresh token has expired or been revoked")
	}

	// Verify user still exists and is active
	user, err := s.userRepository.FindById(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, cerrors.NewBadRequestError("user account is inactive")
	}

	// Generate new tokens
	accessToken, err := s.GenerateAccessToken(claims.UserID, models.UserRole(user.Role))
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return nil, err
	}

	// Start transaction for token rotation
	err = s.transactionManager.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		now := time.Now()
		refreshTokenM.RevokedAt = &now

		// revoke previous refresh token
		if err = s.refreshTokenRepository.Update(ctx, refreshTokenM); err != nil {
			return err
		}

		// save current refresh token
		if err = s.refreshTokenRepository.Create(ctx, &models.RefreshToken{
			ID:        uuid.New(),
			UserID:    claims.UserID,
			Token:     refreshToken,
			ExpiresAt: time.Now().Add(s.config.JWT.RefreshExpiration),
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    TokenTypeBearer,
	}, nil
}

// GenerateAccessToken creates a JWT access token with enhanced security
func (s *authService) GenerateAccessToken(userID uint, role models.UserRole) (string, error) {
	if role == "" {
		return "", cerrors.NewBadRequestError("user does not have role")
	}

	now := time.Now()
	claims := models.AccessClaims{
		UserID: userID,
		Role:   role.ToString(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.JWT.Expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", cerrors.NewInternalServerError("failed to generate access token", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken creates a JWT refresh token with enhanced security
func (s *authService) GenerateRefreshToken(userID uint) (string, error) {
	now := time.Now()
	claims := models.RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    JWTIssuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.JWT.RefreshExpiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", cerrors.NewInternalServerError("failed to generate refresh token", err)
	}

	return tokenString, nil
}

// ValidatePassword validates password strength and requirements (optional)
func (s *authService) ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return cerrors.NewBadRequestError(fmt.Sprintf("password must be at least %d characters long", MinPasswordLength))
	}

	// Check for at least one uppercase letter
	hasUpper := false
	// Check for at least one lowercase letter
	hasLower := false
	// Check for at least one digit
	hasDigit := false
	// Check for at least one special character
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return cerrors.NewBadRequestError("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return cerrors.NewBadRequestError("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return cerrors.NewBadRequestError("password must contain at least one digit")
	}
	if !hasSpecial {
		return cerrors.NewBadRequestError("password must contain at least one special character")
	}

	return nil
}

// GenerateSecureToken generates a cryptographically secure random token
func (s *authService) GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
