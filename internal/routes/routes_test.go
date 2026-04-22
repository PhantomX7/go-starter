package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type alphaRegistrar struct{}

func (alphaRegistrar) RegisterRoutes(ctx *Context) {
	ctx.Root.GET("/alpha", func(c *gin.Context) {
		c.String(http.StatusOK, "alpha")
	})
}

type betaRegistrar struct{}

func (betaRegistrar) RegisterRoutes(ctx *Context) {
	ctx.Public.GET("/beta", func(c *gin.Context) {
		c.String(http.StatusOK, "beta")
	})
}

func TestRegisterRoutesMountsGroupedRegistrars(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	RegisterRoutes(engine, nil, registerParams{
		Registrars: []Registrar{
			alphaRegistrar{},
			betaRegistrar{},
		},
	})

	alphaReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/alpha", nil)
	alphaRec := httptest.NewRecorder()
	engine.ServeHTTP(alphaRec, alphaReq)
	require.Equal(t, http.StatusOK, alphaRec.Code)
	require.Equal(t, "alpha", alphaRec.Body.String())

	betaReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/public/beta", nil)
	betaRec := httptest.NewRecorder()
	engine.ServeHTTP(betaRec, betaReq)
	require.Equal(t, http.StatusOK, betaRec.Code)
	require.Equal(t, "beta", betaRec.Body.String())
}
