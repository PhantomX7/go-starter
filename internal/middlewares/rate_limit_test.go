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

func performRequest(r *gin.Engine, path, remoteAddr string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, nil)
	req.RemoteAddr = remoteAddr
	r.ServeHTTP(rec, req)
	return rec
}

func performLogin(r *gin.Engine, remoteAddr string) *httptest.ResponseRecorder {
	return performRequest(r, "/login", remoteAddr)
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

// Every constructor call must yield its own limiter instance: routes mounting
// AuthRateLimiter() per endpoint get independent buckets, so exhausting one
// endpoint's budget (e.g. /register) must not lock the same IP out of another
// (e.g. /login).
func TestAuthRateLimiterInstancesDoNotShareBuckets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := newMiddleware(nil)
	r := gin.New()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.POST("/register", mw.AuthRateLimiter(), ok)
	r.POST("/login", mw.AuthRateLimiter(), ok)

	for i := 0; i < 6; i++ {
		performRequest(r, "/register", "10.0.0.1:1234")
	}
	require.Equal(t, http.StatusTooManyRequests, performRequest(r, "/register", "10.0.0.1:1234").Code)

	rec := performRequest(r, "/login", "10.0.0.1:1234")
	require.Equal(t, http.StatusOK, rec.Code, "exhausting /register must not consume /login's bucket")
}

// The refresh limiter is deliberately looser than the login/register limiter:
// legitimate clients refresh routinely, so it must absorb a burst of 20 before
// rejecting.
func TestRefreshRateLimiterAllowsLargerBurstThenRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/refresh", newMiddleware(nil).RefreshRateLimiter(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 20; i++ {
		rec := performRequest(r, "/refresh", "10.0.0.1:1234")
		require.Equal(t, http.StatusOK, rec.Code, "request %d within burst should pass", i+1)
	}

	rec := performRequest(r, "/refresh", "10.0.0.1:1234")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "1", rec.Header().Get("Retry-After"))
	require.Contains(t, rec.Body.String(), "too many requests")
}
