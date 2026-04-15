package middlewares

import (
	"github.com/PhantomX7/athleton/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID in Gin
	RequestIDKey = "request_id"
)

// RequestID middleware generates or retrieves a request ID for each request
func (m *Middleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID exists in header (for request tracing across services)
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in Gin context (for easy access in handlers)
		c.Set(RequestIDKey, requestID)

		// Set request ID in request context (for use in services/repositories)
		ctx := utils.SetRequestIDToContext(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set request ID in response header (for client-side tracing)
		c.Writer.Header().Set(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the Gin context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
