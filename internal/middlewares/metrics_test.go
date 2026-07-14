package middlewares_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/middlewares"
)

// findMetric returns the named metric family from a registry gather, failing
// the test when absent.
func findMetric(t *testing.T, families []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()

	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	t.Fatalf("metric %s not found", name)
	return nil
}

func newMetricsTestServer(t *testing.T) (*gin.Engine, *prometheus.Registry) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	reg := prometheus.NewRegistry()
	metrics := middlewares.NewHTTPMetrics(reg)

	server := gin.New()
	server.Use(metrics.Handler())
	server.GET("/things/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	server.GET("/boom", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})
	return server, reg
}

func serveMetricsReq(server *gin.Engine, method, path string) {
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), method, path, nil))
}

func TestHTTPMetricsCountsRequestsByRouteTemplateAndStatus(t *testing.T) {
	server, reg := newMetricsTestServer(t)

	// Two different IDs must land on ONE route label (the template), or a
	// scanner walking random URLs would explode metric cardinality.
	serveMetricsReq(server, http.MethodGet, "/things/1")
	serveMetricsReq(server, http.MethodGet, "/things/2")
	serveMetricsReq(server, http.MethodGet, "/boom")

	requests, err := reg.Gather()
	require.NoError(t, err)

	counter := findMetric(t, requests, "http_requests_total")
	byLabels := map[string]float64{}
	for _, m := range counter.GetMetric() {
		key := ""
		for _, l := range m.GetLabel() {
			key += l.GetName() + "=" + l.GetValue() + ";"
		}
		byLabels[key] = m.GetCounter().GetValue()
	}
	require.Equal(t, float64(2), byLabels["method=GET;route=/things/:id;status=200;"])
	require.Equal(t, float64(1), byLabels["method=GET;route=/boom;status=500;"])
}

func TestHTTPMetricsObservesRequestDuration(t *testing.T) {
	server, reg := newMetricsTestServer(t)

	serveMetricsReq(server, http.MethodGet, "/things/1")

	families, err := reg.Gather()
	require.NoError(t, err)
	histogram := findMetric(t, families, "http_request_duration_seconds")
	require.Len(t, histogram.GetMetric(), 1)
	require.Equal(t, uint64(1), histogram.GetMetric()[0].GetHistogram().GetSampleCount())
}

func TestHTTPMetricsCollapsesUnmatchedRoutes(t *testing.T) {
	server, reg := newMetricsTestServer(t)

	// Unmatched paths must share a single label value, not mint one series
	// per probed URL.
	serveMetricsReq(server, http.MethodGet, "/nope/a")
	serveMetricsReq(server, http.MethodGet, "/nope/b")

	families, err := reg.Gather()
	require.NoError(t, err)
	counter := findMetric(t, families, "http_requests_total")
	require.Len(t, counter.GetMetric(), 1)
	for _, l := range counter.GetMetric()[0].GetLabel() {
		if l.GetName() == "route" {
			require.Equal(t, "unmatched", l.GetValue())
		}
	}
}
