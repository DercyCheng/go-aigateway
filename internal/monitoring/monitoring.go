package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// AlertLevel represents alert severity levels
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents a monitoring alert
type Alert struct {
	ID         string                 `json:"id"`
	Level      AlertLevel             `json:"level"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Rule represents a monitoring rule
type Rule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	MetricKey   string        `json:"metric_key"`
	Operator    string        `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Level       AlertLevel    `json:"level"`
	Enabled     bool          `json:"enabled"`
}

// Metrics represents system metrics
type Metrics struct {
	RequestCount        int64     `json:"request_count"`
	ErrorCount          int64     `json:"error_count"`
	AverageResponseTime float64   `json:"average_response_time"`
	QPS                 float64   `json:"qps"`
	ErrorRate           float64   `json:"error_rate"`
	CPUUsage            float64   `json:"cpu_usage"`
	MemoryUsage         float64   `json:"memory_usage"`
	GoroutineCount      int       `json:"goroutine_count"`
	Timestamp           time.Time `json:"timestamp"`
}

// MonitoringSystem represents the monitoring system
type MonitoringSystem struct {
	config      *config.MonitoringConfig
	redisClient *redis.Client
	rules       map[string]*Rule
	alerts      map[string]*Alert
	metrics     *Metrics
	mutex       sync.RWMutex

	// Prometheus metrics
	requestCounter    prometheus.Counter
	errorCounter      prometheus.Counter
	responseTimeHist  prometheus.Histogram
	activeConnections prometheus.Gauge
	systemCPU         prometheus.Gauge
	systemMemory      prometheus.Gauge

	// Channels for real-time monitoring
	metricsChan chan *Metrics
	alertsChan  chan *Alert
	stopChan    chan struct{}
}

// NewMonitoringSystem creates a new monitoring system
func NewMonitoringSystem(cfg *config.MonitoringConfig, redisClient *redis.Client) *MonitoringSystem {
	if !cfg.Enabled {
		return nil
	}

	ms := &MonitoringSystem{
		config:      cfg,
		redisClient: redisClient,
		rules:       make(map[string]*Rule),
		alerts:      make(map[string]*Alert),
		metrics:     &Metrics{},
		metricsChan: make(chan *Metrics, 100),
		alertsChan:  make(chan *Alert, 100),
		stopChan:    make(chan struct{}),
	}

	// Initialize Prometheus metrics
	ms.initPrometheusMetrics()

	// Add default monitoring rules
	ms.addDefaultRules()

	// Start background monitoring
	go ms.backgroundMonitoring()
	go ms.metricsCollector()
	go ms.alertProcessor()

	return ms
}

// initPrometheusMetrics initializes Prometheus metrics
func (ms *MonitoringSystem) initPrometheusMetrics() {
	ms.requestCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "aigateway_requests_total",
		Help: "Total number of requests processed",
	})

	ms.errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "aigateway_errors_total",
		Help: "Total number of errors",
	})

	ms.responseTimeHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "aigateway_response_time_seconds",
		Help:    "Response time in seconds",
		Buckets: prometheus.DefBuckets,
	})

	ms.activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aigateway_active_connections",
		Help: "Number of active connections",
	})

	ms.systemCPU = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aigateway_cpu_usage_percent",
		Help: "CPU usage percentage",
	})

	ms.systemMemory = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aigateway_memory_usage_bytes",
		Help: "Memory usage in bytes",
	})

	// Register all metrics
	prometheus.MustRegister(
		ms.requestCounter,
		ms.errorCounter,
		ms.responseTimeHist,
		ms.activeConnections,
		ms.systemCPU,
		ms.systemMemory,
	)
}

