package middlewares

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics instruments HTTP traffic for Prometheus: a request counter by
// method/route/status and a latency histogram by method/route. Routes are
// labeled by their template (c.FullPath(), e.g. "/admin/admin-role/:id"), not
// the raw URL, so metric cardinality stays bounded by the route table.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewHTTPMetrics creates the HTTP instruments and registers them on reg. It
// takes the registerer explicitly (rather than the process-global default) so
// each server gets its own registry — tests and the integration harness build
// many servers per process, and double-registering on the global panics.
func NewHTTPMetrics(reg prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, by method, route template, and status code.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, by method and route template.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
	reg.MustRegister(m.requests, m.duration)
	return m
}

// Handler returns the Gin middleware that records both instruments.
func (m *HTTPMetrics) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			// Unmatched paths (404s from scanners probing random URLs) share
			// one label value instead of minting a series per probed URL.
			route = "unmatched"
		}
		method := c.Request.Method
		m.requests.WithLabelValues(method, route, strconv.Itoa(c.Writer.Status())).Inc()
		m.duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	}
}
