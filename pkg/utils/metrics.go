package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto" // For easier registration
)

var (
	// HTTPRequestsTotal replaced by Chi middleware or a more standard one if available
	// Default Chi middleware for Prometheus usually covers this.

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"path", "method", "code"}, // Added "code" for status code
	)

	CacheActionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_actions_total",
			Help: "Total number of cache actions (hits or misses).",
		},
		[]string{"type"}, // "hit" or "miss"
	)

	DBOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_operation_duration_seconds",
			Help:    "Duration of database operations.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"}, // e.g., "GetTargetedCampaigns", "GetCampaignByID"
	)
)

// InitMetrics is called on application startup to register metrics.
// With promauto, explicit registration is handled, but this function can be kept for consistency
// or if any non-promauto metrics were to be added.
func InitMetrics() {
	// promauto.Register(...) handles registration automatically.
	// If you had metrics not using promauto, you'd register them here:
	// prometheus.MustRegister(HTTPRequestDuration)
	// prometheus.MustRegister(CacheActionsTotal)
	// prometheus.MustRegister(DBOperationDuration)
	// No action needed here if all metrics use promauto.
}

// RecordCacheHit increments the cache hit counter.
func RecordCacheHit() {
	CacheActionsTotal.With(prometheus.Labels{"type": "hit"}).Inc()
}

// RecordCacheMiss increments the cache miss counter.
func RecordCacheMiss() {
	CacheActionsTotal.With(prometheus.Labels{"type": "miss"}).Inc()
}

// Example function to observe DB operation duration - to be called from DB interaction points
// Usage:
// timer := prometheus.NewTimer(DBOperationDuration.WithLabelValues("GetTargetedCampaigns"))
// defer timer.ObserveDuration()
// ... your DB call ...
