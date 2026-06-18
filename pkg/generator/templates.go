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
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/controller"
	"github.com/PhantomX7/athleton/internal/routes"
)

type routeRegistrar struct {
	controller controller.{{.PascalCase}}Controller
}

// NewRoutes constructs the {{.KebabCase}} route registrar.
func NewRoutes(controller controller.{{.PascalCase}}Controller) routes.Registrar {
	return &routeRegistrar{controller: controller}
}

// RegisterRoutes mounts the {{.KebabCase}} endpoints.
func (r *routeRegistrar) RegisterRoutes(ctx *routes.Context) {
	{{.CamelCase}}Route := ctx.Admin.Group("/{{.KebabCase}}")
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

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/service"
	"github.com/PhantomX7/athleton/pkg/ginx"
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
// Columns are the typed helpers from internal/generated (run ` + "`make gorm-gen`" + ` after
// adding the model), so a column rename breaks this registration at compile time.
func New{{.PascalCase}}Pagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Column:    generated.{{.PascalCase}}.Name,
			TableName: "{{.TableName}}",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Column:    generated.{{.PascalCase}}.CreatedAt,
			TableName: "{{.TableName}}",
			Type:      pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Column: generated.{{.PascalCase}}.ID, Allowed: true}).
		AddSort("name", pagination.SortConfig{Column: generated.{{.PascalCase}}.Name, Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Column: generated.{{.PascalCase}}.CreatedAt, Allowed: true})

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
	{{.LowerCase}}ID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	var req dto.{{.PascalCase}}UpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.Update(ctx.Request.Context(), {{.LowerCase}}ID, &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} updated successfully", {{.LowerCase}}.ToResponse()))
}

// Delete handles deletion of an existing {{.LowerCase}}.
func (c *{{.CamelCase}}Controller) Delete(ctx *gin.Context) {
	{{.LowerCase}}ID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	if err := c.{{.CamelCase}}Service.Delete(ctx.Request.Context(), {{.LowerCase}}ID); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("{{.PascalCase}} deleted successfully", nil))
}

// FindByID handles fetching a single {{.LowerCase}} by ID.
func (c *{{.CamelCase}}Controller) FindByID(ctx *gin.Context) {
	{{.LowerCase}}ID, ok := ginx.ParseUintParam(ctx, "id")
	if !ok {
		return
	}

	{{.LowerCase}}, err := c.{{.CamelCase}}Service.FindByID(ctx.Request.Context(), {{.LowerCase}}ID)
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

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/repository"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
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
	{{.LowerCase}}s, err := s.{{.CamelCase}}Repository.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.{{.CamelCase}}Repository.Count(ctx, pg)
	if err != nil {
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
	entity := &models.{{.PascalCase}}{
		// TODO: map fields from req explicitly, e.g. Name: req.Name
	}

	if err := s.{{.CamelCase}}Repository.Create(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Update changes an existing {{.LowerCase}} resource. Request fields are pointers,
// so an omitted field (nil) keeps its current value — PATCH semantics.
func (s *{{.CamelCase}}Service) Update(ctx context.Context, {{.CamelCase}}ID uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
		return nil, err
	}

	// TODO: apply req fields onto entity explicitly, e.g.:
	//   if req.Name != nil { entity.Name = *req.Name }

	if err := s.{{.CamelCase}}Repository.Update(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes an existing {{.LowerCase}} resource.
func (s *{{.CamelCase}}Service) Delete(ctx context.Context, {{.CamelCase}}ID uint) error {
	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
		return err
	}

	return s.{{.CamelCase}}Repository.Delete(ctx, entity)
}

// FindByID returns one {{.LowerCase}} resource by ID.
func (s *{{.CamelCase}}Service) FindByID(ctx context.Context, {{.CamelCase}}ID uint) (*models.{{.PascalCase}}, error) {
	entity, err := s.{{.CamelCase}}Repository.FindByID(ctx, {{.CamelCase}}ID)
	if err != nil {
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

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/controller"
	{{.LowerCase}}service "github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"
)

type mock{{.PascalCase}}Service struct {
	indexFn    func(context.Context, *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error)
	createFn   func(context.Context, *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error)
	updateFn   func(context.Context, uint, *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error)
	deleteFn   func(context.Context, uint) error
	findByIDFn func(context.Context, uint) (*models.{{.PascalCase}}, error)
}

func (m *mock{{.PascalCase}}Service) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error) {
	if m.indexFn == nil {
		panic("unexpected Index call")
	}
	return m.indexFn(ctx, pg)
}

func (m *mock{{.PascalCase}}Service) Create(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error) {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, req)
}

func (m *mock{{.PascalCase}}Service) Update(ctx context.Context, {{.CamelCase}}ID uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, {{.CamelCase}}ID, req)
}

func (m *mock{{.PascalCase}}Service) Delete(ctx context.Context, {{.CamelCase}}ID uint) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, {{.CamelCase}}ID)
}

