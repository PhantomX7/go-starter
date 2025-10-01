package controller

import (
	"net/http"
	"strconv"

	// _ "github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/post/dto"
	"github.com/PhantomX7/go-starter/internal/modules/post/service"
	"github.com/PhantomX7/go-starter/pkg/utils"

	"github.com/gin-gonic/gin"
)

type PostController interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindById(ctx *gin.Context)
}

type postController struct {
	postService service.PostService
}

func NewPostController(postService service.PostService) PostController {
	return &postController{
		postService: postService,
	}
}

// @Summary      Create a new post
// @Description  Create a new post with the provided details
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        post  body      dto.PostCreateRequest  true  "Post Create Request"
// @Success      201  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /posts [post]
func (c *postController) Create(ctx *gin.Context) {
	var req dto.PostCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post, err := c.postService.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("Post created successfully", post.ToResponse()))
}

// @Summary      Update a post
// @Description  Update a post with the provided details
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Param        post  body      dto.PostUpdateRequest  true  "Post Update Request"
// @Success      200  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /posts/{id} [put]
func (c *postController) Update(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req dto.PostUpdateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post, err := c.postService.Update(ctx.Request.Context(), uint(postID), &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post updated successfully", post))
}

// @Summary      Delete a post
// @Description  Delete a post with the provided ID
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Success      200  {object}  utils.Response{data=nil}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /posts/{id} [delete]
func (c *postController) Delete(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = c.postService.Delete(ctx.Request.Context(), uint(postID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post deleted successfully", nil))
}

// @Summary      Find a post by ID
// @Description  Find a post with the provided ID
// @Tags         posts
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Success      200  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /posts/{id} [get]
func (c *postController) FindById(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	post, err := c.postService.FindById(ctx.Request.Context(), uint(postID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post found successfully", post))
}
