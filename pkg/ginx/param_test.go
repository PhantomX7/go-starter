package ginx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/ginx"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// ctxWithParam builds a gin.Context carrying a single path parameter.
func ctxWithParam(name, value string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: name, Value: value}}
	return c
}

func TestParseUintParamValid(t *testing.T) {
	c := ctxWithParam("id", "42")

	id, ok := ginx.ParseUintParam(c, "id")

	require.True(t, ok)
	require.Equal(t, uint(42), id)
	require.Empty(t, c.Errors)
}

func TestParseUintParamInvalid(t *testing.T) {
	c := ctxWithParam("id", "not-a-number")

	id, ok := ginx.ParseUintParam(c, "id")

	require.False(t, ok)
	require.Equal(t, uint(0), id)
	require.Len(t, c.Errors, 1, "a malformed param must record a gin error for the middleware")

	// A malformed path param is a client mistake: the recorded error must be
	// an AppError carrying 400 so the error-handler middleware doesn't fall
	// into its default 500 branch.
	var ae *cerrors.AppError
	require.True(t, errors.As(c.Errors[0].Err, &ae), "recorded error must be an AppError, got %T", c.Errors[0].Err)
	require.Equal(t, http.StatusBadRequest, ae.Code)
}

func TestParseUintParamRejectsNegative(t *testing.T) {
	c := ctxWithParam("id", "-1")

	_, ok := ginx.ParseUintParam(c, "id")

	require.False(t, ok)
}
