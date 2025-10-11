package controller

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/modules/user/service"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

// AuthController defines the interface for auth controller operations
type AuthController interface {
	LoginOauth(ctx *gin.Context)
	CallbackOauth(ctx *gin.Context)
}

// authController implements the AuthController interface
type authController struct {
	userService service.UserService
}

// NewAuthController creates a new instance of AuthController
func NewAuthController() AuthController {
	return &authController{
	}
}

func (s *authController) LoginOauth(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), "provider", c.Param("provider"))
	req := c.Request.WithContext(ctx)

	// Begin the authentication process
	gothic.BeginAuthHandler(c.Writer, req)
}

func (s *authController) CallbackOauth(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), "provider", c.Param("provider"))
	req := c.Request.WithContext(ctx)

	// Complete the authentication process
	user, err := gothic.CompleteUserAuth(c.Writer, req)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	// Handle the user data
	c.JSON(200, gin.H{"user": user})
}
