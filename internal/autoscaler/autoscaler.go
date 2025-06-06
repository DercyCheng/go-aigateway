package autoscaler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// AutoScaler 自动扩缩容器
type AutoScaler struct {
	redisClient       *redis.Client
	currentReplicas   int
	minReplicas       int
	maxReplicas       int
	targetCPU         float64 // 目标CPU使用率 (0-100)
	targetQPS         int     // 目标QPS
	scaleUpCooldown   time.Duration
	scaleDownCooldown time.Duration
	lastScaleTime     time.Time
	serviceName       string
}

// ScalingMetrics 扩缩容指标
type ScalingMetrics struct {
	CPUUsage            float64   `json:"cpu_usage"`
	MemoryUsage         float64   `json:"memory_usage"`
	CurrentQPS          int       `json:"current_qps"`
	AverageResponseTime float64   `json:"avg_response_time"`
	ErrorRate           float64   `json:"error_rate"`
	Timestamp           time.Time `json:"timestamp"`
}

// ScalingDecision 扩缩容决策
type ScalingDecision struct {
	Action       string    `json:"action"` // "scale_up", "scale_down", "no_action"
	FromReplicas int       `json:"from_replicas"`
	ToReplicas   int       `json:"to_replicas"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewAutoScaler 创建自动扩缩容器
func NewAutoScaler(redisClient *redis.Client, serviceName string) *AutoScaler {
	return &AutoScaler{
		redisClient:       redisClient,
		currentReplicas:   1,
		minReplicas:       1,
		maxReplicas:       10,
		targetCPU:         70.0,
		targetQPS:         1000,
		scaleUpCooldown:   time.Minute * 3,
		scaleDownCooldown: time.Minute * 5,
		serviceName:       serviceName,
	}
}

// Start 启动自动扩缩容
func (as *AutoScaler) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()

	logrus.Info("AutoScaler started")

	for {
		select {
		case <-ctx.Done():
			logrus.Info("AutoScaler stopped")
			return
		case <-ticker.C:
			if err := as.evaluate(ctx); err != nil {
				logrus.WithError(err).Error("Failed to evaluate scaling")
			}
		}
	}
}

// evaluate 评估是否需要扩缩容
func (as *AutoScaler) evaluate(ctx context.Context) error {
	// 获取当前指标
	metrics, err := as.getCurrentMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current metrics: %w", err)
	}

	// 存储指标到Redis
	if err := as.storeMetrics(ctx, metrics); err != nil {
		logrus.WithError(err).Warn("Failed to store metrics")
	}

	// 根据指标做扩缩容决策
	decision := as.makeScalingDecision(metrics)

	if decision.Action != "no_action" {
		logrus.WithFields(logrus.Fields{
			"action":        decision.Action,
			"from_replicas": decision.FromReplicas,
			"to_replicas":   decision.ToReplicas,
			"reason":        decision.Reason,
		}).Info("Scaling decision made")

		// 执行扩缩容
		if err := as.executeScaling(ctx, decision); err != nil {
			return fmt.Errorf("failed to execute scaling: %w", err)
		}

		// 存储扩缩容决策
		if err := as.storeScalingDecision(ctx, decision); err != nil {
			logrus.WithError(err).Warn("Failed to store scaling decision")
		}
	}

	return nil
}

// getCurrentMetrics 获取当前系统指标
func (as *AutoScaler) getCurrentMetrics(ctx context.Context) (*ScalingMetrics, error) {
	metrics := &ScalingMetrics{
		Timestamp: time.Now(),
	}
	// 从Redis获取QPS统计
	qpsKey := "metrics:qps:current"
	qpsStr, err := as.redisClient.Get(ctx, qpsKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if qpsStr != "" {
		if qps, err := strconv.Atoi(qpsStr); err == nil {
			metrics.CurrentQPS = qps
		}
	}
	// 从Redis获取平均响应时间
	responseTimeKey := "metrics:response_time:avg"
	responseTimeStr, err := as.redisClient.Get(ctx, responseTimeKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if responseTimeStr != "" {
		if responseTime, err := strconv.ParseFloat(responseTimeStr, 64); err == nil {
			metrics.AverageResponseTime = responseTime
		}
	}
	// 从Redis获取错误率
	errorRateKey := "metrics:error_rate:current"
	errorRateStr, err := as.redisClient.Get(ctx, errorRateKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if errorRateStr != "" {
		if errorRate, err := strconv.ParseFloat(errorRateStr, 64); err == nil {
			metrics.ErrorRate = errorRate
		}
	}

	// 模拟CPU和内存使用率（在实际环境中应该从容器监控API获取）
	metrics.CPUUsage = 45.0 + float64(metrics.CurrentQPS)/50.0 // 简单的模拟算法
	metrics.MemoryUsage = 30.0 + float64(metrics.CurrentQPS)/100.0

	return metrics, nil
}

// makeScalingDecision 做扩缩容决策
func (as *AutoScaler) makeScalingDecision(metrics *ScalingMetrics) *ScalingDecision {
	decision := &ScalingDecision{
		Action:       "no_action",
		FromReplicas: as.currentReplicas,
		ToReplicas:   as.currentReplicas,
		Timestamp:    time.Now(),
	}

	// 检查冷却时间
	if time.Since(as.lastScaleTime) < as.scaleUpCooldown {
		decision.Reason = "Still in cooldown period"
		return decision
	}

	// 扩容条件检查
	shouldScaleUp := false
	var scaleUpReasons []string

	if metrics.CPUUsage > as.targetCPU {
		shouldScaleUp = true
		scaleUpReasons = append(scaleUpReasons, fmt.Sprintf("CPU usage %.2f%% > target %.2f%%", metrics.CPUUsage, as.targetCPU))
	}

	if metrics.CurrentQPS > as.targetQPS {
		shouldScaleUp = true
		scaleUpReasons = append(scaleUpReasons, fmt.Sprintf("QPS %d > target %d", metrics.CurrentQPS, as.targetQPS))
	}

	if metrics.AverageResponseTime > 2.0 { // 响应时间超过2秒
		shouldScaleUp = true
		scaleUpReasons = append(scaleUpReasons, fmt.Sprintf("Response time %.2fs > 2.0s", metrics.AverageResponseTime))
	}

	if metrics.ErrorRate > 5.0 { // 错误率超过5%
		shouldScaleUp = true
		scaleUpReasons = append(scaleUpReasons, fmt.Sprintf("Error rate %.2f%% > 5%%", metrics.ErrorRate))
	}

	// 缩容条件检查
	shouldScaleDown := false
	var scaleDownReasons []string
	if metrics.CPUUsage < as.targetCPU*0.3 && // CPU使用率低于目标的30%
		metrics.CurrentQPS < int(float64(as.targetQPS)*0.3) && // QPS低于目标的30%
		metrics.AverageResponseTime < 0.5 && // 响应时间小于0.5秒
		as.currentReplicas > as.minReplicas {
		shouldScaleDown = true
		scaleDownReasons = append(scaleDownReasons, "Low resource utilization")
	}

	// 执行扩容
	if shouldScaleUp && as.currentReplicas < as.maxReplicas {
		decision.Action = "scale_up"
		decision.ToReplicas = as.currentReplicas + 1
		decision.Reason = fmt.Sprintf("Scale up: %v", scaleUpReasons)
	} else if shouldScaleDown && time.Since(as.lastScaleTime) >= as.scaleDownCooldown {
		decision.Action = "scale_down"
		decision.ToReplicas = as.currentReplicas - 1
		decision.Reason = fmt.Sprintf("Scale down: %v", scaleDownReasons)
	}

	return decision
}

// executeScaling 执行扩缩容操作
func (as *AutoScaler) executeScaling(ctx context.Context, decision *ScalingDecision) error {
	switch decision.Action {
	case "scale_up":
		return as.scaleUp(ctx, decision.ToReplicas)
	case "scale_down":
		return as.scaleDown(ctx, decision.ToReplicas)
	default:
		return nil
	}
}

// scaleUp 扩容
func (as *AutoScaler) scaleUp(ctx context.Context, replicas int) error {
	// 使用Docker Compose扩容
	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d", "--scale", fmt.Sprintf("%s=%d", as.serviceName, replicas))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to scale up: %w", err)
	}

	as.currentReplicas = replicas
	as.lastScaleTime = time.Now()

	logrus.WithFields(logrus.Fields{
		"service":  as.serviceName,
		"replicas": replicas,
	}).Info("Scaled up successfully")

	return nil
}

// scaleDown 缩容
func (as *AutoScaler) scaleDown(ctx context.Context, replicas int) error {
	// 使用Docker Compose缩容
	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d", "--scale", fmt.Sprintf("%s=%d", as.serviceName, replicas))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to scale down: %w", err)
	}

	as.currentReplicas = replicas
	as.lastScaleTime = time.Now()

	logrus.WithFields(logrus.Fields{
		"service":  as.serviceName,
		"replicas": replicas,
	}).Info("Scaled down successfully")

	return nil
}

// storeMetrics 存储指标到Redis
func (as *AutoScaler) storeMetrics(ctx context.Context, metrics *ScalingMetrics) error {
	key := fmt.Sprintf("autoscaler:metrics:%d", metrics.Timestamp.Unix())
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err := as.redisClient.Set(ctx, key, data, time.Hour*24).Err(); err != nil {
		return err
	}

	// 保持最近1000条记录
	listKey := "autoscaler:metrics:list"
	as.redisClient.RPush(ctx, listKey, key)
	as.redisClient.LTrim(ctx, listKey, -1000, -1)

	return nil
}

// storeScalingDecision 存储扩缩容决策
func (as *AutoScaler) storeScalingDecision(ctx context.Context, decision *ScalingDecision) error {
	key := fmt.Sprintf("autoscaler:decisions:%d", decision.Timestamp.Unix())
	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	if err := as.redisClient.Set(ctx, key, data, time.Hour*24*7).Err(); err != nil {
		return err
	}

	// 保持最近100条决策记录
	listKey := "autoscaler:decisions:list"
	as.redisClient.RPush(ctx, listKey, key)
	as.redisClient.LTrim(ctx, listKey, -100, -1)

	return nil
}

// GetScalingHistory 获取扩缩容历史
func (as *AutoScaler) GetScalingHistory(ctx context.Context, limit int) ([]*ScalingDecision, error) {
	listKey := "autoscaler:decisions:list"
	keys, err := as.redisClient.LRange(ctx, listKey, -int64(limit), -1).Result()
	if err != nil {
		return nil, err
	}

	var decisions []*ScalingDecision
	for _, key := range keys {
		data, err := as.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var decision ScalingDecision
		if err := json.Unmarshal([]byte(data), &decision); err != nil {
			continue
		}

		decisions = append(decisions, &decision)
	}

	return decisions, nil
}
