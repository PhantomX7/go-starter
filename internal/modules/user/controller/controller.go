// Package controller exposes HTTP handlers for user management.
package controller

import (
	"net/http"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/pkg/ginx"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// userFilterDefinition is the static filter/sort schema for the user list
// endpoint. It is built once at package init rather than per request: only the
// query-string conditions change between requests, while the definition itself
// is immutable after construction. NewPagination treats it as read-only (it
// only reads the filters/sorts maps), so a single shared instance is safe to
// use concurrently across request goroutines.
var userFilterDefinition = pagination.NewFilterDefinition().
	AddFilter("username", pagination.FilterConfig{
		Column: generated.User.Username,
		Type:   pagination.FilterTypeString,
	}).
	AddFilter("email", pagination.FilterConfig{
		Column: generated.User.Email,
		Type:   pagination.FilterTypeString,
	}).
	AddFilter("role", pagination.FilterConfig{
		Field: "role", // enum column is models.UserRole, not a scalar field helper — stay on the string path
		Type:  pagination.FilterTypeEnum,
		EnumValues: []string{
			models.UserRoleAdmin.ToString(),
			models.UserRoleUser.ToString(),
		},
	}).
	AddFilter("created_at", pagination.FilterConfig{
		Column: generated.Timestamp.CreatedAt,
		Type:   pagination.FilterTypeDate,
	}).
	AddSort("id", pagination.SortConfig{Column: generated.User.ID, Allowed: true}).
	AddSort("username", pagination.SortConfig{Column: generated.User.Username, Allowed: true}).
	AddSort("created_at", pagination.SortConfig{Column: generated.Timestamp.CreatedAt, Allowed: true})

// NewUserPagination binds the per-request query conditions to the shared,
// pre-built user filter definition.
func NewUserPagination(conditions map[string][]string) *pagination.Pagination {
	return pagination.NewPagination(conditions, userFilterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     300,
		DefaultOrder: "id asc",
	})
}

// UserController defines the interface for user controller operations
type UserController interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	FindByID(ctx *gin.Context)
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

// @Summary		List users
// @Description	Get a paginated list of users
// @Tags			user
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			limit		query		int		false	"Limit"
// @Param			offset		query		int		false	"Offset"
// @Param			sort		query		string	false	"Sort"
// @Param			username	query		string	false	"Filter by username"
// @Param			email		query		string	false	"Filter by email"
// @Param			role		query		string	false	"Filter by role"
// @Success		200			{object}	response.Response{data=[]dto.UserResponse,meta=response.Meta}
// @Failure		400			{object}	response.Response
// @Failure		500			{object}	response.Response
// @Router			/admin/user [get]
func (c *userController) Index(ctx *gin.Context) {
	users, meta, err := c.userService.Index(
		ctx.Request.Context(),
		NewUserPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse(users, meta))
}

// @Summary		Create an admin user
// @Description	Create a new admin account with an assigned admin role; the account must rotate its password on first login
// @Tags			user
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			user	body		dto.AdminUserCreateRequest	true	"Admin User Create Request"
// @Success		201		{object}	response.Response{data=dto.UserResponse}
// @Failure		400		{object}	response.Response
// @Failure		500		{object}	response.Response
// @Router			/admin/user [post]
func (c *userController) Create(ctx *gin.Context) {
	var req dto.AdminUserCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	user, err := c.userService.Create(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("Admin user created successfully", user.ToResponse()))
}

// @Summary		Update a user
// @Description	Update a user with the provided details
// @Tags			user
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		uint					true	"User ID"
// @Param			user	body		dto.UserUpdateRequest	true	"User Update Request"
// @Success		200		{object}	response.Response{data=dto.UserResponse}
// @Failure		400		{object}	response.Response
// @Failure		500		{object}	response.Response
// @Router			/admin/user/{id} [patch]
func (c *userController) Update(ctx *gin.Context) {
	userID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	var req dto.UserUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		// Let the error middleware handle the validation error
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	user, err := c.userService.Update(ctx.Request.Context(), userID, &req)
	if err != nil {
		// Let the error middleware handle the internal error
		_ = ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("User updated successfully", user))
}

// @Summary		Find a user by ID
// @Description	Find a user with the provided ID
// @Tags			user
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		uint	true	"User ID"
// @Success		200	{object}	response.Response{data=dto.UserResponse}
// @Failure		400	{object}	response.Response
// @Failure		500	{object}	response.Response
// @Router			/admin/user/{id} [get]
func (c *userController) FindByID(ctx *gin.Context) {
	userID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}
	user, err := c.userService.FindByID(ctx.Request.Context(), userID)
	if err != nil {
		// Let the error middleware handle the internal error
		_ = ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("User found successfully", user))
}

// AssignAdminRole handles assigning an admin role to a user
//
//	@Summary		Assign admin role
//	@Description	Assign an admin role to a user (changes user role to admin)
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint							true	"User ID"
//	@Param			body	body		dto.UserAssignAdminRoleRequest	true	"Assign Admin Role Request"
//	@Success		200		{object}	response.Response{data=dto.UserResponse}
//	@Failure		400		{object}	response.Response
//	@Failure		404		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/admin/user/{id}/admin-role [post]
func (c *userController) AssignAdminRole(ctx *gin.Context) {
	userID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	var req dto.UserAssignAdminRoleRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	user, err := c.userService.AssignAdminRole(ctx.Request.Context(), userID, &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Admin role assigned successfully", user.ToResponse()))
}

// ChangePassword handles root changing an admin's password
//
//	@Summary		Change an admin's password
//	@Description	Root sets a new password for an admin account
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		uint							true	"User ID"
//	@Param			body	body		dto.ChangeAdminPasswordRequest	true	"Change Admin Password Request"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		404		{object}	response.Response
//	@Failure		500		{object}	response.Response
//	@Router			/admin/user/{id}/change-password [post]
func (c *userController) ChangePassword(ctx *gin.Context) {
	userID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	var req dto.ChangeAdminPasswordRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	if err := c.userService.ChangePassword(ctx.Request.Context(), userID, &req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Password changed successfully", nil))
}
