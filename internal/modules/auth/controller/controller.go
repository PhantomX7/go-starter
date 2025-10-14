package controller

import (
	"net/http"

	"github.com/PhantomX7/go-starter/internal/middlewares"
	"github.com/PhantomX7/go-starter/internal/modules/auth/dto"
	"github.com/PhantomX7/go-starter/internal/modules/auth/service"
	"github.com/PhantomX7/go-starter/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthController defines the interface for auth controller operations
type AuthController interface {
	Login(ctx *gin.Context)
	Register(ctx *gin.Context)
	GetMe(ctx *gin.Context)
	Refresh(ctx *gin.Context)
}

// authController implements the AuthController interface
type authController struct {
	authService service.AuthService
}

// NewAuthController creates a new instance of AuthController
func NewAuthController(middleware *middlewares.Middleware, authService service.AuthService) AuthController {
	return &authController{
		authService: authService,
	}
}

// @Summary		Login
// @Description	Login with email and password
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			req	body		dto.LoginRequest							true	"Login request"
// @Success		200	{object}	response.Response{data=dto.AuthResponse}	"Login success"
// @Failure		400	{object}	response.Response							"Bad request"
// @Failure		401	{object}	response.Response							"Unauthorized"
// @Failure		500	{object}	response.Response							"Internal server error"
// @Router			/auth/login [post]
func (c *authController) Login(ctx *gin.Context) {
	var req dto.LoginRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	res, err := c.authService.Login(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("login success", res))
}

// @Summary		Register
// @Description	Register a new user
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			req	body		dto.RegisterRequest							true	"Register request"
// @Success		200	{object}	response.Response{data=dto.AuthResponse}	"Register success"
// @Failure		400	{object}	response.Response							"Bad request"
// @Failure		409	{object}	response.Response							"Conflict"
// @Failure		500	{object}	response.Response							"Internal server error"
// @Router			/auth/register [post]
func (c *authController) Register(ctx *gin.Context) {
	var req dto.RegisterRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	res, err := c.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("register success", res))
}

// @Summary		Get Me
// @Description	Get authenticated user information
// @Security		BearerAuth
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Success		200	{object}	response.Response{data=dto.UserResponse}	"Get me success"
// @Failure		401	{object}	response.Response							"Unauthorized"
// @Failure		500	{object}	response.Response							"Internal server error"
// @Router			/auth/me [get]
func (c *authController) GetMe(ctx *gin.Context) {
	res, err := c.authService.GetMe(ctx.Request.Context())
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("get me success", res))
}

// @Summary		Refresh
// @Description	Refresh access token using refresh token
// @Security		BearerAuth
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			req	body		dto.RefreshRequest							true	"Refresh request"
// @Success		200	{object}	response.Response{data=dto.AuthResponse}	"Refresh success"
// @Failure		400	{object}	response.Response							"Bad request"
// @Failure		401	{object}	response.Response							"Unauthorized"
// @Failure		500	{object}	response.Response							"Internal server error"
// @Router			/auth/refresh [post]
func (c *authController) Refresh(ctx *gin.Context) {
	var req dto.RefreshRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	res, err := c.authService.Refresh(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("refresh success", res))
}
