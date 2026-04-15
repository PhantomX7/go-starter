package generator

// moduleTemplate defines the main module file template
const moduleTemplate = `package {{.SnakeCase}}

import (
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/controller"
	"github.com/PhantomX7/athletonnal/modules/{{.SnakeCase}}/repository"
	"github.com/PhantomX7/athletonnal/modules/{{.SnakeCase}}/service"

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

	"github.com/PhantomX7/athletonnal/dto"
	"github.com/PhantomX7/athletonnal/modules/{{.SnakeCase}}/service"
	"github.com/PhantomX7/athletonagination"
	"github.com/PhantomX7/athletonesponse"

	"github.com/gin-gonic/gin"
)

type {{.PascalCase}}Controller interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindById(ctx *gin.Context)
}

type {{.CamelCase}}Controller struct {
	{{.CamelCase}}Service service.{{.PascalCase}}Service
}

func New{{.PascalCase}}Controller({{.CamelCase}}Service service.{{.PascalCase}}Service) {{.PascalCase}}Controller {
	return &{{.CamelCase}}Controller{
		{{.CamelCase}}Service: {{.CamelCase}}Service,
	}
}

// new{{.PascalCase}}Pagination creates a new pagination instance for {{.LowerCase}}s
func new{{.PascalCase}}Pagination(conditions map[string][]string) *pagination.Pagination {
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
		MaxLimit:     100,
		DefaultOrder: "id desc",
	})
}

// Index handles the listing of {{.LowerCase}}s with pagination
func (c *{{.CamelCase}}Controller) Index(ctx *gin.Context) {
	{{.LowerCase}}s, meta, err := c.{{.CamelCase}}Service.Index(
		ctx.Request.Context(),
		new{{.PascalCase}}Pagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
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
// @Success      201  {object}  response.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /{{.LowerCase}} [post]
func (c *{{.CamelCase}}Controller) Create(ctx *gin.Context) {
	var req dto.{{.PascalCase}}CreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
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
// @Success      200  {object}  response.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /{{.LowerCase}}/{id} [put]
func (c *{{.CamelCase}}Controller) Update(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.{{.PascalCase}}UpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Update(ctx.Request.Context(), uint({{.LowerCase}}ID), &req)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} updated successfully", {{.LowerCase}}.ToResponse()))
}

// @Summary      Delete a {{.LowerCase}}
// @Description  Delete a {{.LowerCase}} with the provided ID
// @Tags         {{.LowerCase}}
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "{{.PascalCase}} ID"
// @Success      200  {object}  response.Response{data=nil}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /{{.LowerCase}}/{id} [delete]
func (c *{{.CamelCase}}Controller) Delete(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	err = c.{{.CamelCase}}Service.Delete(ctx.Request.Context(), uint({{.LowerCase}}ID))
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
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
// @Success      200  {object}  response.Response{data=dto.{{.PascalCase}}Response}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /{{.LowerCase}}/{id} [get]
func (c *{{.CamelCase}}Controller) FindById(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.FindById(ctx.Request.Context(), uint({{.LowerCase}}ID))
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} found successfully", {{.LowerCase}}.ToResponse()))
}
`

