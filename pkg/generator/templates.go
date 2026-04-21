package generator

const moduleTemplate = `// Package {{.SnakeCase}} wires the {{.KebabCase}} module.
package {{.SnakeCase}}

import (
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/controller"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/repository"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/service"
	"github.com/PhantomX7/athleton/internal/routes"

	"go.uber.org/fx"
)

// Module wires the {{.KebabCase}} module dependencies into the Fx container.
var Module = fx.Options(
	fx.Provide(
		controller.New{{.PascalCase}}Controller,
		service.New{{.PascalCase}}Service,
		repository.New{{.PascalCase}}Repository,
		fx.Annotate(
			NewRoutes,
			fx.As(new(routes.Registrar)),
			fx.ResultTags(` + "`group:\"routes\"`" + `),
		),
	),
)
`

const routesTemplate = `// Package {{.SnakeCase}} wires the {{.KebabCase}} module.
package {{.SnakeCase}}

import (
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/controller"
	"github.com/PhantomX7/athleton/internal/routes"

	"github.com/gin-gonic/gin"
)

type routeRegistrar struct {
	controller controller.{{.PascalCase}}Controller
}

// NewRoutes constructs the {{.KebabCase}} route registrar.
func NewRoutes(controller controller.{{.PascalCase}}Controller) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the {{.KebabCase}} endpoints.
func (r *routeRegistrar) RegisterRoutes(api *gin.RouterGroup, middleware *middlewares.Middleware) {
	adminAPI := api.Group("/admin", middleware.RequireAuth())
	{{.CamelCase}}Route := adminAPI.Group("/{{.KebabCase}}")
	{{.CamelCase}}Route.GET("", r.controller.Index)
	{{.CamelCase}}Route.GET("/:id", r.controller.FindByID)
	{{.CamelCase}}Route.POST("", r.controller.Create)
	{{.CamelCase}}Route.PATCH("/:id", r.controller.Update)
	{{.CamelCase}}Route.DELETE("/:id", r.controller.Delete)
}
`

const controllerTemplate = `// Package controller exposes HTTP handlers for {{.KebabCase}} management.
package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// {{.PascalCase}}Controller exposes HTTP handlers for {{.KebabCase}} resources.
type {{.PascalCase}}Controller interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindByID(ctx *gin.Context)
}

type {{.CamelCase}}Controller struct {
	{{.CamelCase}}Service service.{{.PascalCase}}Service
}

// New{{.PascalCase}}Controller constructs a {{.PascalCase}}Controller.
func New{{.PascalCase}}Controller({{.CamelCase}}Service service.{{.PascalCase}}Service) {{.PascalCase}}Controller {
	return &{{.CamelCase}}Controller{
		{{.CamelCase}}Service: {{.CamelCase}}Service,
	}
}

// New{{.PascalCase}}Pagination creates a new pagination instance for {{.LowerCase}}s.
func New{{.PascalCase}}Pagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field: "{{.LowerCase}}s.name",
			Type:  pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "{{.LowerCase}}s.created_at",
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

// Index handles the listing of {{.LowerCase}}s with pagination.
func (c *{{.CamelCase}}Controller) Index(ctx *gin.Context) {
	{{.LowerCase}}s, meta, err := c.{{.CamelCase}}Service.Index(
		ctx.Request.Context(),
		New{{.PascalCase}}Pagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildPaginationResponse({{.LowerCase}}s, meta))
}

// Create handles the creation of a new {{.LowerCase}}.
func (c *{{.CamelCase}}Controller) Create(ctx *gin.Context) {
	var req dto.{{.PascalCase}}CreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Create(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("{{.PascalCase}} created successfully", {{.LowerCase}}.ToResponse()))
}

// Update handles updates to an existing {{.LowerCase}}.
func (c *{{.CamelCase}}Controller) Update(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.{{.PascalCase}}UpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Update(ctx.Request.Context(), uint({{.LowerCase}}ID), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} updated successfully", {{.LowerCase}}.ToResponse()))
}

// Delete handles deletion of an existing {{.LowerCase}}.
func (c *{{.CamelCase}}Controller) Delete(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	if err := c.{{.CamelCase}}Service.Delete(ctx.Request.Context(), uint({{.LowerCase}}ID)); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} deleted successfully", nil))
}

// FindByID handles fetching a single {{.LowerCase}} by ID.
func (c *{{.CamelCase}}Controller) FindByID(ctx *gin.Context) {
	{{.LowerCase}}ID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.FindByID(ctx.Request.Context(), uint({{.LowerCase}}ID))
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} found successfully", {{.LowerCase}}.ToResponse()))
}
`

