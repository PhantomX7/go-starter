package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/go-starter/internal/modules/user/dto"
	"github.com/PhantomX7/go-starter/internal/modules/user/service"
	"github.com/PhantomX7/go-starter/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserController defines the interface for user controller operations
type UserController interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindById(ctx *gin.Context)
}

// userController implements the UserController interface
type userController struct {
	userService service.UserService
}

// NewUserController creates a new instance of UserController
func NewUserController(userService service.UserService) UserController {
	return &userController{
		userService: userService,
	}
}

// Index handles the listing of users with pagination
func (c *userController) Index(ctx *gin.Context) {
	users, meta, err := c.userService.Index(
		ctx.Request.Context(),
		dto.NewUserPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse(users, meta))
}

// @Summary      Create a new user
// @Description  Create a new user with the provided details
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        user  body      dto.UserCreateRequest  true  "User Create Request"
// @Success      201  {object}  utils.Response{data=dto.UserResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /user [post]
func (c *userController) Create(ctx *gin.Context) {
	var req dto.UserCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	user, err := c.userService.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("User created successfully", user.ToResponse()))
}

// @Summary      Update a user
// @Description  Update a user with the provided details
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "User ID"
// @Param        user  body      dto.UserUpdateRequest  true  "User Update Request"
// @Success      200  {object}  utils.Response{data=dto.UserResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /user/{id} [put]
func (c *userController) Update(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.UserUpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		// Let the error middleware handle the validation error
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	user, err := c.userService.Update(ctx.Request.Context(), uint(userID), &req)
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("User updated successfully", user))
}

// @Summary      Delete a user
// @Description  Delete a user with the provided ID
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "User ID"
// @Success      200  {object}  utils.Response{data=nil}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /user/{id} [delete]
func (c *userController) Delete(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	err = c.userService.Delete(ctx.Request.Context(), uint(userID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("User deleted successfully", nil))
}

// @Summary      Find a user by ID
// @Description  Find a user with the provided ID
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "User ID"
// @Success      200  {object}  utils.Response{data=dto.UserResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /user/{id} [get]
func (c *userController) FindById(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	user, err := c.userService.FindById(ctx.Request.Context(), uint(userID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("User found successfully", user))
}
