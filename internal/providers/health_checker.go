package providers

import (
	"context"
	"sync"
	"time"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	manager  *Manager
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(manager *Manager, interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		manager:  manager,
		interval: interval,
		timeout:  timeout,
		stopChan: make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.run()
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
}

// run 运行健康检查循环
func (hc *HealthChecker) run() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAllProviders()
		case <-hc.stopChan:
			return
		}
	}
}

// checkAllProviders 检查所有提供商的健康状态
func (hc *HealthChecker) checkAllProviders() {
	hc.manager.mu.RLock()
	providers := make(map[string]Provider)
	for name, provider := range hc.manager.providers {
		providers[name] = provider
	}
	hc.manager.mu.RUnlock()

	// 并发检查所有提供商
	var wg sync.WaitGroup
	for name, provider := range providers {
		wg.Add(1)
		go func(name string, provider Provider) {
			defer wg.Done()
			hc.checkProvider(name, provider)
		}(name, provider)
	}

	wg.Wait()
}

// checkProvider 检查单个提供商的健康状态
func (hc *HealthChecker) checkProvider(name string, provider Provider) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	// 执行健康检查
	err := provider.HealthCheck(ctx)

	// 更新状态
	hc.manager.mu.Lock()
	defer hc.manager.mu.Unlock()

	metrics := hc.manager.providerMetrics[name]
	if metrics == nil {
		return
	}

	if err != nil {
		metrics.Status = ProviderStatusUnhealthy
	} else {
		// 如果之前是不健康状态，恢复为健康状态
		if metrics.Status == ProviderStatusUnhealthy {
			metrics.Status = ProviderStatusHealthy
		}
	}
}
