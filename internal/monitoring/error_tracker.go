package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// ErrorTracker tracks and analyzes errors for alerting
type ErrorTracker struct {
	redis  *redis.Client
	logger *logrus.Logger

	// Prometheus metrics
	errorCounter   *prometheus.CounterVec
	errorRate      *prometheus.GaugeVec
	responseTime   *prometheus.HistogramVec
	securityEvents *prometheus.CounterVec
}

// ErrorEvent represents an error event
type ErrorEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Code        string                 `json:"code"`
	Source      string                 `json:"source"`
	UserID      string                 `json:"user_id,omitempty"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	RequestPath string                 `json:"request_path,omitempty"`
	Method      string                 `json:"method,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(redisClient *redis.Client) *ErrorTracker {
	return &ErrorTracker{
		redis:  redisClient,
		logger: logrus.New(),

		errorCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "error_events_total",
				Help: "Total number of error events",
			},
			[]string{"level", "code", "source"},
		),

		errorRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "error_rate",
				Help: "Current error rate per minute",
			},
			[]string{"level", "source"},
		),

		responseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),

		securityEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_events_total",
				Help: "Total number of security events",
			},
			[]string{"event_type", "severity"},
		),
	}
}

// TrackError tracks an error event
func (et *ErrorTracker) TrackError(ctx context.Context, event *ErrorEvent) error {
	// Update Prometheus metrics
	et.errorCounter.WithLabelValues(event.Level, event.Code, event.Source).Inc()

	// Store in Redis for analysis
	eventJSON, err := json.Marshal(event)
	if err != nil {
		et.logger.WithError(err).Error("Failed to marshal error event")
		return err
	}

	// Store with TTL
	key := fmt.Sprintf("errors:%s:%d", event.Source, event.Timestamp.Unix())
	err = et.redis.Set(ctx, key, eventJSON, 24*time.Hour).Err()
	if err != nil {
		et.logger.WithError(err).Error("Failed to store error event in Redis")
		return err
	}

	// Update error rate metrics
	go et.updateErrorRates(ctx)

	// Check for alert conditions
	go et.checkAlertConditions(ctx, event)

	return nil
}

// TrackSecurityEvent tracks a security-related event
func (et *ErrorTracker) TrackSecurityEvent(ctx context.Context, eventType, severity string, details map[string]interface{}) {
	et.securityEvents.WithLabelValues(eventType, severity).Inc()

	event := &ErrorEvent{
		Timestamp: time.Now(),
		Level:     "security",
		Message:   fmt.Sprintf("Security event: %s", eventType),
		Code:      eventType,
		Source:    "security",
		Details:   details,
	}

	if err := et.TrackError(ctx, event); err != nil {
		et.logger.WithError(err).Error("Failed to track security event")
	}
}

// updateErrorRates calculates and updates error rate metrics
func (et *ErrorTracker) updateErrorRates(ctx context.Context) {
	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	// Query recent errors
	pattern := fmt.Sprintf("errors:*:%d", oneMinuteAgo.Unix())
	keys, err := et.redis.Keys(ctx, pattern).Result()
	if err != nil {
		et.logger.WithError(err).Error("Failed to query error keys")
		return
	}

	// Count errors by level and source
	errorCounts := make(map[string]map[string]int)
	for _, key := range keys {
		eventJSON, err := et.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var event ErrorEvent
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
			continue
		}

		if errorCounts[event.Level] == nil {
			errorCounts[event.Level] = make(map[string]int)
		}
		errorCounts[event.Level][event.Source]++
	}

	// Update metrics
	for level, sources := range errorCounts {
		for source, count := range sources {
			et.errorRate.WithLabelValues(level, source).Set(float64(count))
		}
	}
}

// checkAlertConditions checks if any alert conditions are met
func (et *ErrorTracker) checkAlertConditions(ctx context.Context, event *ErrorEvent) {
	// High error rate check
	if event.Level == "error" || event.Level == "critical" {
		recentErrors := et.countRecentErrors(ctx, event.Source, 5*time.Minute)
		if recentErrors > 10 {
			et.logger.WithFields(logrus.Fields{
				"source":        event.Source,
				"recent_errors": recentErrors,
				"threshold":     10,
			}).Warn("High error rate detected")

			// Here you would integrate with your alerting system
			// e.g., send to Slack, PagerDuty, etc.
		}
	}

	// Security event check
	if event.Level == "security" {
		recentSecurityEvents := et.countRecentSecurityEvents(ctx, 5*time.Minute)
		if recentSecurityEvents > 5 {
			et.logger.WithFields(logrus.Fields{
				"recent_security_events": recentSecurityEvents,
				"threshold":              5,
			}).Error("Multiple security events detected")

			// Immediate alert for security issues
		}
	}
}

// countRecentErrors counts errors in the specified time window
func (et *ErrorTracker) countRecentErrors(ctx context.Context, source string, window time.Duration) int {
	now := time.Now()
	start := now.Add(-window)

	pattern := fmt.Sprintf("errors:%s:*", source)
	keys, err := et.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return 0
	}

	count := 0
	for _, key := range keys {
		// Extract timestamp from key
		var timestamp int64
		if _, err := fmt.Sscanf(key, "errors:%s:%d", source, &timestamp); err != nil {
			continue
		}

		eventTime := time.Unix(timestamp, 0)
		if eventTime.After(start) {
			count++
		}
	}

	return count
}

// countRecentSecurityEvents counts security events in the specified time window
func (et *ErrorTracker) countRecentSecurityEvents(ctx context.Context, window time.Duration) int {
	return et.countRecentErrors(ctx, "security", window)
}
