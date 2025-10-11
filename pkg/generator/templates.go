package generator

// moduleTemplate defines the main module file template
const moduleTemplate = `package {{.SnakeCase}}

import (
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/controller"
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/repository"
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/service"

	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		controller.New{{.PascalCase}}Controller,
		service.New{{.PascalCase}}Service,
		repository.New{{.PascalCase}}Repository,
	),
)
`

// controllerTemplate defines the controller file template
const controllerTemplate = `package controller

import (
	"net/http"
	"strconv"

	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/dto"
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/service"
	"github.com/LezendaCom/komputermedan/pkg/response"

	"github.com/gin-gonic/gin"
)

// {{.PascalCase}}Controller defines the interface for {{.LowerCase}} controller operations
type {{.PascalCase}}Controller interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindById(ctx *gin.Context)
}

// {{.CamelCase}}Controller implements the {{.PascalCase}}Controller interface
type {{.CamelCase}}Controller struct {
	{{.CamelCase}}Service service.{{.PascalCase}}Service
}

// New{{.PascalCase}}Controller creates a new instance of {{.PascalCase}}Controller
func New{{.PascalCase}}Controller({{.CamelCase}}Service service.{{.PascalCase}}Service) {{.PascalCase}}Controller {
	return &{{.CamelCase}}Controller{
		{{.CamelCase}}Service: {{.CamelCase}}Service,
	}
}

// Index handles the listing of {{.LowerCase}}s with pagination
func (c *{{.CamelCase}}Controller) Index(ctx *gin.Context) {
	{{.LowerCase}}s, meta, err := c.{{.CamelCase}}Service.Index(
		ctx.Request.Context(),
		dto.New{{.PascalCase}}Pagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse({{.LowerCase}}s, meta))
}

// @Summary      Create a new {{.LowerCase}}
// @Description  Create a new {{.LowerCase}} with the provided details
// @Tags         {{.LowerCase}}
// @Accept       json
// @Produce      json
// @Param        {{.LowerCase}}  body      dto.{{.PascalCase}}CreateRequest  true  "{{.PascalCase}} Create Request"
// @Success      201  {object}  utils.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /{{.LowerCase}} [post]
func (c *{{.CamelCase}}Controller) Create(ctx *gin.Context) {
	var req dto.{{.PascalCase}}CreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("{{.PascalCase}} created successfully", {{.LowerCase}}.ToResponse()))
}

// @Summary      Update a {{.LowerCase}}
// @Description  Update a {{.LowerCase}} with the provided details
// @Tags         {{.LowerCase}}
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "{{.PascalCase}} ID"
// @Param        {{.LowerCase}}  body      dto.{{.PascalCase}}UpdateRequest  true  "{{.PascalCase}} Update Request"
// @Success      200  {object}  utils.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /{{.LowerCase}}/{id} [put]
func (c *{{.CamelCase}}Controller) Update(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.{{.PascalCase}}UpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		// Let the error middleware handle the validation error
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Update(ctx.Request.Context(), uint({{.LowerCase}}ID), &req)
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} updated successfully", {{.LowerCase}}))
}

// @Summary      Delete a {{.LowerCase}}
// @Description  Delete a {{.LowerCase}} with the provided ID
// @Tags         {{.LowerCase}}
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "{{.PascalCase}} ID"
// @Success      200  {object}  utils.Response{data=nil}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /{{.LowerCase}}/{id} [delete]
func (c *{{.CamelCase}}Controller) Delete(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	err = c.{{.CamelCase}}Service.Delete(ctx.Request.Context(), uint({{.LowerCase}}ID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} deleted successfully", nil))
}

// @Summary      Find a {{.LowerCase}} by ID
// @Description  Find a {{.LowerCase}} with the provided ID
// @Tags         {{.LowerCase}}
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "{{.PascalCase}} ID"
// @Success      200  {object}  utils.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /{{.LowerCase}}/{id} [get]
func (c *{{.CamelCase}}Controller) FindById(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	{{.LowerCase}}, err := c.{{.CamelCase}}Service.FindById(ctx.Request.Context(), uint({{.LowerCase}}ID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} found successfully", {{.LowerCase}}))
}
`