func (m *mock{{.PascalCase}}Service) FindByID(ctx context.Context, {{.CamelCase}}ID uint) (*models.{{.PascalCase}}, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, {{.CamelCase}}ID)
}

var _ {{.LowerCase}}service.{{.PascalCase}}Service = (*mock{{.PascalCase}}Service)(nil)

func Test{{.PascalCase}}ControllerIndexReturnsPaginatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		indexFn: func(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error) {
			require.NotNil(t, ctx)
			require.Equal(t, 2, pg.Limit)
			require.Equal(t, 4, pg.Offset)
			return []*models.{{.PascalCase}}{
					{Name: "first"},
				}, response.Meta{
					Total:  7,
					Offset: 4,
					Limit:  2,
				}, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/{{.KebabCase}}?limit=2&offset=4", nil)

	ctrl.Index(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body["status"])
	require.Equal(t, "Success", body["message"])
}

func Test{{.PascalCase}}ControllerIndexPropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("service failed")
	svc := &mock{{.PascalCase}}Service{
		indexFn: func(context.Context, *pagination.Pagination) ([]*models.{{.PascalCase}}, response.Meta, error) {
			return nil, response.Meta{}, expectedErr
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/{{.KebabCase}}", nil)

	ctrl.Index(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
	require.ErrorIs(t, ctx.Errors[0].Err, expectedErr)
}

func Test{{.PascalCase}}ControllerCreateReturnsCreatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		createFn: func(ctx context.Context, req *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error) {
			require.Equal(t, "New Name", req.Name)
			require.Equal(t, "content", req.Description)
			created := &models.{{.PascalCase}}{Name: req.Name, Description: req.Description}
			created.ID = 3
			return created, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/{{.KebabCase}}",
		strings.NewReader(` + "`" + `{"name":"New Name","description":"content"}` + "`" + `))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrl.Create(ctx)

	require.Equal(t, http.StatusCreated, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "{{.PascalCase}} created successfully", body["message"])
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), data["id"])
}

func Test{{.PascalCase}}ControllerCreateRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		createFn: func(context.Context, *dto.{{.PascalCase}}CreateRequest) (*models.{{.PascalCase}}, error) {
			t.Fatal("Create should not be called for invalid payloads")
			return nil, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/{{.KebabCase}}",
		strings.NewReader(` + "`" + `{"description":"missing required name"}` + "`" + `))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	ctrl.Create(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypeBind))
}

func Test{{.PascalCase}}ControllerUpdateReturnsUpdatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		updateFn: func(ctx context.Context, {{.CamelCase}}ID uint, req *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
			require.Equal(t, uint(5), {{.CamelCase}}ID)
			require.NotNil(t, req.Name)
			require.Equal(t, "Renamed", *req.Name)
			updated := &models.{{.PascalCase}}{Name: *req.Name}
			updated.ID = {{.CamelCase}}ID
			return updated, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/{{.KebabCase}}/5",
		strings.NewReader(` + "`" + `{"name":"Renamed"}` + "`" + `))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Params = gin.Params{{"{{"}}Key: "id", Value: "5"}}

	ctrl.Update(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "{{.PascalCase}} updated successfully", body["message"])
}

