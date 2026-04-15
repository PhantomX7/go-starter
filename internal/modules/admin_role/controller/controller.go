// internal/modules/admin_role/controller/controller.go
package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdminRoleController interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindById(ctx *gin.Context)
	GetAllPermissions(ctx *gin.Context)
}

type adminRoleController struct {
	adminRoleService service.AdminRoleService
}

func NewAdminRoleController(adminRoleService service.AdminRoleService) AdminRoleController {
	return &adminRoleController{
		adminRoleService: adminRoleService,
	}
}

// newAdminRolePagination creates a new pagination instance for admin roles
func newAdminRolePagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field: "name",
			Type:  pagination.FilterTypeString,
		}).
		AddFilter("is_active", pagination.FilterConfig{
			Field: "is_active",
			Type:  pagination.FilterTypeBool,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultOrder: "id desc",
	})
}

// Index handles the listing of admin roles with pagination
// @Summary      List admin roles
// @Description  Get a paginated list of admin roles
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Param        page    query     int     false  "Page number"
// @Param        limit   query     int     false  "Items per page"
// @Param        name    query     string  false  "Filter by name"
// @Param        is_active query   bool    false  "Filter by active status"
// @Success      200  {object}  response.Response{data=[]dto.AdminRoleListResponse}
// @Failure      500  {object}  response.Response
// @Router       /admin/admin-role [get]
func (c *adminRoleController) Index(ctx *gin.Context) {
	roles, meta, err := c.adminRoleService.Index(
		ctx.Request.Context(),
		newAdminRolePagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildPaginationResponse(roles, meta))
}

// Create handles the creation of a new admin role
// @Summary      Create admin role
// @Description  Create a new admin role with permissions
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateAdminRoleRequest  true  "Admin Role Create Request"
// @Success      201  {object}  response.Response{data=dto.AdminRoleResponse}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /admin/admin-role [post]
func (c *adminRoleController) Create(ctx *gin.Context) {
	var req dto.CreateAdminRoleRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	adminRole, err := c.adminRoleService.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("Admin role created successfully", adminRole.ToResponse()))
}

// Update handles the update of an existing admin role
// @Summary      Update admin role
// @Description  Update an admin role's details and permissions
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Param        id    path      uint                       true  "Admin Role ID"
// @Param        body  body      dto.UpdateAdminRoleRequest true  "Admin Role Update Request"
// @Success      200  {object}  response.Response{data=dto.AdminRoleResponse}
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /admin/admin-role/{id} [put]
func (c *adminRoleController) Update(ctx *gin.Context) {
	roleID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.UpdateAdminRoleRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	adminRole, err := c.adminRoleService.Update(ctx.Request.Context(), uint(roleID), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Admin role updated successfully", adminRole.ToResponse()))
}

// Delete handles the deletion of an admin role
// @Summary      Delete admin role
// @Description  Delete an admin role (cannot delete if users are assigned)
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Param        id   path      uint  true  "Admin Role ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /admin/admin-role/{id} [delete]
func (c *adminRoleController) Delete(ctx *gin.Context) {
	roleID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	if err := c.adminRoleService.Delete(ctx.Request.Context(), uint(roleID)); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Admin role deleted successfully", nil))
}

// FindById handles fetching a single admin role by ID
// @Summary      Get admin role by ID
// @Description  Get an admin role's details including permissions
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Param        id   path      uint  true  "Admin Role ID"
// @Success      200  {object}  response.Response{data=dto.AdminRoleResponse}
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /admin/admin-role/{id} [get]
func (c *adminRoleController) FindById(ctx *gin.Context) {
	roleID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	adminRole, err := c.adminRoleService.FindById(ctx.Request.Context(), uint(roleID))
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Admin role found successfully", adminRole))
}

// GetAllPermissions returns all available permissions
// @Summary      Get all permissions
// @Description  Get all available permissions grouped by resource
// @Tags         admin-role
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Response{data=map[string][]string}
// @Router       /admin/admin-role/permissions [get]
func (c *adminRoleController) GetAllPermissions(ctx *gin.Context) {
	permissions := c.adminRoleService.GetAllPermissions(ctx.Request.Context())
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Permissions retrieved successfully", permissions))
}
