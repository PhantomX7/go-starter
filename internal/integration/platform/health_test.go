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

	// /healthz differs from the cheap probes: it reports each dependency
	// check by name (the database ping runs against the real in-memory DB).
	for path, want := range map[string]string{
		"/livez":   `{"status":"ok"}`,
		"/healthz": `{"status":"ok","checks":{"database":"ok"}}`,
		"/readyz":  `{"status":"ok"}`,
	} {
		t.Run(path, func(t *testing.T) {
			rec := app.Request(t, http.MethodGet, path, nil, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.JSONEq(t, want, rec.Body.String())
		})
	}
}
