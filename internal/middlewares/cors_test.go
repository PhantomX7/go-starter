package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/pkg/config"
)

// newCORSRouter builds a router whose CORS middleware allows the given
// origins; an empty list leaves the middleware in wildcard mode.
func newCORSRouter(allowedOrigins ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.Server.CORSAllowedOrigins = allowedOrigins
	r := gin.New()
	r.Use(newMiddlewareWithConfig(cfg, nil).CORS())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func performCORSRequest(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), method, "/test", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	r.ServeHTTP(rec, req)
	return rec
}

func TestCORSSetsHeadersOnRegularRequests(t *testing.T) {
	r := newCORSRouter()

	rec := performCORSRequest(r, http.MethodGet, "")

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

	rec := performCORSRequest(r, http.MethodOptions, "")

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	require.Empty(t, rec.Body.String())
}

func TestCORSEchoesAllowlistedOrigin(t *testing.T) {
	r := newCORSRouter("https://app.example.com", "https://admin.example.com")

	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		rec := performCORSRequest(r, method, "https://admin.example.com")

		require.Equal(t, "https://admin.example.com", rec.Header().Get("Access-Control-Allow-Origin"),
			"%s must echo an allowlisted origin, never the wildcard", method)
		require.Contains(t, rec.Header().Values("Vary"), "Origin")
	}
}

func TestCORSOmitsAllowOriginForDisallowedOrigin(t *testing.T) {
	r := newCORSRouter("https://app.example.com")

	cases := map[string]string{
		"unlisted origin":  "https://evil.example.com",
		"no origin header": "",
	}

	for name, origin := range cases {
		t.Run(name, func(t *testing.T) {
			rec := performCORSRequest(r, http.MethodGet, origin)

			require.Equal(t, http.StatusOK, rec.Code, "the request itself still succeeds; the browser enforces the missing header")
			require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
			// Vary: Origin must be present even on the deny path so a cache never
			// replays a headerless response to an allowlisted origin (or vice versa).
			require.Contains(t, rec.Header().Values("Vary"), "Origin")
		})
	}
}
