package middlewares

import (
	"net/http"

	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// BodySizeLimit caps the request body at maxBytes. Reads past the cap fail
// inside binding with *http.MaxBytesError, which we surface as 413 instead of
// letting an oversized payload exhaust memory.
func (m *Middleware) BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Fast path: reject early when the declared length already exceeds
		// the cap, before reading anything.
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge,
				response.BuildResponseFailed("request body too large"))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
