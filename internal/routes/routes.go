package routes

import (
	"github.com/PhantomX7/go-starter/internal/middlewares"
	post "github.com/PhantomX7/go-starter/internal/modules/post/controller"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	route *gin.Engine,
	middlewares *middlewares.Middleware,
	PostController post.PostController,
) {
	api := route.Group("/api")
	{
		postRoute := api.Group("/post")
		{
			postRoute.POST("", PostController.Create)
			postRoute.PUT("/:id", PostController.Update)
			postRoute.DELETE("/:id", PostController.Delete)
			postRoute.GET("/:id", PostController.FindById)
		}
	}
}
