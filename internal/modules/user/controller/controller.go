package controller

import (
	"github.com/PhantomX7/go-starter/internal/modules/user/service"

	"github.com/gin-gonic/gin"
)

type UserController interface {
	FindByToken(c *gin.Context)
	Create(c *gin.Context)
}

type userController struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) UserController {
	return &userController{
		userService: userService,
	}
}

func (u *userController) FindByToken(c *gin.Context) {

}

func (u *userController) Create(c *gin.Context) {

}
