package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert 告警信息
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

// Rule 监控规则
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

// MonitoringSystem 监控系统
type MonitoringSystem struct {
	redisClient *redis.Client
	rules       map[string]*Rule
	alerts      map[string]*Alert
}

// NewMonitoringSystem 创建监控系统
func NewMonitoringSystem(redisClient *redis.Client) *MonitoringSystem {
	ms := &MonitoringSystem{
		redisClient: redisClient,
		rules:       make(map[string]*Rule),
		alerts:      make(map[string]*Alert),
	}

	// 添加默认监控规则
	ms.addDefaultRules()

	return ms
}

// addDefaultRules 添加默认监控规则
func (ms *MonitoringSystem) addDefaultRules() {
	rules := []*Rule{
		{
			ID:          "high_qps",
			Name:        "High QPS Alert",
			Description: "QPS exceeds threshold",
			MetricKey:   "metrics:qps:current",
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
			MetricKey:   "metrics:error_rate:current",
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
			MetricKey:   "metrics:response_time:avg",
			Operator:    ">",
			Threshold:   2.0,
			Duration:    time.Minute * 3,
			Level:       AlertLevelWarning,
			Enabled:     true,
		},
		{
			ID:          "low_backend_success_rate",
			Name:        "Low Backend Success Rate Alert",
			Description: "Backend success rate below 95%",
			MetricKey:   "metrics:backend:success_rate",
			Operator:    "<",
			Threshold:   95.0,
			Duration:    time.Minute * 2,
			Level:       AlertLevelCritical,
			Enabled:     true,
		},
	}

	for _, rule := range rules {
		ms.rules[rule.ID] = rule
	}
}

// Start 启动监控系统
func (ms *MonitoringSystem) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	logrus.Info("Monitoring system started")

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Monitoring system stopped")
			return
		case <-ticker.C:
			ms.checkRules(ctx)
		}
	}
}

// checkRules 检查所有监控规则
func (ms *MonitoringSystem) checkRules(ctx context.Context) {
	for _, rule := range ms.rules {
		if !rule.Enabled {
			continue
		}

		if err := ms.checkRule(ctx, rule); err != nil {
			logrus.WithError(err).WithField("rule_id", rule.ID).Error("Failed to check monitoring rule")
		}
	}
}

// checkRule 检查单个监控规则
func (ms *MonitoringSystem) checkRule(ctx context.Context, rule *Rule) error {
	// 从Redis获取指标值
	valueStr, err := ms.redisClient.Get(ctx, rule.MetricKey).Result()
	if err == redis.Nil {
		return nil // 指标不存在，跳过
	} else if err != nil {
		return err
	}

	// 解析指标值
	var value float64
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		// 如果不是JSON，尝试直接解析为float64
		if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
			return fmt.Errorf("failed to parse metric value: %w", err)
		}
	}

	// 检查是否触发告警
	triggered := ms.evaluateCondition(value, rule.Operator, rule.Threshold)

	if triggered {
		// 创建或更新告警
		if err := ms.createOrUpdateAlert(ctx, rule, value); err != nil {
			return err
		}
	} else {
		// 解决告警
		if err := ms.resolveAlert(ctx, rule.ID); err != nil {
			return err
		}
	}

	return nil
}

// evaluateCondition 评估条件
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
