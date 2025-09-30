package routes

import (
	post "github.com/PhantomX7/go-starter/internal/modules/post/controller"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	route *gin.Engine,

	PostController post.PostController,
) {
	api := route.Group("/api")
	{
		postRoute := api.Group("/posts")
		{
			postRoute.POST("", PostController.Create)
			postRoute.PUT("/:id", PostController.Update)
			postRoute.DELETE("/:id", PostController.Delete)
			postRoute.GET("/:id", PostController.FindById)
		}
	}
}
