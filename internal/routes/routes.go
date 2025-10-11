package routes

import (
	"github.com/PhantomX7/go-starter/internal/middlewares"
	auth "github.com/PhantomX7/go-starter/internal/modules/auth/controller"
	post "github.com/PhantomX7/go-starter/internal/modules/post/controller"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	route *gin.Engine,
	middlewares *middlewares.Middleware,
	PostController post.PostController,
	AuthController auth.AuthController,
) {
	api := route.Group("/api")
	{
		authRoute := api.Group("/auth")
		{
			authRoute.GET("/:provider", AuthController.LoginOauth)
			authRoute.GET("/:provider/callback", AuthController.CallbackOauth)
		}
		postRoute := api.Group("/post")
		{
			postRoute.GET("", PostController.Index)
			postRoute.POST("", PostController.Create)
			postRoute.PUT("/:id", PostController.Update)
			postRoute.DELETE("/:id", PostController.Delete)
			postRoute.GET("/:id", PostController.FindById)
		}
	}
}
