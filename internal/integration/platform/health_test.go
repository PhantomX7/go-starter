package platform_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"
)

// TestHealthEndpoints verifies the liveness and readiness probes registered by
// bootstrap.SetupServer, including the /readyz database ping against the real
// (in-memory) database connection.
func TestHealthEndpoints(t *testing.T) {
	app := harness.New(t)

	for _, path := range []string{"/livez", "/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			rec := app.Request(t, http.MethodGet, path, nil, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
		})
	}
}
