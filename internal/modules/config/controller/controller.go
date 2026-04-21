// Package controller exposes HTTP handlers for configuration management.
package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/modules/config/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// ConfigController exposes HTTP handlers for configuration resources.
type ConfigController interface {
	Index(ctx *gin.Context)
	Update(ctx *gin.Context)
	FindByKey(ctx *gin.Context)
}

type configController struct {
	configService service.ConfigService
}

// NewConfigController constructs a ConfigController.
func NewConfigController(configService service.ConfigService) ConfigController {
	return &configController{
		configService: configService,
	}
}

// newConfigPagination creates a new pagination instance for configs
func newConfigPagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("key", pagination.FilterConfig{
			Field: "key",
			Type:  pagination.FilterTypeString,
		}).
		AddSort("key", pagination.SortConfig{Field: "key", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultOrder: "id desc",
	})
}

// @Summary      List configs
// @Description  Get a paginated list of configs
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        sort    query     string  false  "Sort"
// @Param        key     query     string  false  "Filter by key"
// @Param        group   query     string  false  "Filter by group"
// @Success      200  {object}  response.Response{data=[]dto.ConfigResponse,meta=response.Meta}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /config [get]
func (c *configController) Index(ctx *gin.Context) {
	configs, meta, err := c.configService.Index(
		ctx.Request.Context(),
		newConfigPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse(configs, meta))
}

// @Summary      Update a config
// @Description  Update a config with the provided details
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        id      path      uint                     true  "Config ID"
// @Param        config  body      dto.ConfigUpdateRequest  true  "Config Update Request"
// @Success      200  {object}  response.Response{data=dto.ConfigResponse}
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /config/{id} [put]
func (c *configController) Update(ctx *gin.Context) {
	configID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.ConfigUpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	config, err := c.configService.Update(ctx.Request.Context(), uint(configID), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Config updated successfully", config.ToResponse()))
}

// @Summary      Find a config by key
// @Description  Find a config with the provided key
// @Tags         config
// @Accept       json
// @Produce      json
// @Param        key   path      string  true  "Config Key"
// @Success      200  {object}  response.Response{data=dto.ConfigResponse}
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /config/key/{key} [get]
func (c *configController) FindByKey(ctx *gin.Context) {
	config, err := c.configService.FindByKey(ctx.Request.Context(), ctx.Param("key"))
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Config found successfully", config.ToResponse()))
}