const serviceTemplate = `// Package service contains the {{.KebabCase}} business logic.
package service

import (
	"context"
	"errors"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

// {{.PascalCase}}Service defines the business operations for {{.KebabCase}} resources.
type {{.PascalCase}}Service interface {
	Index(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error)
	Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error)
	Update(ctx context.Context, {{.CamelCase}}ID uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error)
	Delete(ctx context.Context, {{.CamelCase}}ID uint) error
	FindByID(ctx context.Context, {{.CamelCase}}ID uint) (*models.{{.PascalCase}}, error)
}

type {{.CamelCase}}Service struct {
	{{.CamelCase}}Repository repository.{{.PascalCase}}Repository
}

// New{{.PascalCase}}Service constructs a {{.PascalCase}}Service.
func New{{.PascalCase}}Service({{.CamelCase}}Repository repository.{{.PascalCase}}Repository) {{.PascalCase}}Service {
	return &{{.CamelCase}}Service{
		{{.CamelCase}}Repository: {{.CamelCase}}Repository,
	}
}

// Index returns a paginated collection of {{.LowerCase}} resources.
func (s *{{.CamelCase}}Service) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error) {
	log := logger.Ctx(ctx)
	log.Info("Fetching {{.LowerCase}}s", zap.Int("page", pg.GetPage()), zap.Int("limit", pg.Limit), zap.Int("offset", pg.Offset))

	{{.LowerCase}}s, err := s.{{.CamelCase}}Repository.FindAll(ctx, pg)
	if err != nil {
		log.Error("Failed to fetch {{.LowerCase}}s", zap.Error(err))
		return nil, response.Meta{}, err
	}

	count, err := s.{{.CamelCase}}Repository.Count(ctx, pg)
	if err != nil {
		log.Error("Failed to count {{.LowerCase}}s", zap.Error(err))
		return nil, response.Meta{}, err
	}

	return {{.LowerCase}}s, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create persists a new {{.LowerCase}} resource.
func (s *{{.CamelCase}}Service) Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error) {
	log := logger.Ctx(ctx)
	entity := &models.{{.PascalCase}}{}

	if err := copier.Copy(entity, req); err != nil {
		log.Error("Failed to copy {{.LowerCase}} payload", zap.Error(err))
		return nil, err
	}

	if err := s.{{.CamelCase}}Repository.Create(ctx, entity); err != nil {
		log.Error("Failed to create {{.LowerCase}}", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Update changes an existing {{.LowerCase}} resource.
func (s *{{.CamelCase}}Service) Update(ctx context.Context, {{.CamelCase}}ID uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
	log := logger.Ctx(ctx, zap.Uint("{{.LowerCase}}_id", {{.CamelCase}}ID))

	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
		log.Error("Failed to find {{.LowerCase}} for update", zap.Error(err))
		return nil, err
	}

	if err := copier.Copy(entity, req); err != nil {
		log.Error("Failed to copy {{.LowerCase}} payload", zap.Error(err))
		return nil, err
	}

	if err := s.{{.CamelCase}}Repository.Update(ctx, entity); err != nil {
		log.Error("Failed to update {{.LowerCase}}", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Delete removes an existing {{.LowerCase}} resource.
func (s *{{.CamelCase}}Service) Delete(ctx context.Context, {{.CamelCase}}ID uint) error {
	log := logger.Ctx(ctx, zap.Uint("{{.LowerCase}}_id", {{.CamelCase}}ID))

	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
		log.Error("Failed to find {{.LowerCase}} for deletion", zap.Error(err))
		return err
	}

	if err := s.{{.CamelCase}}Repository.Delete(ctx, entity); err != nil {
		log.Error("Failed to delete {{.LowerCase}}", zap.Error(err))
		return err
	}

	return nil
}

// FindByID returns one {{.LowerCase}} resource by ID.
func (s *{{.CamelCase}}Service) FindByID(ctx context.Context, {{.CamelCase}}ID uint) (*models.{{.PascalCase}}, error) {
	log := logger.Ctx(ctx, zap.Uint("{{.LowerCase}}_id", {{.CamelCase}}ID))

	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
		if !errors.Is(err, cerrors.ErrNotFound) {
			log.Error("Failed to find {{.LowerCase}}", zap.Error(err))
		}
		return nil, err
	}

	return entity, nil
}
`

const repositoryTemplate = `// Package repository provides {{.KebabCase}} persistence primitives.
package repository

import (
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// {{.PascalCase}}Repository defines the persistence operations for {{.KebabCase}} resources.
type {{.PascalCase}}Repository interface {
	repository.Repository[models.{{.PascalCase}}]
}

type {{.CamelCase}}Repository struct {
	repository.BaseRepository[models.{{.PascalCase}}]
}

// New{{.PascalCase}}Repository constructs a {{.PascalCase}}Repository.
func New{{.PascalCase}}Repository(db *gorm.DB) {{.PascalCase}}Repository {
	return &{{.CamelCase}}Repository{
		BaseRepository: repository.NewBaseRepository[models.{{.PascalCase}}](db),
	}
}
`

const controllerTestTemplate = `package controller_test

import "testing"

func Test{{.PascalCase}}ControllerPlaceholder(t *testing.T) {
	t.Skip("TODO: implement controller tests for {{.PascalCase}}")
}
`

const serviceTestTemplate = `package service_test

import "testing"

func Test{{.PascalCase}}ServicePlaceholder(t *testing.T) {
	t.Skip("TODO: implement service tests for {{.PascalCase}}")
}
`

const repositoryTestTemplate = `package repository_test

import "testing"

func Test{{.PascalCase}}RepositoryPlaceholder(t *testing.T) {
	t.Skip("TODO: implement repository tests for {{.PascalCase}}")
}
`
