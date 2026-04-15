package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/modules/log/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

type LogController interface {
	Index(ctx *gin.Context)
	FindById(ctx *gin.Context)
}

type logController struct {
	logService service.LogService
}

func NewLogController(logService service.LogService) LogController {
	return &logController{
		logService: logService,
	}
}

// newLogPagination creates a new pagination instance for logs
func newLogPagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("user_id", pagination.FilterConfig{
			Field:     "user_id",
			TableName: "logs",
			Type:      pagination.FilterTypeID,
		}).
		AddFilter("action", pagination.FilterConfig{
			Field:     "action",
			TableName: "logs",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("entity_id", pagination.FilterConfig{
			Field:     "entity_id",
			TableName: "logs",
			Type:      pagination.FilterTypeID,
		}).
		AddFilter("entity_type", pagination.FilterConfig{
			Field:     "entity_type",
			TableName: "logs",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("message", pagination.FilterConfig{
			Field:     "message",
			TableName: "logs",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultOrder: "id desc",
	})
}

// Index handles the listing of logs with pagination
func (c *logController) Index(ctx *gin.Context) {
	logs, meta, err := c.logService.Index(
		ctx.Request.Context(),
		newLogPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK,
		response.BuildPaginationResponse(logs, meta))
}

// @Summary      Find a log by ID
// @Description  Find a log with the provided ID
// @Tags         log
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Log ID"
// @Success      200  {object}  response.Response{data=dto.LogResponse}
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /log/{id} [get]
func (c *logController) FindById(ctx *gin.Context) {
	logID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	log, err := c.logService.FindById(ctx.Request.Context(), uint(logID))
	if err != nil {
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Log found successfully", log.ToResponse()))
}
