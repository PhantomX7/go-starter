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
	postController post.PostController,
	authController auth.AuthController,
) {
	api := route.Group("/api")
	{
		authRoute := api.Group("/auth")
		{
			authRoute.POST("/register", authController.Register)
			authRoute.POST("/login", authController.Login)
			authRoute.POST("/refresh", authController.Refresh)
			authRoute.GET("/me", middlewares.AuthHandle(), authController.GetMe)
			// authRoute.GET("/:provider", authHandler.OAuthLogin)
			// authRoute.GET("/:provider/callback", authHandler.OAuthCallback)
		}
		postRoute := api.Group("/post")
		{
			postRoute.GET("", postController.Index)
			postRoute.POST("", postController.Create)
			postRoute.PUT("/:id", postController.Update)
			postRoute.DELETE("/:id", postController.Delete)
			postRoute.GET("/:id", postController.FindById)
		}
	}
}
