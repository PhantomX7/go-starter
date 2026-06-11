// Package controller exposes HTTP handlers for post management.
package controller

import (
	"net/http"
	"strconv"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/generated"
	"github.com/PhantomX7/athleton/internal/modules/post/service"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// PostController exposes HTTP handlers for post resources.
type PostController interface {
	Index(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindByID(ctx *gin.Context)
}

type postController struct {
	postService service.PostService
}

// NewPostController constructs a PostController.
func NewPostController(postService service.PostService) PostController {
	return &postController{
		postService: postService,
	}
}

// NewPostPagination creates a new pagination instance for posts.
// Columns are the typed helpers from internal/generated (run `make gorm-gen`
// after model changes), so a column rename breaks this registration at compile time.
func NewPostPagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Column:    generated.Post.Name,
			TableName: "posts",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Column:    generated.Post.CreatedAt,
			TableName: "posts",
			Type:      pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Column: generated.Post.ID, Allowed: true}).
		AddSort("name", pagination.SortConfig{Column: generated.Post.Name, Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Column: generated.Post.CreatedAt, Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     100,
		DefaultOrder: "id desc",
	})
}

// Index handles the listing of posts with pagination.
func (c *postController) Index(ctx *gin.Context) {
	posts, meta, err := c.postService.Index(
		ctx.Request.Context(),
		NewPostPagination(ctx.Request.URL.Query()),
	)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	ctx.JSON(http.StatusOK, response.BuildPaginationResponse(posts, meta))
}

// Create handles the creation of a new post.
func (c *postController) Create(ctx *gin.Context) {
	var req dto.PostCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	post, err := c.postService.Create(ctx.Request.Context(), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusCreated, response.BuildResponseSuccess("Post created successfully", post.ToResponse()))
}

// Update handles updates to an existing post.
func (c *postController) Update(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.PostUpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	post, err := c.postService.Update(ctx.Request.Context(), uint(postID), &req)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Post updated successfully", post.ToResponse()))
}

// Delete handles deletion of an existing post.
func (c *postController) Delete(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	if err := c.postService.Delete(ctx.Request.Context(), uint(postID)); err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Post deleted successfully", nil))
}

// FindByID handles fetching a single post by ID.
func (c *postController) FindByID(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	post, err := c.postService.FindByID(ctx.Request.Context(), uint(postID))
	if err != nil {
		_ = ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	ctx.JSON(http.StatusOK, response.BuildResponseSuccess("Post found successfully", post.ToResponse()))
}
