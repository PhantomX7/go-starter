package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware creates a timeout middleware with the specified duration
func (m *Middleware) TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		panicChan := make(chan any, 1)

		go func() {
			defer func() {
				if err := recover(); err != nil {
					// Capture panic with full stack trace
					panicChan <- err
				}
			}()

			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			// Completed successfully
			return

		case p := <-panicChan:
			// Re-panic in the main goroutine to preserve stack trace
			panic(p)

		case <-ctx.Done():
			// Timeout occurred
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request Timeout",
				"message": "Request took longer than " + timeout.String(),
			})
			c.Abort()
		}
	}
}
