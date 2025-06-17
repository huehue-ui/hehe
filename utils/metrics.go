package utils

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests.",
		},
		[]string{"path", "method"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(RequestCount)
}
