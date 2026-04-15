package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// NewUserPagination creates a new pagination instance for users
func NewUserPagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("username", pagination.FilterConfig{
			Field: "username",
			Type:  pagination.FilterTypeString,
		}).
		AddFilter("email", pagination.FilterConfig{
			Field: "email",
			Type:  pagination.FilterTypeString,
		}).
		AddFilter("role", pagination.FilterConfig{
			Field: "role",
			Type:  pagination.FilterTypeEnum,
			EnumValues: []string{
				models.UserRoleAdmin.ToString(),
				models.UserRoleUser.ToString(),
				models.UserRoleReseller.ToString()},
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true}).
		AddSort("username", pagination.SortConfig{Field: "username", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     300,
		DefaultOrder: "id asc",
	})
}

// UserController defines the interface for user controller operations
type UserController interface {
	Index(ctx *gin.Context)
	Update(ctx *gin.Context)
	FindById(ctx *gin.Context)
	AssignAdminRole(ctx *gin.Context)
	ChangePassword(ctx *gin.Context)
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
		NewUserPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse(users, meta))
}

// @Summary		Update a user
// @Description	Update a user with the provided details
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			id		path		uint					true	"User ID"
// @Param			user	body		dto.UserUpdateRequest	true	"User Update Request"
// @Success		200		{object}	response.Response{data=dto.UserResponse}
// @Failure		400		{object}	response.Response
// @Failure		500		{object}	response.Response
// @Router			/user/{id} [put]
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

// @Summary		Find a user by ID
// @Description	Find a user with the provided ID
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			id	path		uint	true	"User ID"
// @Success		200	{object}	response.Response{data=dto.UserResponse}
// @Failure		400	{object}	response.Response
// @Failure		500	{object}	response.Response
// @Router			/user/{id} [get]
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

// AssignAdminRole handles assigning an admin role to a user
// @Summary      Assign admin role
// @Description  Assign an admin role to a user (changes user role to admin)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id    path      uint                          true  "User ID"
// @Param        body  body      dto.UserAssignAdminRoleRequest true  "Assign Admin Role Request"
// @Success      200  {object}  response.Response{data=dto.UserResponse}
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /admin/user/{id}/admin-role [post]
func (c *userController) AssignAdminRole(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.UserAssignAdminRoleRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	user, err := c.userService.AssignAdminRole(ctx.Request.Context(), uint(userID), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Admin role assigned successfully", user.ToResponse()))
}

// ChangePassword handles root changing an admin's password
func (c *userController) ChangePassword(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.ChangeAdminPasswordRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	err = c.userService.ChangePassword(ctx.Request.Context(), uint(userID), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Password changed successfully", nil))
}
