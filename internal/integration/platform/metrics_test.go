package platform_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"
)

// TestMetricsEndpoint verifies the Prometheus scrape surface registered by
// bootstrap.SetupServer: traffic through the real middleware chain must show
// up in the exposition, labeled by route template.
func TestMetricsEndpoint(t *testing.T) {
	app := harness.New(t)

	// Generate one known request before scraping.
	rec := app.Request(t, http.MethodGet, "/livez", nil, "")
	require.Equal(t, http.StatusOK, rec.Code)

	scrape := app.Request(t, http.MethodGet, "/metrics", nil, "")
	require.Equal(t, http.StatusOK, scrape.Code)
	body := scrape.Body.String()
	require.Contains(t, body, "http_requests_total")
	require.Contains(t, body, "http_request_duration_seconds")
	require.Contains(t, body, `route="/livez"`)
}