// serviceTemplate defines the service file template
const serviceTemplate = `package service

import (
	"context"

	"github.com/PhantomX7/athletonnal/dto"
	"github.com/PhantomX7/athletonnal/models"
	"github.com/PhantomX7/athletonnal/modules/{{.SnakeCase}}/repository"
	cerrors "github.com/PhantomX7/athletonrrors"
	"github.com/PhantomX7/athletonogger"
	"github.com/PhantomX7/athletonagination"
	"github.com/PhantomX7/athletonesponse"
	"github.com/PhantomX7/athletontils"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

type {{.PascalCase}}Service interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error)
	Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error)
	Update(ctx context.Context, {{.LowerCase}}Id uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error)
	Delete(ctx context.Context, {{.LowerCase}}Id uint) error
	FindById(ctx context.Context, {{.LowerCase}}Id uint) (*models.{{.PascalCase}}, error)
}

type {{.CamelCase}}Service struct {
	{{.CamelCase}}Repository repository.{{.PascalCase}}Repository
}

func New{{.PascalCase}}Service({{.CamelCase}}Repository repository.{{.PascalCase}}Repository) {{.PascalCase}}Service {
	return &{{.CamelCase}}Service{
		{{.CamelCase}}Repository: {{.CamelCase}}Repository,
	}
}

// Index implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Fetching {{.LowerCase}}s with pagination",
		zap.String("request_id", requestID),
		zap.Int("page", pg.GetPage()),
		zap.Int("limit", pg.Limit),
		zap.Int("offset", pg.Offset),
	)

	{{.LowerCase}}s, err := s.{{.CamelCase}}Repository.FindAll(ctx, pg)
	if err != nil {
		logger.Error("Failed to fetch {{.LowerCase}}s",
			zap.String("request_id", requestID),
			zap.Int("page", pg.GetPage()),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	count, err := s.{{.CamelCase}}Repository.Count(ctx, pg)
	if err != nil {
		logger.Error("Failed to count {{.LowerCase}}s",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, response.Meta{}, err
	}

	logger.Info("{{.PascalCase}}s fetched successfully",
		zap.String("request_id", requestID),
		zap.Int("returned_count", len({{.LowerCase}}s)),
		zap.Int64("total_count", count),
	)

	return {{.LowerCase}}s, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Creating {{.LowerCase}}",
		zap.String("request_id", requestID),
	)

	{{.LowerCase}} := &models.{{.PascalCase}}{}

	err := copier.Copy({{.LowerCase}}, req)
	if err != nil {
		logger.Error("Failed to copy {{.LowerCase}} data",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return {{.LowerCase}}, err
	}

	err = s.{{.CamelCase}}Repository.Create(ctx, {{.LowerCase}})
	if err != nil {
		logger.Error("Failed to create {{.LowerCase}}",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Info("{{.PascalCase}} created successfully",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}.ID),
	)

	return {{.LowerCase}}, nil
}

// Update implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Update(ctx context.Context, {{.LowerCase}}Id uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Updating {{.LowerCase}}",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	{{.LowerCase}}, err := s.{{.CamelCase}}Repository.FindById(ctx, {{.LowerCase}}Id)
	if err != nil {
		logger.Error("Failed to find {{.LowerCase}} for update",
			zap.String("request_id", requestID),
			zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
			zap.Error(err),
		)
		return {{.LowerCase}}, err
	}

	err = copier.Copy(&{{.LowerCase}}, &req)
	if err != nil {
		logger.Error("Failed to copy {{.LowerCase}} data",
			zap.String("request_id", requestID),
			zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
			zap.Error(err),
		)
		return {{.LowerCase}}, err
	}

	err = s.{{.CamelCase}}Repository.Update(ctx, {{.LowerCase}})
	if err != nil {
		logger.Error("Failed to update {{.LowerCase}}",
			zap.String("request_id", requestID),
			zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
			zap.Error(err),
		)
		return {{.LowerCase}}, err
	}

	logger.Info("{{.PascalCase}} updated successfully",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	return {{.LowerCase}}, nil
}

// Delete implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) Delete(ctx context.Context, {{.LowerCase}}Id uint) error {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Info("Deleting {{.LowerCase}}",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	{{.LowerCase}}, err := s.{{.CamelCase}}Repository.FindById(ctx, {{.LowerCase}}Id)
	if err != nil {
		logger.Error("Failed to find {{.LowerCase}} for deletion",
			zap.String("request_id", requestID),
			zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
			zap.Error(err),
		)
		return err
	}

	err = s.{{.CamelCase}}Repository.Delete(ctx, {{.LowerCase}})
	if err != nil {
		logger.Error("Failed to delete {{.LowerCase}}",
			zap.String("request_id", requestID),
			zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
			zap.Error(err),
		)
		return err
	}

	logger.Info("{{.PascalCase}} deleted successfully",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	return nil
}

// FindById implements {{.PascalCase}}Service.
func (s *{{.CamelCase}}Service) FindById(ctx context.Context, {{.LowerCase}}Id uint) (*models.{{.PascalCase}}, error) {
	requestID := utils.GetRequestIDFromContext(ctx)

	logger.Debug("Finding {{.LowerCase}} by ID",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	{{.LowerCase}}, err := s.{{.CamelCase}}Repository.FindById(ctx, {{.LowerCase}}Id)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			logger.Error("Failed to find {{.LowerCase}} by ID",
				zap.String("request_id", requestID),
				zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
				zap.Error(err),
			)
		}
		return {{.LowerCase}}, err
	}

	logger.Debug("Found {{.LowerCase}} by ID successfully",
		zap.String("request_id", requestID),
		zap.Uint("{{.LowerCase}}_id", {{.LowerCase}}Id),
	)

	return {{.LowerCase}}, nil
}
`

// repositoryTemplate defines the repository file template
const repositoryTemplate = `package repository

import (
	"github.com/PhantomX7/athletonnal/models"
	"github.com/PhantomX7/athletonepository"

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
