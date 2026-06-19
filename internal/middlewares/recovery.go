package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
)

// Recovery recovers from panics in downstream handlers and replies with the
// standard JSON envelope (status:false) and a 500 — the same shape every other
// error path returns — instead of gin.Recovery()'s bare, body-less 500 that
// breaks clients decoding the envelope. The panic value and stack are logged at
// error level, correlated by request_id.
//
// It must sit INSIDE the Logger middleware (so the recovered 500 is still
// access-logged) and OUTSIDE the ErrorHandler: a panic unwinds straight past
// ErrorHandler's frame, so only a recovery deferred here can format it.
func (m *Middleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		logger.Error("Recovered from panic",
			zap.String("request_id", GetRequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Any("panic", err),
			zap.Stack("stack"),
		)
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			response.BuildResponseFailed("An unexpected error occurred."),
		)
	})
}