// serviceTemplate defines the service file template
const serviceTemplate = `package service

import (
	"context"

	"github.com/LezendaCom/komputermedan/internal/models"
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/dto"
	"github.com/LezendaCom/komputermedan/internal/modules/{{.SnakeCase}}/repository"
	"github.com/LezendaCom/komputermedan/pkg/pagination"
	"github.com/LezendaCom/komputermedan/pkg/response"

	"github.com/jinzhu/copier"
)

// {{.PascalCase}}Service defines the interface for {{.LowerCase}} service operations
type {{.PascalCase}}Service interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]models.{{.PascalCase}}, response.Meta, error)
	Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (models.{{.PascalCase}}, error)
	Update(ctx context.Context, {{.LowerCase}}Id uint, req *dto.{{.PascalCase}}UpdateRequest) (models.{{.PascalCase}}, error)
	Delete(ctx context.Context, {{.LowerCase}}Id uint) error
	FindById(ctx context.Context, {{.LowerCase}}Id uint) (models.{{.PascalCase}}, error)
}

// {{.CamelCase}}Service implements the {{.PascalCase}}Service interface
type {{.CamelCase}}Service struct {
	{{.CamelCase}}Repository repository.{{.PascalCase}}Repository
}

// New{{.PascalCase}}Service creates a new instance of {{.PascalCase}}Service
func New{{.PascalCase}}Service(repository repository.{{.PascalCase}}Repository) {{.PascalCase}}Service {
	return &{{.CamelCase}}Service{
		{{.CamelCase}}Repository: repository,
	}
}

// Index implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Index(ctx context.Context, pg *pagination.Pagination) ([]models.{{.PascalCase}}, response.Meta, error) {
	{{.LowerCase}}s, err := s.{{.CamelCase}}Repository.FindAll(ctx, pg)
	if err != nil {
		return {{.LowerCase}}s, response.Meta{}, err
	}

	count, err := s.{{.CamelCase}}Repository.Count(ctx, pg)
	if err != nil {
		return {{.LowerCase}}s, response.Meta{}, err
	}

	return {{.LowerCase}}s, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (models.{{.PascalCase}}, error) {
	var {{.LowerCase}} models.{{.PascalCase}}

	err := copier.Copy(&{{.LowerCase}}, &req)
	if err != nil {
		return {{.LowerCase}}, err
	}

	err = s.{{.CamelCase}}Repository.Create(ctx, &{{.LowerCase}})
	if err != nil {
		return {{.LowerCase}}, err
	}

	return {{.LowerCase}}, nil
}

// Update implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Update(ctx context.Context, {{.LowerCase}}Id uint, req *dto.{{.PascalCase}}UpdateRequest) (models.{{.PascalCase}}, error) {
	{{.LowerCase}}, err := s.{{.CamelCase}}Repository.FindById(ctx, {{.LowerCase}}Id)
	if err != nil {
		return {{.LowerCase}}, err
	}

	err = copier.Copy(&{{.LowerCase}}, &req)
	if err != nil {
		return {{.LowerCase}}, err
	}

	err = s.{{.CamelCase}}Repository.Update(ctx, &{{.LowerCase}})
	if err != nil {
		return {{.LowerCase}}, err
	}

	return {{.LowerCase}}, nil
}

// Delete implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Delete(ctx context.Context, {{.LowerCase}}Id uint) error {
	var {{.LowerCase}} models.{{.PascalCase}}

	{{.LowerCase}}.ID = {{.LowerCase}}Id

	return s.{{.CamelCase}}Repository.Delete(ctx, &{{.LowerCase}})
}

// FindById implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) FindById(ctx context.Context, {{.LowerCase}}Id uint) (models.{{.PascalCase}}, error) {
	{{.LowerCase}}, err := s.{{.CamelCase}}Repository.FindById(ctx, {{.LowerCase}}Id)
	if err != nil {
		return {{.LowerCase}}, err
	}
	return {{.LowerCase}}, nil
}
`

// repositoryTemplate defines the repository file template
const repositoryTemplate = `package repository

import (
	"github.com/LezendaCom/komputermedan/internal/models"
	"github.com/LezendaCom/komputermedan/pkg/repository"

	"gorm.io/gorm"
)

// {{.PascalCase}}Repository defines the interface for {{.LowerCase}} repository operations
type {{.PascalCase}}Repository interface {
	repository.IRepository[models.{{.PascalCase}}]
}

// {{.CamelCase}}Repository implements the {{.PascalCase}}Repository interface
type {{.CamelCase}}Repository struct {
	repository.Repository[models.{{.PascalCase}}]
}

// New{{.PascalCase}}Repository creates a new instance of {{.PascalCase}}Repository
func New{{.PascalCase}}Repository(db *gorm.DB) {{.PascalCase}}Repository {
	return &{{.CamelCase}}Repository{
		Repository: repository.Repository[models.{{.PascalCase}}]{
			DB: db,
		},
	}
}
`

// dtoTemplate defines the DTO file template
const dtoTemplate = `package dto

import (
	"time"

	"github.com/LezendaCom/komputermedan/pkg/pagination"
)

// {{.PascalCase}}CreateRequest defines the structure for creating a new {{.LowerCase}}
type {{.PascalCase}}CreateRequest struct {
	Name        string ` + "`json:\"name\" form:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\" form:\"description\"`" + `
}

// {{.PascalCase}}UpdateRequest defines the structure for updating a {{.LowerCase}}
type {{.PascalCase}}UpdateRequest struct {
	Name        string ` + "`json:\"name\" form:\"name\"`" + `
	Description string ` + "`json:\"description\" form:\"description\"`" + `
}

// {{.PascalCase}}Response defines the structure for {{.LowerCase}} response
type {{.PascalCase}}Response struct {
	ID          uint      ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt   time.Time ` + "`json:\"updated_at\"`" + `
}

// New{{.PascalCase}}Pagination creates a new pagination instance for {{.LowerCase}}s
func New{{.PascalCase}}Pagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field:     "name",
			TableName: "{{.LowerCase}}s",
			Type:      pagination.FilterTypeString,
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
		MaxLimit:     300,
		DefaultOrder: "id asc",
	})
}
`