package middlewares_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newBodyLimitRouter(maxBytes int64, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := newMiddleware(nil)
	r.Use(mw.ErrorHandler(), mw.BodySizeLimit(maxBytes))
	r.POST("/test", handler)
	return r
}

func TestBodySizeLimitRejectsDeclaredOversizeBody(t *testing.T) {
	r := newBodyLimitRouter(8, func(c *gin.Context) {
		t.Fatal("handler must not run when Content-Length exceeds the cap")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(`{"data":"way past the eight byte cap"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
}

func TestBodySizeLimitAllowsBodyWithinCap(t *testing.T) {
	r := newBodyLimitRouter(1024, func(c *gin.Context) {
		var payload map[string]any
		require.NoError(t, c.ShouldBindJSON(&payload))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(`{"data":"small"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestBodySizeLimitUndeclaredOversizeBodySurfacesAs413(t *testing.T) {
	// With no declared Content-Length (chunked upload), the fast path can't
	// reject early; MaxBytesReader trips inside binding and the error handler
	// must surface it as 413.
	r := newBodyLimitRouter(8, func(c *gin.Context) {
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypeBind)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(`{"data":"way past the eight byte cap"}`))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = -1
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}
