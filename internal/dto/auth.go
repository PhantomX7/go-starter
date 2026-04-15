package dto

type LoginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Name         string `json:"name" form:"name" binding:"required"`
	BusinessName string `json:"business_name" form:"business_name" binding:"required"`
	Email        string `json:"email" form:"email" binding:"required,unique=users.email"`
	Phone        string `json:"phone" form:"phone" binding:"required"`
	Password     string `json:"password" form:"password" binding:"required,min=8"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" form:"old_password" binding:"required"`
	NewPassword string `json:"new_password" form:"new_password" binding:"required,min=8"`
	ExceptToken string `json:"except_token" form:"except_token" binding:"required"` // excepted refresh token
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" form:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type MeResponse struct {
	UserResponse
}