func Test{{.PascalCase}}ControllerUpdateRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		updateFn: func(context.Context, uint, *dto.{{.PascalCase}}UpdateRequest) (*models.{{.PascalCase}}, error) {
			t.Fatal("Update should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/{{.KebabCase}}/bad", nil)
	ctx.Params = gin.Params{{"{{"}}Key: "id", Value: "bad"}}

	ctrl.Update(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}

func Test{{.PascalCase}}ControllerDeleteReturnsSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		deleteFn: func(ctx context.Context, {{.CamelCase}}ID uint) error {
			require.Equal(t, uint(8), {{.CamelCase}}ID)
			return nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/{{.KebabCase}}/8", nil)
	ctx.Params = gin.Params{{"{{"}}Key: "id", Value: "8"}}

	ctrl.Delete(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "{{.PascalCase}} deleted successfully", body["message"])
}

func Test{{.PascalCase}}ControllerFindByIDReturns{{.PascalCase}}(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		findByIDFn: func(ctx context.Context, {{.CamelCase}}ID uint) (*models.{{.PascalCase}}, error) {
			require.Equal(t, uint(2), {{.CamelCase}}ID)
			found := &models.{{.PascalCase}}{Name: "Found"}
			found.ID = 2
			return found, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/{{.KebabCase}}/2", nil)
	ctx.Params = gin.Params{{"{{"}}Key: "id", Value: "2"}}

	ctrl.FindByID(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "{{.PascalCase}} found successfully", body["message"])
}

func Test{{.PascalCase}}ControllerFindByIDRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mock{{.PascalCase}}Service{
		findByIDFn: func(context.Context, uint) (*models.{{.PascalCase}}, error) {
			t.Fatal("FindByID should not be called for invalid ids")
			return nil, nil
		},
	}

	ctrl := controller.New{{.PascalCase}}Controller(svc)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/{{.KebabCase}}/bad", nil)
	ctx.Params = gin.Params{{"{{"}}Key: "id", Value: "bad"}}

	ctrl.FindByID(ctx)

	require.Len(t, ctx.Errors, 1)
	require.True(t, ctx.Errors[0].IsType(gin.ErrorTypePublic))
}
`

const serviceTestTemplate = `package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	{{.LowerCase}}repository "github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/repository"
	"github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/service"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
)

type mock{{.PascalCase}}Repository struct {
	createFn   func(context.Context, *models.{{.PascalCase}}) error
	updateFn   func(context.Context, *models.{{.PascalCase}}) error
	deleteFn   func(context.Context, *models.{{.PascalCase}}) error
	findByIDFn func(context.Context, uint, ...repository.Association) (*models.{{.PascalCase}}, error)
	findAllFn  func(context.Context, *pagination.Pagination) ([]*models.{{.PascalCase}}, error)
	countFn    func(context.Context, *pagination.Pagination) (int64, error)
}

func (m *mock{{.PascalCase}}Repository) Create(ctx context.Context, entity *models.{{.PascalCase}}) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}

func (m *mock{{.PascalCase}}Repository) Update(ctx context.Context, entity *models.{{.PascalCase}}) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}

func (m *mock{{.PascalCase}}Repository) Delete(ctx context.Context, entity *models.{{.PascalCase}}) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, entity)
}

func (m *mock{{.PascalCase}}Repository) FindByID(ctx context.Context, id uint, preloads ...repository.Association) (*models.{{.PascalCase}}, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mock{{.PascalCase}}Repository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.{{.PascalCase}}, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mock{{.PascalCase}}Repository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
}

var _ {{.LowerCase}}repository.{{.PascalCase}}Repository = (*mock{{.PascalCase}}Repository)(nil)

