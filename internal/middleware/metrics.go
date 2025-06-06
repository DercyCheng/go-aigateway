package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 基础HTTP指标
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
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// QPS指标
	requestsPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_per_second",
			Help: "Current requests per second",
		},
		[]string{"endpoint"},
	)

	// API密钥使用指标
	apiKeyUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_key_usage_total",
			Help: "Total number of API key usages",
		},
		[]string{"key_prefix"},
	)

	// 限流指标
	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"client_ip"},
	)

	// 代理请求指标
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
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"endpoint"},
	)

	// 后端服务成功率指标
	backendSuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_success_rate",
			Help: "Backend service success rate (percentage)",
		},
		[]string{"endpoint"},
	)

	// 高级监控指标
	concurrentRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "concurrent_requests",
			Help: "Current number of concurrent requests",
		},
	)

	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "endpoint"},
	)

	requestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// 错误指标
	errorRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_error_rate",
			Help: "HTTP error rate (percentage)",
		},
		[]string{"endpoint"},
	)

	// 熔断器指标
	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"client_id"},
	)

	circuitBreakerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total circuit breaker requests",
		},
		[]string{"client_id", "state"},
	)

	// 连接池指标
	activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)

	// 内存和CPU使用率指标
	memoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)

	cpuUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "Current CPU usage percentage",
		},
	)

	// 缓存指标
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_type"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_type"},
	)
)

// PrometheusMetrics middleware to collect metrics
func PrometheusMetrics() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		
		// 增加并发请求计数
		concurrentRequests.Inc()
		defer concurrentRequests.Dec()

		// 记录请求大小
		if c.Request.ContentLength > 0 {
			requestSize.WithLabelValues(c.Request.Method, c.FullPath()).Observe(float64(c.Request.ContentLength))
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		method := c.Request.Method

		// 基础指标
		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)

		// 响应大小
		responseSize.WithLabelValues(method, endpoint).Observe(float64(c.Writer.Size()))

		// 更新QPS（简化版本，实际应该使用时间窗口计算）
		requestsPerSecond.WithLabelValues(endpoint).Set(1.0 / duration)
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
	
	// 计算成功率
	successRate := calculateSuccessRate(endpoint, status)
	backendSuccessRate.WithLabelValues(endpoint).Set(successRate)
}

// RecordAdvancedMetrics 记录高级指标
func RecordAdvancedMetrics(metricType, endpoint, status string, duration time.Duration) {
	switch metricType {
	case "circuit_breaker":
		circuitBreakerRequests.WithLabelValues(endpoint, status).Inc()
	case "rate_limit":
		// 已在其他地方处理
	}
}

// UpdateBackendSuccessRate 更新后端服务成功率
func UpdateBackendSuccessRate(endpoint string, successRate float64) {
	backendSuccessRate.WithLabelValues(endpoint).Set(successRate)
}

// UpdateErrorRate 更新错误率
func UpdateErrorRate(endpoint string, errorRate float64) {
	errorRate.WithLabelValues(endpoint).Set(errorRate)
}

// RecordCacheHit 记录缓存命中
func RecordCacheHit(cacheType string) {
	cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss 记录缓存未命中
func RecordCacheMiss(cacheType string) {
	cacheMisses.WithLabelValues(cacheType).Inc()
}

// UpdateCircuitBreakerState 更新熔断器状态
func UpdateCircuitBreakerState(clientID string, state int) {
	circuitBreakerState.WithLabelValues(clientID).Set(float64(state))
}

// UpdateSystemMetrics 更新系统指标
func UpdateSystemMetrics(memBytes uint64, cpuPercent float64) {
	memoryUsage.Set(float64(memBytes))
	cpuUsage.Set(cpuPercent)
}

// UpdateActiveConnections 更新活跃连接数
func UpdateActiveConnections(count int) {
	activeConnections.Set(float64(count))
}

// calculateSuccessRate 计算成功率（简化版本）
func calculateSuccessRate(endpoint string, status int) float64 {
	// 这里应该实现一个滑动窗口的成功率计算
	// 现在简化为基于单次请求的状态
	if status >= 200 && status < 400 {
		return 100.0
	}
	return 0.0
}

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
