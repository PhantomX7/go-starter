package middlewares

import (
	"errors"
	"net/http"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorHandler is a middleware to handle errors encountered during requests
// It intercepts validation errors from Gin's binding operations and provides
// structured error responses with appropriate HTTP status codes
func (m *Middleware) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var ve validator.ValidationErrors
		var ae *cerrors.AppError

		switch {
		case errors.As(err, &ve):
			c.AbortWithStatusJSON(http.StatusBadRequest, response.BuildResponseValidationError(ve))
		case errors.As(err, &ae):
			if errors.Is(ae, cerrors.ErrNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, response.BuildResponseFailed(ae.Err.Error()))
				return
			}
			c.AbortWithStatusJSON(ae.Code, response.BuildResponseFailed(ae.Message))
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.BuildResponseFailed("An unexpected error occurred."))
		}
	}
}
