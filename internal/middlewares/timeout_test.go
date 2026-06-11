package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTimeoutRouter(timeout time.Duration, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(newMiddleware(nil).TimeoutMiddleware(timeout))
	r.GET("/test", handler)
	return r
}

func TestTimeoutMiddlewarePassesFastRequestsThrough(t *testing.T) {
	r := newTimeoutRouter(time.Second, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"ok":true}`, rec.Body.String())
}

func TestTimeoutMiddlewareSetsRequestDeadline(t *testing.T) {
	deadlineCh := make(chan bool, 1)
	r := newTimeoutRouter(time.Second, func(c *gin.Context) {
		_, ok := c.Request.Context().Deadline()
		deadlineCh <- ok
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.True(t, <-deadlineCh, "handler should observe a context deadline")
}

func TestTimeoutMiddlewareRespondsWithTimeoutWhenHandlerWroteNothing(t *testing.T) {
	r := newTimeoutRouter(10*time.Millisecond, func(c *gin.Context) {
		<-c.Request.Context().Done()
		// Handler notices the deadline and bails out without writing.
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusRequestTimeout, rec.Code)
	require.Contains(t, rec.Body.String(), "Request Timeout")
}

func TestTimeoutMiddlewareKeepsResponseWrittenBeforeDeadline(t *testing.T) {
	r := newTimeoutRouter(10*time.Millisecond, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		<-c.Request.Context().Done()
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"ok":true}`, rec.Body.String())
}

func TestTimeoutMiddlewareCancelsContextAfterDeadline(t *testing.T) {
	errCh := make(chan error, 1)
	r := newTimeoutRouter(10*time.Millisecond, func(c *gin.Context) {
		<-c.Request.Context().Done()
		errCh <- c.Request.Context().Err()
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.ErrorIs(t, <-errCh, context.DeadlineExceeded)
}