func strPtr(s string) *string {
	return &s
}

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func Test{{.PascalCase}}ServiceIndexReturnsEntitiesAndMeta(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(map[string][]string{"limit": {"2"}}, nil, pagination.PaginationOptions{})
	repo := &mock{{.PascalCase}}Repository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.{{.PascalCase}}, error) {
			require.Same(t, pg, gotPg)
			return []*models.{{.PascalCase}}{
				{Name: "first"},
				{Name: "second"},
			}, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Same(t, pg, gotPg)
			return 5, nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}s, meta, err := svc.Index(context.Background(), pg)

	require.NoError(t, err)
	require.Len(t, {{.CamelCase}}s, 2)
	require.Equal(t, int64(5), meta.Total)
	require.Equal(t, pg.Limit, meta.Limit)
	require.Equal(t, pg.Offset, meta.Offset)
}

func Test{{.PascalCase}}ServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mock{{.PascalCase}}Repository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.{{.PascalCase}}, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}s, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, {{.CamelCase}}s)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}

func Test{{.PascalCase}}ServiceCreateCopiesRequestAndPersists(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		createFn: func(ctx context.Context, entity *models.{{.PascalCase}}) error {
			require.Equal(t, "New Name", entity.Name)
			require.Equal(t, "fresh content", entity.Description)
			entity.ID = 9
			return nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.Create(context.Background(), &dto.{{.PascalCase}}CreateRequest{
		Name:        "New Name",
		Description: "fresh content",
	})

	require.NoError(t, err)
	require.NotNil(t, {{.CamelCase}})
	require.Equal(t, uint(9), {{.CamelCase}}.ID)
}

func Test{{.PascalCase}}ServiceCreateReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("insert failed")
	repo := &mock{{.PascalCase}}Repository{
		createFn: func(context.Context, *models.{{.PascalCase}}) error {
			return expectedErr
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.Create(context.Background(), &dto.{{.PascalCase}}CreateRequest{Name: "New Name"})

	require.Nil(t, {{.CamelCase}})
	require.ErrorIs(t, err, expectedErr)
}

func Test{{.PascalCase}}ServiceUpdateAppliesChanges(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.{{.PascalCase}}, error) {
			require.Equal(t, uint(4), id)
			existing := &models.{{.PascalCase}}{Name: "Old", Description: "old text"}
			existing.ID = 4
			return existing, nil
		},
		updateFn: func(ctx context.Context, entity *models.{{.PascalCase}}) error {
			require.Equal(t, uint(4), entity.ID)
			require.Equal(t, "Updated", entity.Name)
			// Omitted fields (nil pointers) must keep their current value.
			require.Equal(t, "old text", entity.Description)
			return nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.Update(context.Background(), 4, &dto.{{.PascalCase}}UpdateRequest{Name: strPtr("Updated")})

	require.NoError(t, err)
	require.NotNil(t, {{.CamelCase}})
	require.Equal(t, "Updated", {{.CamelCase}}.Name)
	require.Equal(t, "old text", {{.CamelCase}}.Description)
}

func Test{{.PascalCase}}ServiceUpdatePropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.{{.PascalCase}}, error) {
			return nil, cerrors.NewNotFoundError("{{.LowerCase}} not found")
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.Update(context.Background(), 4, &dto.{{.PascalCase}}UpdateRequest{Name: strPtr("Updated")})

	require.Nil(t, {{.CamelCase}})
	require.ErrorIs(t, err, cerrors.ErrNotFound)
}

func Test{{.PascalCase}}ServiceDeleteRemovesEntity(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.{{.PascalCase}}, error) {
			existing := &models.{{.PascalCase}}{Name: "Doomed"}
			existing.ID = id
			return existing, nil
		},
		deleteFn: func(ctx context.Context, entity *models.{{.PascalCase}}) error {
			require.Equal(t, uint(6), entity.ID)
			return nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	require.NoError(t, svc.Delete(context.Background(), 6))
}

func Test{{.PascalCase}}ServiceDeletePropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.{{.PascalCase}}, error) {
			return nil, cerrors.NewNotFoundError("{{.LowerCase}} not found")
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	err := svc.Delete(context.Background(), 6)

	require.ErrorIs(t, err, cerrors.ErrNotFound)
}

