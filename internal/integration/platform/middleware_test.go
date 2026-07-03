package platform_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"
)

// TestUnknownRouteReturnsJSONEnvelope verifies the NoRoute handler returns the
// uniform failure envelope instead of Gin's plain-text 404.
func TestUnknownRouteReturnsJSONEnvelope(t *testing.T) {
	app := harness.New(t)

	rec := app.Request(t, http.MethodGet, "/api/v1/does-not-exist", nil, "")
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "route not found", env.Message)
}

// TestOversizedBodyRejectedWith413 sends a payload one byte over the
// configured cap and expects the body-limit middleware to reject it before
// any handler work happens.
func TestOversizedBodyRejectedWith413(t *testing.T) {
	app := harness.New(t)

	oversized := bytes.Repeat([]byte("a"), int(harness.TestMaxBodyBytes)+1)
	rec := app.Request(t, http.MethodPost, "/api/v1/auth/login", oversized, "")
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "request body too large", env.Message)
}

// TestValidationErrorReturnsStructuredEnvelope verifies that a binding
// validation failure surfaces as a 400 with the structured validation shape
// produced by the centralized error handler.
func TestValidationErrorReturnsStructuredEnvelope(t *testing.T) {
	app := harness.New(t)
	tokens := app.LoginAs(t, harness.RootUsername, harness.TestPassword)

	rec := app.Request(t, http.MethodPost, "/api/v1/admin/admin-role", map[string]string{}, tokens.AccessToken)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "Validation failed", env.Message)

	var validation struct {
		TotalErrors int               `json:"total_errors"`
		Fields      map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(env.Error, &validation))
	require.Equal(t, 2, validation.TotalErrors)
	require.Contains(t, validation.Fields, "name")
	require.Contains(t, validation.Fields, "permissions")
}

// TestMalformedJSONBodyReturns400 verifies that a syntactically invalid JSON
// body is reported as a client error (400), not a server fault: the error
// handler maps non-validator bind errors to a 400 envelope.
func TestMalformedJSONBodyReturns400(t *testing.T) {
	app := harness.New(t)
	tokens := app.LoginAs(t, harness.RootUsername, harness.TestPassword)

	rec := app.Request(t, http.MethodPost, "/api/v1/admin/admin-role", `{"name": "broken"`, tokens.AccessToken)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "Invalid request body.", env.Message)
}

// TestMethodNotAllowedReturnsJSONEnvelope verifies HandleMethodNotAllowed is
// enabled and the NoMethod handler returns the uniform envelope.
func TestMethodNotAllowedReturnsJSONEnvelope(t *testing.T) {
	app := harness.New(t)

	// /livez only registers GET; DELETE must hit the NoMethod handler.
	rec := app.Request(t, http.MethodDelete, "/livez", nil, "")
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "method not allowed", env.Message)
}

// TestRequestIDHeaderPropagation verifies the request-ID middleware both
// generates an ID and echoes a caller-provided one.
func TestRequestIDHeaderPropagation(t *testing.T) {
	app := harness.New(t)

	t.Run("generated when absent", func(t *testing.T) {
		rec := app.Request(t, http.MethodGet, "/livez", nil, "")
		require.Equal(t, http.StatusOK, rec.Code)
		require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
	})

	t.Run("echoed when provided", func(t *testing.T) {
		req, rec := harness.NewRequestWithHeader(t, http.MethodGet, "/livez", "X-Request-ID", "integration-rid-123")
		app.Engine.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "integration-rid-123", rec.Header().Get("X-Request-ID"))
	})
}
