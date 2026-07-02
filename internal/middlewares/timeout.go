package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware enforces a per-request deadline via the request context
// without spawning a goroutine for c.Next().
func (m *Middleware) TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Let downstream handlers check ctx.Err() / ctx.Done() themselves.
		// Gin handlers run synchronously — don't wrap c.Next() in a goroutine.
		c.Next()

		// If the context deadline was exceeded by a slow DB call / external service,
		// downstream code should have already noticed. We catch it here as a fallback.
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			// Use the standard failure envelope so timeout responses decode
			// like every other error path (Recovery, ErrorHandler, NoRoute).
			c.AbortWithStatusJSON(http.StatusRequestTimeout,
				response.BuildResponseFailed("request timeout: exceeded "+timeout.String()))
		}
	}
}