func Test{{.PascalCase}}ServiceFindByIDReturnsEntity(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.{{.PascalCase}}, error) {
			require.Equal(t, uint(2), id)
			existing := &models.{{.PascalCase}}{Name: "Found"}
			existing.ID = 2
			return existing, nil
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.FindByID(context.Background(), 2)

	require.NoError(t, err)
	require.Equal(t, "Found", {{.CamelCase}}.Name)
}

func Test{{.PascalCase}}ServiceFindByIDPropagatesNotFound(t *testing.T) {
	setupLogger(t)

	repo := &mock{{.PascalCase}}Repository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.{{.PascalCase}}, error) {
			return nil, cerrors.NewNotFoundError("{{.LowerCase}} not found")
		},
	}

	svc := service.New{{.PascalCase}}Service(repo)

	{{.CamelCase}}, err := svc.FindByID(context.Background(), 2)

	require.Nil(t, {{.CamelCase}})
	require.ErrorIs(t, err, cerrors.ErrNotFound)
}
`

const repositoryTestTemplate = `package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/internal/models"
	{{.LowerCase}}repository "github.com/PhantomX7/athleton/internal/modules/{{.SnakeCase}}/repository"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.{{.PascalCase}}{}))

	return db
}

func Test{{.PascalCase}}RepositoryCreatePersistsEntity(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	{{.CamelCase}} := &models.{{.PascalCase}}{Name: "First", Description: "hello", IsActive: true}

	require.NoError(t, repo.Create(context.Background(), {{.CamelCase}}))
	require.NotZero(t, {{.CamelCase}}.ID)

	var stored models.{{.PascalCase}}
	require.NoError(t, db.First(&stored, {{.CamelCase}}.ID).Error)
	require.Equal(t, "First", stored.Name)
	require.Equal(t, "hello", stored.Description)
}

func Test{{.PascalCase}}RepositoryFindByIDReturnsEntity(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	seed := &models.{{.PascalCase}}{Name: "Seeded", Description: "seeded entity"}
	require.NoError(t, db.Create(seed).Error)

	got, err := repo.FindByID(context.Background(), seed.ID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, seed.ID, got.ID)
	require.Equal(t, "Seeded", got.Name)
}

func Test{{.PascalCase}}RepositoryFindByIDReturnsNotFound(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	got, err := repo.FindByID(context.Background(), 999)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func Test{{.PascalCase}}RepositoryUpdateChangesEntity(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	seed := &models.{{.PascalCase}}{Name: "Old Name"}
	require.NoError(t, db.Create(seed).Error)

	seed.Name = "New Name"
	require.NoError(t, repo.Update(context.Background(), seed))

	var stored models.{{.PascalCase}}
	require.NoError(t, db.First(&stored, seed.ID).Error)
	require.Equal(t, "New Name", stored.Name)
}

func Test{{.PascalCase}}RepositoryDeleteRemovesEntity(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	seed := &models.{{.PascalCase}}{Name: "Doomed"}
	require.NoError(t, db.Create(seed).Error)

	require.NoError(t, repo.Delete(context.Background(), seed))

	_, err := repo.FindByID(context.Background(), seed.ID)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func Test{{.PascalCase}}RepositoryFindAllAndCountHonorPagination(t *testing.T) {
	db := setupDB(t)
	repo := {{.LowerCase}}repository.New{{.PascalCase}}Repository(db)

	for _, name := range []string{"a", "b", "c"} {
		require.NoError(t, db.Create(&models.{{.PascalCase}}{Name: name}).Error)
	}

	pg := pagination.NewPagination(map[string][]string{
		"limit":  {"2"},
		"offset": {"0"},
	}, nil, pagination.PaginationOptions{DefaultLimit: 20, MaxLimit: 100, DefaultOrder: "id asc"})

	{{.CamelCase}}s, err := repo.FindAll(context.Background(), pg)
	require.NoError(t, err)
	require.Len(t, {{.CamelCase}}s, 2)

	count, err := repo.Count(context.Background(), pg)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}
`
