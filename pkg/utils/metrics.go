package utils

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RequestCount is a Prometheus counter vector that tracks the total number of HTTP requests.
	// It is labeled by 'path' (URL path) and 'method' (HTTP method).
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",      // Name of the metric as it will be exposed.
			Help: "Number of HTTP requests.", // Help text for the metric.
		},
		[]string{"path", "method"}, // Labels for the metric.
	)
)

// InitMetrics registers the Prometheus metrics with the default registry.
// This function should be called once during application startup.
func InitMetrics() {
	// MustRegister will panic if the metric registration fails (e.g., duplicate metric name).
	prometheus.MustRegister(RequestCount)
}
