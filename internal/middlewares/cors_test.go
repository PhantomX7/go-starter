package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCORSRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(newMiddleware(nil).CORS())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestCORSSetsHeadersOnRegularRequests(t *testing.T) {
	r := newCORSRouter()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "DELETE")
	// Credentials must never be combined with a wildcard origin — browsers
	// reject the pair, silently breaking every credentialed CORS request.
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSShortCircuitsPreflightRequests(t *testing.T) {
	r := newCORSRouter()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/test", nil))

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Empty(t, rec.Body.String())
}
