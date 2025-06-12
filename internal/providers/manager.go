package providers

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Manager AI服务提供商管理器
type Manager struct {
	providers       map[string]Provider
	providerMetrics map[string]*ProviderMetrics
	loadBalancer    LoadBalancer
	healthChecker   *HealthChecker
	mu              sync.RWMutex
	config          *ManagerConfig
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	LoadBalanceStrategy LoadBalanceStrategy `yaml:"load_balance_strategy"`
	HealthCheckEnabled  bool                `yaml:"health_check_enabled"`
	HealthCheckInterval time.Duration       `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration       `yaml:"health_check_timeout"`
	RetryEnabled        bool                `yaml:"retry_enabled"`
	MaxRetries          int                 `yaml:"max_retries"`
	RetryDelay          time.Duration       `yaml:"retry_delay"`
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	SelectProvider(providers []Provider, model string) (Provider, error)
}

// NewManager 创建管理器
func NewManager(config *ManagerConfig) *Manager {
	manager := &Manager{
		providers:       make(map[string]Provider),
		providerMetrics: make(map[string]*ProviderMetrics),
		config:          config,
	}

	// 初始化负载均衡器
	switch config.LoadBalanceStrategy {
	case LoadBalanceRoundRobin:
		manager.loadBalancer = NewRoundRobinBalancer()
	case LoadBalanceRandom:
		manager.loadBalancer = NewRandomBalancer()
	case LoadBalanceLeastConnections:
		manager.loadBalancer = NewLeastConnectionsBalancer(manager)
	case LoadBalanceWeighted:
		manager.loadBalancer = NewWeightedBalancer(manager)
	default:
		manager.loadBalancer = NewRoundRobinBalancer()
	}

	// 初始化健康检查器
	if config.HealthCheckEnabled {
		manager.healthChecker = NewHealthChecker(manager, config.HealthCheckInterval, config.HealthCheckTimeout)
		manager.healthChecker.Start()
	}

	return manager
}

// RegisterProvider 注册提供商
func (m *Manager) RegisterProvider(provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := provider.GetName()
	m.providers[name] = provider
	m.providerMetrics[name] = &ProviderMetrics{
		Status: ProviderStatusHealthy,
	}
}

// GetProvider 获取指定提供商
func (m *Manager) GetProvider(name string) (Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	return provider, exists
}

// GetHealthyProviders 获取健康的提供商列表
func (m *Manager) GetHealthyProviders() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthyProviders []Provider
	for name, provider := range m.providers {
		if metrics := m.providerMetrics[name]; metrics.Status == ProviderStatusHealthy {
			healthyProviders = append(healthyProviders, provider)
		}
	}

	return healthyProviders
}

// GetProvidersForModel 获取支持指定模型的提供商
func (m *Manager) GetProvidersForModel(model string) []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var supportedProviders []Provider
	for name, provider := range m.providers {
		metrics := m.providerMetrics[name]
		if metrics.Status != ProviderStatusHealthy {
			continue
		}

		// 检查提供商是否支持该模型
		for _, supportedModel := range provider.GetModels() {
			if supportedModel.Name == model {
				supportedProviders = append(supportedProviders, provider)
				break
			}
		}
	}

	return supportedProviders
}

// Chat 聊天补全（自动选择提供商）
func (m *Manager) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	providers := m.GetProvidersForModel(req.Model)
	if len(providers) == 0 {
		return nil, fmt.Errorf("no healthy providers found for model: %s", req.Model)
	}

	var lastErr error
	maxRetries := 1
	if m.config.RetryEnabled {
		maxRetries = m.config.MaxRetries
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 选择提供商
		provider, err := m.loadBalancer.SelectProvider(providers, req.Model)
		if err != nil {
			lastErr = err
			continue
		}

		// 记录请求开始时间
		startTime := time.Now()

		// 发送请求
		response, err := provider.Chat(ctx, req)

		// 更新指标
		m.updateMetrics(provider.GetName(), startTime, err)

		if err == nil {
			return response, nil
		}

		lastErr = err

		// 如果不是最后一次尝试，则等待重试延迟
		if attempt < maxRetries-1 && m.config.RetryEnabled {
			time.Sleep(m.config.RetryDelay)
		}
	}

	return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// ChatStream 流式聊天补全（自动选择提供商）
func (m *Manager) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamResponse, error) {
	providers := m.GetProvidersForModel(req.Model)
	if len(providers) == 0 {
		return nil, fmt.Errorf("no healthy providers found for model: %s", req.Model)
	}

	// 选择提供商
	provider, err := m.loadBalancer.SelectProvider(providers, req.Model)
	if err != nil {
		return nil, err
	}

	// 发送流式请求
	return provider.ChatStream(ctx, req)
}

// updateMetrics 更新提供商指标
func (m *Manager) updateMetrics(providerName string, startTime time.Time, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.providerMetrics[providerName]
	if metrics == nil {
		return
	}

	latency := time.Since(startTime)
	metrics.RequestCount++
	metrics.LastRequestTime = time.Now()

	// 更新平均延迟
	if metrics.RequestCount == 1 {
		metrics.AverageLatency = latency
	} else {
		// 使用指数移动平均
		alpha := 0.1 // 平滑因子
		metrics.AverageLatency = time.Duration(float64(metrics.AverageLatency)*(1-alpha) + float64(latency)*alpha)
	}

	if err != nil {
		metrics.ErrorCount++
		// 如果错误率过高，标记为不健康
		errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
		if errorRate > 0.5 && metrics.RequestCount > 10 {
			metrics.Status = ProviderStatusUnhealthy
		}
	}
}

// GetMetrics 获取所有提供商指标
func (m *Manager) GetMetrics() map[string]*ProviderMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ProviderMetrics)
	for name, metrics := range m.providerMetrics {
		// 复制指标，避免并发访问问题
		result[name] = &ProviderMetrics{
			RequestCount:    metrics.RequestCount,
			ErrorCount:      metrics.ErrorCount,
			AverageLatency:  metrics.AverageLatency,
			LastRequestTime: metrics.LastRequestTime,
			Status:          metrics.Status,
		}
	}

	return result
}

// Stop 停止管理器
func (m *Manager) Stop() {
	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	counter int
	mu      sync.Mutex
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

func (b *RoundRobinBalancer) SelectProvider(providers []Provider, model string) (Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	provider := providers[b.counter%len(providers)]
	b.counter++

	return provider, nil
}

// RandomBalancer 随机负载均衡器
type RandomBalancer struct{}

func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{}
}

func (b *RandomBalancer) SelectProvider(providers []Provider, model string) (Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	index := rand.Intn(len(providers))
	return providers[index], nil
}

// LeastConnectionsBalancer 最少连接负载均衡器
type LeastConnectionsBalancer struct {
	manager *Manager
}

func NewLeastConnectionsBalancer(manager *Manager) *LeastConnectionsBalancer {
	return &LeastConnectionsBalancer{manager: manager}
}

func (b *LeastConnectionsBalancer) SelectProvider(providers []Provider, model string) (Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// 选择请求数最少的提供商
	var bestProvider Provider
	var minRequests int64 = -1

	for _, provider := range providers {
		metrics := b.manager.providerMetrics[provider.GetName()]
		if metrics != nil && (minRequests == -1 || metrics.RequestCount < minRequests) {
			minRequests = metrics.RequestCount
			bestProvider = provider
		}
	}

	if bestProvider == nil {
		return providers[0], nil
	}

	return bestProvider, nil
}

// WeightedBalancer 加权负载均衡器
type WeightedBalancer struct {
	manager *Manager
}

func NewWeightedBalancer(manager *Manager) *WeightedBalancer {
	return &WeightedBalancer{manager: manager}
}

func (b *WeightedBalancer) SelectProvider(providers []Provider, model string) (Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// 根据错误率和延迟计算权重
	type providerScore struct {
		provider Provider
		score    float64
	}

	var scores []providerScore

	for _, provider := range providers {
		metrics := b.manager.providerMetrics[provider.GetName()]
		if metrics == nil {
			scores = append(scores, providerScore{provider: provider, score: 1.0})
			continue
		}

		// 计算分数（越高越好）
		score := 1.0

		// 错误率惩罚
		if metrics.RequestCount > 0 {
			errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount)
			score *= (1.0 - errorRate)
		}

		// 延迟惩罚
		if metrics.AverageLatency > 0 {
			// 假设1秒延迟对应0.5的惩罚
			latencyPenalty := float64(metrics.AverageLatency) / float64(time.Second) * 0.5
			score *= (1.0 - latencyPenalty)
		}

		if score < 0.1 {
			score = 0.1 // 最小分数
		}

		scores = append(scores, providerScore{provider: provider, score: score})
	}

	// 按分数排序，选择最高分的
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].provider, nil
}
