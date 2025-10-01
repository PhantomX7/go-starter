package controller

import (
	"net/http"
	"strconv"

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
// @Tags         post
// @Accept       json
// @Produce      json
// @Param        post  body      dto.PostCreateRequest  true  "Post Create Request"
// @Success      201  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /post [post]
func (c *postController) Create(ctx *gin.Context) {
	var req dto.PostCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	post, err := c.postService.Create(ctx.Request.Context(), &req)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("Post created successfully", post.ToResponse()))
}

// @Summary      Update a post
// @Description  Update a post with the provided details
// @Tags         post
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Param        post  body      dto.PostUpdateRequest  true  "Post Update Request"
// @Success      200  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /post/{id} [put]
func (c *postController) Update(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	var req dto.PostUpdateRequest
	if err = ctx.ShouldBind(&req); err != nil {
		// Let the error middleware handle the validation error
		ctx.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	post, err := c.postService.Update(ctx.Request.Context(), uint(postID), &req)
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post updated successfully", post))
}

// @Summary      Delete a post
// @Description  Delete a post with the provided ID
// @Tags         post
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Success      200  {object}  utils.Response{data=nil}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /post/{id} [delete]
func (c *postController) Delete(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	err = c.postService.Delete(ctx.Request.Context(), uint(postID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post deleted successfully", nil))
}

// @Summary      Find a post by ID
// @Description  Find a post with the provided ID
// @Tags         post
// @Accept       json
// @Produce      json
// @Param        id    path      uint                  true  "Post ID"
// @Success      200  {object}  utils.Response{data=dto.PostResponse}
// @Failure      400  {object}  utils.Response
// @Failure      500  {object}  utils.Response
// @Router       /post/{id} [get]
func (c *postController) FindById(ctx *gin.Context) {
	postID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		// Let the error middleware handle the parameter parsing error
		ctx.Error(err).SetType(gin.ErrorTypePublic)
		return
	}
	post, err := c.postService.FindById(ctx.Request.Context(), uint(postID))
	if err != nil {
		// Let the error middleware handle the internal error
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Post found successfully", post))
}
