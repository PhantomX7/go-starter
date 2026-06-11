package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type requestIDCapture struct {
	ginContextID     string
	requestContextID string
}

func newRequestIDRouter(capture *requestIDCapture) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(newMiddleware(nil).RequestID())
	r.GET("/test", func(c *gin.Context) {
		capture.ginContextID = middlewares.GetRequestID(c)
		capture.requestContextID = utils.GetRequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequestIDGeneratesUUIDWhenHeaderMissing(t *testing.T) {
	capture := &requestIDCapture{}
	r := newRequestIDRouter(capture)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

	responseID := rec.Header().Get(middlewares.RequestIDHeader)
	require.NotEmpty(t, responseID)
	_, err := uuid.Parse(responseID)
	require.NoError(t, err, "generated request ID should be a valid UUID")
	require.Equal(t, responseID, capture.ginContextID)
	require.Equal(t, responseID, capture.requestContextID)
}

func TestRequestIDPropagatesIncomingHeader(t *testing.T) {
	capture := &requestIDCapture{}
	r := newRequestIDRouter(capture)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middlewares.RequestIDHeader, "trace-123")
	r.ServeHTTP(rec, req)

	require.Equal(t, "trace-123", rec.Header().Get(middlewares.RequestIDHeader))
	require.Equal(t, "trace-123", capture.ginContextID)
	require.Equal(t, "trace-123", capture.requestContextID)
}

func TestGetRequestIDReturnsEmptyWithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	require.Empty(t, middlewares.GetRequestID(ctx))
}
