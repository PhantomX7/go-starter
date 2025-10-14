package dto

import "github.com/PhantomX7/go-starter/internal/modules/user/dto"

type LoginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
}

type RegisterRequest struct {
	Username string `json:"username" form:"username" binding:"required,unique=users.username"`
	Email    string `json:"email" form:"email" binding:"required,email"`
	Phone    string `json:"phone" form:"phone" binding:"required"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
}

type RefreshRequest struct {
	RefreshToken string `form:"refresh_token" json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type MeResponse struct {
	dto.UserResponse
}