// addDefaultRules adds default monitoring rules
func (ms *MonitoringSystem) addDefaultRules() {
	rules := []*Rule{
		{
			ID:          "high_qps",
			Name:        "High QPS Alert",
			Description: "QPS exceeds threshold",
			MetricKey:   "qps",
			Operator:    ">",
			Threshold:   1000,
			Duration:    time.Minute * 2,
			Level:       AlertLevelWarning,
			Enabled:     true,
		},
		{
			ID:          "high_error_rate",
			Name:        "High Error Rate Alert",
			Description: "Error rate exceeds 5%",
			MetricKey:   "error_rate",
			Operator:    ">",
			Threshold:   5.0,
			Duration:    time.Minute * 1,
			Level:       AlertLevelCritical,
			Enabled:     true,
		},
		{
			ID:          "high_response_time",
			Name:        "High Response Time Alert",
			Description: "Average response time exceeds 2 seconds",
			MetricKey:   "average_response_time",
			Operator:    ">",
			Threshold:   2.0,
			Duration:    time.Minute * 3,
			Level:       AlertLevelWarning,
			Enabled:     true,
		},
		{
			ID:          "high_cpu_usage",
			Name:        "High CPU Usage Alert",
			Description: "CPU usage exceeds 80%",
			MetricKey:   "cpu_usage",
			Operator:    ">",
			Threshold:   80.0,
			Duration:    time.Minute * 5,
			Level:       AlertLevelWarning,
			Enabled:     true,
		},
		{
			ID:          "high_memory_usage",
			Name:        "High Memory Usage Alert",
			Description: "Memory usage exceeds 85%",
			MetricKey:   "memory_usage",
			Operator:    ">",
			Threshold:   85.0,
			Duration:    time.Minute * 5,
			Level:       AlertLevelCritical,
			Enabled:     true,
		},
	}

	for _, rule := range rules {
		ms.rules[rule.ID] = rule
	}
}

// RecordRequest records a new request metric
func (ms *MonitoringSystem) RecordRequest() {
	if ms == nil {
		return
	}
	ms.requestCounter.Inc()

	ms.mutex.Lock()
	ms.metrics.RequestCount++
	ms.mutex.Unlock()
}

// RecordError records an error metric
func (ms *MonitoringSystem) RecordError() {
	if ms == nil {
		return
	}
	ms.errorCounter.Inc()

	ms.mutex.Lock()
	ms.metrics.ErrorCount++
	ms.mutex.Unlock()
}

// RecordResponseTime records response time metric
func (ms *MonitoringSystem) RecordResponseTime(duration time.Duration) {
	if ms == nil {
		return
	}
	ms.responseTimeHist.Observe(duration.Seconds())
}

// UpdateActiveConnections updates active connections metric
func (ms *MonitoringSystem) UpdateActiveConnections(count int) {
	if ms == nil {
		return
	}
	ms.activeConnections.Set(float64(count))
}

// backgroundMonitoring runs background monitoring tasks
func (ms *MonitoringSystem) backgroundMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ms.stopChan:
			return
		case <-ticker.C:
			ms.collectSystemMetrics()
			ms.checkRules()
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (ms *MonitoringSystem) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ms.mutex.Lock()
	ms.metrics.GoroutineCount = runtime.NumGoroutine()
	ms.metrics.MemoryUsage = float64(m.Alloc) / 1024 / 1024 // MB
	ms.metrics.Timestamp = time.Now()

	// Calculate QPS and error rate from counters
	if ms.metrics.RequestCount > 0 {
		ms.metrics.ErrorRate = (float64(ms.metrics.ErrorCount) / float64(ms.metrics.RequestCount)) * 100
	}
	ms.mutex.Unlock()

	// Update Prometheus metrics
	ms.systemMemory.Set(float64(m.Alloc))

	// Send metrics to channel for processing
	select {
	case ms.metricsChan <- ms.metrics:
	default:
		// Channel full, skip this update
	}
}

// checkRules checks monitoring rules and generates alerts
func (ms *MonitoringSystem) checkRules() {
	ms.mutex.RLock()
	currentMetrics := *ms.metrics
	ms.mutex.RUnlock()

	for _, rule := range ms.rules {
		if !rule.Enabled {
			continue
		}

		var value float64
		switch rule.MetricKey {
		case "qps":
			value = currentMetrics.QPS
		case "error_rate":
			value = currentMetrics.ErrorRate
		case "average_response_time":
			value = currentMetrics.AverageResponseTime
		case "cpu_usage":
			value = currentMetrics.CPUUsage
		case "memory_usage":
			value = currentMetrics.MemoryUsage
		default:
			continue
		}

		if ms.evaluateCondition(value, rule.Operator, rule.Threshold) {
			alert := &Alert{
				ID:        fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix()),
				Level:     rule.Level,
				Title:     rule.Name,
				Message:   fmt.Sprintf("%s: %s %s %.2f (threshold: %.2f)", rule.Name, rule.MetricKey, rule.Operator, value, rule.Threshold),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"rule_id":       rule.ID,
					"metric_key":    rule.MetricKey,
					"current_value": value,
					"threshold":     rule.Threshold,
					"operator":      rule.Operator,
				},
			}

			ms.alerts[alert.ID] = alert

			// Send alert to channel
			select {
			case ms.alertsChan <- alert:
			default:
				logrus.Warn("Alert channel full, dropping alert")
			}

			logrus.WithFields(logrus.Fields{
				"alert_id": alert.ID,
				"level":    alert.Level,
				"message":  alert.Message,
			}).Warn("Alert triggered")
		}
	}
}

