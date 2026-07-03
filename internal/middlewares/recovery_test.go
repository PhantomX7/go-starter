package middlewares_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// A panic in a handler must surface as the standard JSON envelope with a 500,
// not gin.Recovery()'s bare body-less 500 — otherwise clients decoding the
// {status,message} envelope choke exactly on the worst-case path.
func TestRecoveryReturnsJSONEnvelopeOnPanic(t *testing.T) {
	setupLogger(t)

	mw := newMiddleware(newCasbinClient(nil))

	engine := gin.New()
	engine.Use(mw.Recovery())
	engine.GET("/boom", func(_ *gin.Context) {
		panic("kaboom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

	var body struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.False(t, body.Status)
	require.NotEmpty(t, body.Message)
}
