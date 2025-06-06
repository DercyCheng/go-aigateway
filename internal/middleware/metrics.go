package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	apiKeyUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_key_usage_total",
			Help: "Total number of API key usages",
		},
		[]string{"key_prefix"},
	)

	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"client_ip"},
	)

	proxyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_requests_total",
			Help: "Total number of proxy requests",
		},
		[]string{"endpoint", "status"},
	)

	proxyRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "proxy_request_duration_seconds",
			Help:    "Proxy request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)
)

// PrometheusMetrics middleware to collect metrics
func PrometheusMetrics() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	})
}

// RecordAPIKeyUsage records API key usage metrics
func RecordAPIKeyUsage(keyPrefix string) {
	apiKeyUsage.WithLabelValues(keyPrefix).Inc()
}

// RecordRateLimitHit records rate limit hits
func RecordRateLimitHit(clientIP string) {
	rateLimitHits.WithLabelValues(clientIP).Inc()
}

// RecordProxyRequest records proxy request metrics
func RecordProxyRequest(endpoint string, status int, duration time.Duration) {
	statusStr := strconv.Itoa(status)
	proxyRequestsTotal.WithLabelValues(endpoint, statusStr).Inc()
	proxyRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}
