package middlewares

import (
	"time"

	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger is a middleware that logs HTTP requests and responses
func (m *Middleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get request ID from context
		requestID := GetRequestID(c)

		// Process request
		c.Next()

		// Calculate request duration
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Prepare base log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		}

		// Add user info if authenticated
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		if role, exists := c.Get("role"); exists {
			fields = append(fields, zap.Any("role", role))
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			lastError := c.Errors.Last()
			fields = append(fields, zap.String("error", lastError.Error()))
		}

		// Log based on status code and latency
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		case latency > 3*time.Second:
			// Log slow successful requests as warnings
			logger.Warn("Slow request", fields...)
		case statusCode >= 300:
			logger.Info("Redirection", fields...)
		default:
			// Use Debug for normal successful requests to reduce noise
			logger.Debug("Request completed", fields...)
		}
	}
}
