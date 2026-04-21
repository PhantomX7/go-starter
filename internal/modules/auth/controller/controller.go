// Package controller exposes HTTP handlers for authentication flows.
package controller

import (
	"net/http"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/modules/auth/service"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthController defines the interface for auth controller operations
// Note: Login is handled directly by gin-jwt middleware
type AuthController interface {
	Register(ctx *gin.Context)
	GetMe(ctx *gin.Context)
	Refresh(ctx *gin.Context)
	ChangePassword(ctx *gin.Context)
	Logout(ctx *gin.Context)
}

type authController struct {
	authService service.AuthService
}

// NewAuthController constructs an AuthController.
func NewAuthController(authService service.AuthService) AuthController {
	return &authController{
		authService: authService,
	}
}

func (c *authController) Register(ctx *gin.Context) {
	var req dto.RegisterRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	res, err := c.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("register success", res))
}

func (c *authController) GetMe(ctx *gin.Context) {
	res, err := c.authService.GetMe(ctx.Request.Context())
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("get me success", res))
}

func (c *authController) Refresh(ctx *gin.Context) {
	var req dto.RefreshRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	res, err := c.authService.Refresh(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("refresh success", res))
}

func (c *authController) ChangePassword(ctx *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	err := c.authService.ChangePassword(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("password changed successfully", nil))
}

func (c *authController) Logout(ctx *gin.Context) {
	var req dto.LogoutRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	err := c.authService.Logout(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("logout successful", nil))
}