// evaluateCondition evaluates a monitoring condition
func (ms *MonitoringSystem) evaluateCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// metricsCollector processes metrics from the channel
func (ms *MonitoringSystem) metricsCollector() {
	for {
		select {
		case <-ms.stopChan:
			return
		case metrics := <-ms.metricsChan:
			ms.storeMetrics(metrics)
		}
	}
}

// alertProcessor processes alerts from the channel
func (ms *MonitoringSystem) alertProcessor() {
	for {
		select {
		case <-ms.stopChan:
			return
		case alert := <-ms.alertsChan:
			ms.processAlert(alert)
		}
	}
}

// storeMetrics stores metrics to Redis
func (ms *MonitoringSystem) storeMetrics(metrics *Metrics) {
	if ms.redisClient == nil {
		return
	}

	ctx := context.Background()

	// Store current metrics
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal metrics")
		return
	}

	key := fmt.Sprintf("metrics:current:%d", time.Now().Unix())
	if err := ms.redisClient.Set(ctx, key, metricsJSON, ms.config.MetricsRetention).Err(); err != nil {
		logrus.WithError(err).Error("Failed to store metrics in Redis")
	}

	// Store time-series data
	pipe := ms.redisClient.Pipeline()
	timestamp := time.Now().Unix()

	pipe.ZAdd(ctx, "metrics:qps", redis.Z{Score: float64(timestamp), Member: metrics.QPS})
	pipe.ZAdd(ctx, "metrics:error_rate", redis.Z{Score: float64(timestamp), Member: metrics.ErrorRate})
	pipe.ZAdd(ctx, "metrics:response_time", redis.Z{Score: float64(timestamp), Member: metrics.AverageResponseTime})
	pipe.ZAdd(ctx, "metrics:cpu_usage", redis.Z{Score: float64(timestamp), Member: metrics.CPUUsage})
	pipe.ZAdd(ctx, "metrics:memory_usage", redis.Z{Score: float64(timestamp), Member: metrics.MemoryUsage})

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		logrus.WithError(err).Error("Failed to store time-series metrics")
	}
}

// processAlert processes and potentially sends alerts
func (ms *MonitoringSystem) processAlert(alert *Alert) {
	if ms.redisClient == nil {
		return
	}

	ctx := context.Background()

	// Store alert in Redis
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal alert")
		return
	}

	key := fmt.Sprintf("alerts:%s", alert.ID)
	if err := ms.redisClient.Set(ctx, key, alertJSON, 24*time.Hour).Err(); err != nil {
		logrus.WithError(err).Error("Failed to store alert in Redis")
	}

	// Add to alerts list
	if err := ms.redisClient.LPush(ctx, "alerts:list", alert.ID).Err(); err != nil {
		logrus.WithError(err).Error("Failed to add alert to list")
	}

	// Keep only recent alerts (last 1000)
	ms.redisClient.LTrim(ctx, "alerts:list", 0, 999)
}

