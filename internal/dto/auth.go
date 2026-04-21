package dto

// LoginRequest is the payload for authenticating a user.
type LoginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
}

// RegisterRequest is the payload for registering a new user account.
type RegisterRequest struct {
	Name         string `json:"name" form:"name" binding:"required"`
	BusinessName string `json:"business_name" form:"business_name" binding:"required"`
	Email        string `json:"email" form:"email" binding:"required,unique=users.email"`
	Phone        string `json:"phone" form:"phone" binding:"required"`
	Password     string `json:"password" form:"password" binding:"required,min=8"`
}

// ChangePasswordRequest is the payload for rotating the authenticated user's password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" form:"old_password" binding:"required"`
	NewPassword string `json:"new_password" form:"new_password" binding:"required,min=8"`
	ExceptToken string `json:"except_token" form:"except_token" binding:"required"` // excepted refresh token
}

// RefreshRequest is the payload for rotating an access token via a refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
}

// AuthResponse is the token payload returned after successful authentication.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// MeResponse is the profile payload returned for the authenticated user.
type MeResponse struct {
	UserResponse
}
