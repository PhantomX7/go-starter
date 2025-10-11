package middlewares

import (
	"log"
	"net/http"

	"github.com/PhantomX7/go-starter/pkg/errors"
	"github.com/PhantomX7/go-starter/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorHandler is a middleware to handle errors encountered during requests
// It intercepts validation errors from Gin's binding operations and provides
// structured error responses with appropriate HTTP status codes
func (m *Middleware) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// IMPORTANT: This runs the next handler in the chain.
		// Any errors that occur will be collected by Gin.
		c.Next()

		// After the handler has run, check if there are any errors.
		if len(c.Errors) > 0 {
			// Get the last error in the list.
			err := c.Errors.Last().Err

			// Use a switch to handle different types of errors.
			switch e := err.(type) {
			case validator.ValidationErrors:
				// If the error is a validation error, format it.
				c.AbortWithStatusJSON(http.StatusBadRequest, response.BuildResponseValidationError(
					e,
				))
			case *errors.AppError:
				// If the error is a custom AppError, return it.
				if e.Code == http.StatusInternalServerError {
					log.Printf("Internal server error: %v", e)
				}
				c.AbortWithStatusJSON(e.Code, response.BuildResponseFailed(e.Message))
			default:
				// For any other error, return a generic 500.
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_SERVER_ERROR",
					"message": "An unexpected error occurred.",
				})
			}
			return
		}
	}
}