// GetMetrics returns current system metrics
func (ms *MonitoringSystem) GetMetrics() *Metrics {
	if ms == nil {
		return nil
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	metrics := *ms.metrics
	return &metrics
}

// GetAlerts returns recent alerts
func (ms *MonitoringSystem) GetAlerts(limit int) ([]*Alert, error) {
	if ms == nil {
		return nil, fmt.Errorf("monitoring system not enabled")
	}

	if ms.redisClient == nil {
		// Return in-memory alerts
		var alerts []*Alert
		count := 0
		for _, alert := range ms.alerts {
			if count >= limit {
				break
			}
			alerts = append(alerts, alert)
			count++
		}
		return alerts, nil
	}

	ctx := context.Background()

	// Get alert IDs from Redis
	alertIDs, err := ms.redisClient.LRange(ctx, "alerts:list", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	var alerts []*Alert
	for _, alertID := range alertIDs {
		alertJSON, err := ms.redisClient.Get(ctx, fmt.Sprintf("alerts:%s", alertID)).Result()
		if err != nil {
			continue
		}

		var alert Alert
		if err := json.Unmarshal([]byte(alertJSON), &alert); err != nil {
			continue
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetMetricsHandler returns an HTTP handler for Prometheus metrics
func (ms *MonitoringSystem) GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// Close stops the monitoring system
func (ms *MonitoringSystem) Close() error {
	if ms == nil {
		return nil
	}

	close(ms.stopChan)
	return nil
}

// createOrUpdateAlert 创建或更新告警
func (ms *MonitoringSystem) createOrUpdateAlert(ctx context.Context, rule *Rule, value float64) error {
	alertID := rule.ID

	// 检查是否已存在未解决的告警
	if alert, exists := ms.alerts[alertID]; exists && !alert.Resolved {
		return nil // 告警已存在且未解决
	}

	// 创建新告警
	alert := &Alert{
		ID:        alertID,
		Level:     rule.Level,
		Title:     rule.Name,
		Message:   fmt.Sprintf("%s: current value %.2f %s threshold %.2f", rule.Description, value, rule.Operator, rule.Threshold),
		Timestamp: time.Now(),
		Resolved:  false,
		Metadata: map[string]interface{}{
			"rule_id":       rule.ID,
			"current_value": value,
			"threshold":     rule.Threshold,
			"operator":      rule.Operator,
		},
	}

	ms.alerts[alertID] = alert

	// 存储到Redis
	alertData, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	alertKey := fmt.Sprintf("alerts:%s", alertID)
	if err := ms.redisClient.Set(ctx, alertKey, alertData, time.Hour*24).Err(); err != nil {
		return err
	}

	// 添加到告警列表
	alertListKey := "alerts:active"
	ms.redisClient.SAdd(ctx, alertListKey, alertID)

	// 记录日志
	logrus.WithFields(logrus.Fields{
		"alert_id":      alert.ID,
		"level":         alert.Level,
		"title":         alert.Title,
		"current_value": value,
		"threshold":     rule.Threshold,
	}).Warn("Alert triggered")

	return nil
}

// resolveAlert 解决告警
func (ms *MonitoringSystem) resolveAlert(ctx context.Context, ruleID string) error {
	alertID := ruleID

	alert, exists := ms.alerts[alertID]
	if !exists || alert.Resolved {
		return nil // 告警不存在或已解决
	}

	// 标记为已解决
	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	// 更新Redis
	alertData, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	alertKey := fmt.Sprintf("alerts:%s", alertID)
	if err := ms.redisClient.Set(ctx, alertKey, alertData, time.Hour*24).Err(); err != nil {
		return err
	}

	// 从活跃告警列表中移除
	alertListKey := "alerts:active"
	ms.redisClient.SRem(ctx, alertListKey, alertID)

	// 添加到已解决告警列表
	resolvedListKey := "alerts:resolved"
	ms.redisClient.SAdd(ctx, resolvedListKey, alertID)

	logrus.WithFields(logrus.Fields{
		"alert_id": alert.ID,
		"title":    alert.Title,
	}).Info("Alert resolved")

	return nil
}

// GetActiveAlerts 获取活跃告警
func (ms *MonitoringSystem) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	alertListKey := "alerts:active"
	alertIDs, err := ms.redisClient.SMembers(ctx, alertListKey).Result()
	if err != nil {
		return nil, err
	}

	var alerts []*Alert
	for _, alertID := range alertIDs {
		alertKey := fmt.Sprintf("alerts:%s", alertID)
		alertData, err := ms.redisClient.Get(ctx, alertKey).Result()
		if err != nil {
			continue
		}

		var alert Alert
		if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
			continue
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetAlertHistory 获取告警历史
func (ms *MonitoringSystem) GetAlertHistory(ctx context.Context, limit int) ([]*Alert, error) {
	// 获取活跃和已解决的告警
	activeAlerts, _ := ms.GetActiveAlerts(ctx)

	resolvedListKey := "alerts:resolved"
	resolvedIDs, err := ms.redisClient.SMembers(ctx, resolvedListKey).Result()
	if err != nil {
		return activeAlerts, nil
	}

	var allAlerts []*Alert
	allAlerts = append(allAlerts, activeAlerts...)

	for _, alertID := range resolvedIDs {
		if len(allAlerts) >= limit {
			break
		}

		alertKey := fmt.Sprintf("alerts:%s", alertID)
		alertData, err := ms.redisClient.Get(ctx, alertKey).Result()
		if err != nil {
			continue
		}

		var alert Alert
		if err := json.Unmarshal([]byte(alertData), &alert); err != nil {
			continue
		}

		allAlerts = append(allAlerts, &alert)
	}

	return allAlerts, nil
}

// AddRule 添加监控规则
func (ms *MonitoringSystem) AddRule(rule *Rule) {
	ms.rules[rule.ID] = rule
}

// RemoveRule 移除监控规则
func (ms *MonitoringSystem) RemoveRule(ruleID string) {
	delete(ms.rules, ruleID)
}

// GetRules 获取所有监控规则
func (ms *MonitoringSystem) GetRules() map[string]*Rule {
	return ms.rules
}
