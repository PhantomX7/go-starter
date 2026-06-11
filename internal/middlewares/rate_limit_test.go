package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRateLimitRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", newMiddleware(nil).AuthRateLimiter(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func performLogin(r *gin.Engine, remoteAddr string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", nil)
	req.RemoteAddr = remoteAddr
	r.ServeHTTP(rec, req)
	return rec
}

func TestAuthRateLimiterAllowsBurstThenRejects(t *testing.T) {
	r := newRateLimitRouter()

	for i := 0; i < 5; i++ {
		rec := performLogin(r, "10.0.0.1:1234")
		require.Equal(t, http.StatusOK, rec.Code, "request %d within burst should pass", i+1)
	}

	rec := performLogin(r, "10.0.0.1:1234")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "1", rec.Header().Get("Retry-After"))
	require.Contains(t, rec.Body.String(), "too many requests")
}

func TestAuthRateLimiterTracksClientsIndependently(t *testing.T) {
	r := newRateLimitRouter()

	for i := 0; i < 6; i++ {
		performLogin(r, "10.0.0.1:1234")
	}

	rec := performLogin(r, "10.0.0.2:1234")
	require.Equal(t, http.StatusOK, rec.Code, "a different client IP must not share the exhausted bucket")
}
