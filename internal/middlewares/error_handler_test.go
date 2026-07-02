package middlewares_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
)

func newErrorHandlerRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(newMiddleware(nil).ErrorHandler())
	r.POST("/test", handler)
	return r
}

func TestErrorHandlerLeavesSuccessfulRequestsUntouched(t *testing.T) {
	r := newErrorHandlerRouter(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"ok":true}`, rec.Body.String())
}

func TestErrorHandlerFormatsValidationErrors(t *testing.T) {
	r := newErrorHandlerRouter(func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypeBind)
		}
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", strings.NewReader(`{"description":"no name"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
	require.Equal(t, "Validation failed", body["message"])

	errPayload, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), errPayload["total_errors"])
	fields, ok := errPayload["fields"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "name is required", fields["name"])
}

func TestErrorHandlerMapsNotFoundAppErrors(t *testing.T) {
	r := newErrorHandlerRouter(func(c *gin.Context) {
		_ = c.Error(cerrors.NewNotFoundError("post not found")).SetType(gin.ErrorTypePublic)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil))

	require.Equal(t, http.StatusNotFound, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
	require.Equal(t, "post not found", body["message"])
}

func TestErrorHandlerUsesAppErrorStatusCode(t *testing.T) {
	r := newErrorHandlerRouter(func(c *gin.Context) {
		_ = c.Error(cerrors.NewUnauthorizedError("invalid credentials")).SetType(gin.ErrorTypePublic)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "invalid credentials", body["message"])
}

func TestErrorHandlerMapsMaxBytesErrorTo413(t *testing.T) {
	// A body that overflows http.MaxBytesReader surfaces from ShouldBind as
	// *http.MaxBytesError; the contract promised by BodySizeLimit is a 413,
	// not the generic 400/500.
	r := newErrorHandlerRouter(func(c *gin.Context) {
		_ = c.Error(&http.MaxBytesError{Limit: 1024}).SetType(gin.ErrorTypeBind)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil))

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["status"])
}

func TestErrorHandlerHidesUnexpectedErrorDetails(t *testing.T) {
	r := newErrorHandlerRouter(func(c *gin.Context) {
		_ = c.Error(errors.New("pq: connection refused")).SetType(gin.ErrorTypePrivate)
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "connection refused")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "An unexpected error occurred.", body["message"])
}
