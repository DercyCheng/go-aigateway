package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
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
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
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
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"endpoint"},
	)

	// 新增的高级监控指标
	backendSuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backend_success_rate",
			Help: "Backend service success rate percentage",
		},
		[]string{"backend", "endpoint"},
	)

	requestQPS = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "request_qps",
			Help: "Current requests per second",
		},
		[]string{"endpoint"},
	)

	concurrentConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "concurrent_connections",
			Help: "Current number of concurrent connections",
		},
	)

	errorRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "error_rate",
			Help: "Error rate percentage",
		},
		[]string{"endpoint"},
	)

	responseTimePercentile = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "response_time_percentile_seconds",
			Help:    "Response time percentiles",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"endpoint", "percentile"},
	)

	activeUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of active users in the last minute",
		},
	)

	bytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bytes_transferred_total",
			Help: "Total bytes transferred",
		},
		[]string{"direction"}, // "in" or "out"
	)
)

// AdvancedMetricsCollector 高级指标收集器
type AdvancedMetricsCollector struct {
	redisClient *redis.Client
}

// NewAdvancedMetricsCollector 创建高级指标收集器
func NewAdvancedMetricsCollector(redisClient *redis.Client) *AdvancedMetricsCollector {
	return &AdvancedMetricsCollector{
		redisClient: redisClient,
	}
}

// PrometheusMetrics middleware to collect metrics
func PrometheusMetrics() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// 增加并发连接数
		concurrentConnections.Inc()
		defer concurrentConnections.Dec()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		endpoint := c.FullPath()
		method := c.Request.Method

		// 记录基础指标
		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)

		// 记录字节传输量
		bytesTransferred.WithLabelValues("in").Add(float64(c.Request.ContentLength))
		bytesTransferred.WithLabelValues("out").Add(float64(c.Writer.Size()))

		// 记录响应时间百分位数
		responseTimePercentile.WithLabelValues(endpoint, "p50").Observe(duration)
		responseTimePercentile.WithLabelValues(endpoint, "p95").Observe(duration)
		responseTimePercentile.WithLabelValues(endpoint, "p99").Observe(duration)
	})
}

// AdvancedPrometheusMetrics 高级Prometheus指标中间件
func AdvancedPrometheusMetrics(collector *AdvancedMetricsCollector) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		endpoint := c.FullPath()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// 更新Redis中的实时指标
		ctx := context.Background()
		go collector.updateRealTimeMetrics(ctx, endpoint, status, duration, c)
	})
}

// updateRealTimeMetrics 更新实时指标到Redis
func (amc *AdvancedMetricsCollector) updateRealTimeMetrics(ctx context.Context, endpoint string, status int, duration time.Duration, c *gin.Context) {
	// 更新QPS统计
	qpsKey := "metrics:qps:current"
	amc.redisClient.Incr(ctx, qpsKey)
	amc.redisClient.Expire(ctx, qpsKey, time.Second)

	// 更新平均响应时间（使用滑动窗口）
	responseTimeKey := "metrics:response_time:samples"
	amc.redisClient.RPush(ctx, responseTimeKey, duration.Seconds())
	amc.redisClient.LTrim(ctx, responseTimeKey, -100, -1) // 保持最近100个样本
	amc.redisClient.Expire(ctx, responseTimeKey, time.Minute*5)

	// 计算平均响应时间
	samples, _ := amc.redisClient.LRange(ctx, responseTimeKey, 0, -1).Result()
	if len(samples) > 0 {
		var total float64
		for _, sample := range samples {
			if val, err := strconv.ParseFloat(sample, 64); err == nil {
				total += val
			}
		}
		avg := total / float64(len(samples))
		amc.redisClient.Set(ctx, "metrics:response_time:avg", avg, time.Minute*5)
	}

	// 更新错误率统计
	errorKey := "metrics:errors:total"
	totalKey := "metrics:requests:total"

	amc.redisClient.Incr(ctx, totalKey)
	amc.redisClient.Expire(ctx, totalKey, time.Minute)

	if status >= 400 {
		amc.redisClient.Incr(ctx, errorKey)
		amc.redisClient.Expire(ctx, errorKey, time.Minute)
	}

	// 计算错误率
	go func() {
		errorCount, _ := amc.redisClient.Get(ctx, errorKey).Int()
		totalCount, _ := amc.redisClient.Get(ctx, totalKey).Int()

		if totalCount > 0 {
			errorRateVal := float64(errorCount) / float64(totalCount) * 100
			amc.redisClient.Set(ctx, "metrics:error_rate:current", errorRateVal, time.Minute*5)
			errorRate.WithLabelValues(endpoint).Set(errorRateVal)
		}
	}()

	// 更新后端成功率
	if status < 400 {
		backendSuccessRate.WithLabelValues("backend", endpoint).Set(100.0)
	} else {
		// 计算最近的成功率
		go amc.calculateBackendSuccessRate(ctx, endpoint)
	}

	// 更新活跃用户数（基于IP和API Key）
	userKey := c.GetHeader("Authorization")
	if userKey == "" {
		userKey = c.ClientIP()
	}

	activeUserKey := "metrics:active_users"
	amc.redisClient.SAdd(ctx, activeUserKey, userKey)
	amc.redisClient.Expire(ctx, activeUserKey, time.Minute)

	// 更新活跃用户数指标
	go func() {
		count, _ := amc.redisClient.SCard(ctx, activeUserKey).Result()
		activeUsers.Set(float64(count))
	}()
}

// calculateBackendSuccessRate 计算后端成功率
func (amc *AdvancedMetricsCollector) calculateBackendSuccessRate(ctx context.Context, endpoint string) {
	successKey := fmt.Sprintf("metrics:backend:success:%s", endpoint)
	totalKey := fmt.Sprintf("metrics:backend:total:%s", endpoint)

	successCount, _ := amc.redisClient.Get(ctx, successKey).Int()
	totalCount, _ := amc.redisClient.Get(ctx, totalKey).Int()

	if totalCount > 0 {
		successRateVal := float64(successCount) / float64(totalCount) * 100
		backendSuccessRate.WithLabelValues("backend", endpoint).Set(successRateVal)
	}
}

// StartMetricsCollector 启动指标收集器
func (amc *AdvancedMetricsCollector) StartMetricsCollector(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second) // 每10秒更新一次指标
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			amc.collectAndUpdateMetrics(ctx)
		}
	}
}

// collectAndUpdateMetrics 收集并更新指标
func (amc *AdvancedMetricsCollector) collectAndUpdateMetrics(ctx context.Context) {
	// 更新QPS指标
	qpsStr, _ := amc.redisClient.Get(ctx, "metrics:qps:current").Result()
	if qps, err := strconv.Atoi(qpsStr); err == nil {
		requestQPS.WithLabelValues("total").Set(float64(qps * 6)) // 转换为每分钟请求数
	}

	// 清理过期的指标数据
	amc.cleanupExpiredMetrics(ctx)
}

// cleanupExpiredMetrics 清理过期的指标数据
func (amc *AdvancedMetricsCollector) cleanupExpiredMetrics(ctx context.Context) {
	// 清理过期的QPS数据
	pattern := "metrics:qps:*"
	keys, _ := amc.redisClient.Keys(ctx, pattern).Result()
	for _, key := range keys {
		ttl, _ := amc.redisClient.TTL(ctx, key).Result()
		if ttl < 0 { // 没有过期时间的key
			amc.redisClient.Del(ctx, key)
		}
	}
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
