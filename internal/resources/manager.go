package resources

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"go-aigateway/internal/errors"

	"github.com/sirupsen/logrus"
)

// ResourceManager manages application resources with proper cleanup
type ResourceManager struct {
	resources map[string]ManagedResource
	mutex     sync.RWMutex
	logger    *logrus.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// ManagedResource represents a resource that needs cleanup
type ManagedResource interface {
	ID() string
	Type() string
	Close() error
	HealthCheck() error
}

// ResourceConfig holds configuration for resource management
type ResourceConfig struct {
	MaxIdleTime     time.Duration
	HealthCheckRate time.Duration
	CleanupTimeout  time.Duration
}

// NewResourceManager creates a new resource manager
func NewResourceManager(cfg *ResourceConfig) *ResourceManager {
	ctx, cancel := context.WithCancel(context.Background())

	rm := &ResourceManager{
		resources: make(map[string]ManagedResource),
		logger:    logrus.New(),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start background cleanup and health monitoring
	go rm.backgroundMaintenance(cfg)

	return rm
}

// Register adds a resource to be managed
func (rm *ResourceManager) Register(resource ManagedResource) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.resources[resource.ID()]; exists {
		return errors.New(errors.ErrCodeValidation, "Resource already registered: "+resource.ID())
	}

	rm.resources[resource.ID()] = resource
	rm.logger.WithFields(logrus.Fields{
		"resource_id":   resource.ID(),
		"resource_type": resource.Type(),
	}).Info("Resource registered")

	return nil
}

// Unregister removes a resource from management
func (rm *ResourceManager) Unregister(resourceID string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	resource, exists := rm.resources[resourceID]
	if !exists {
		return errors.NotFoundError("Resource: " + resourceID)
	}

	if err := resource.Close(); err != nil {
		rm.logger.WithError(err).WithField("resource_id", resourceID).Error("Failed to close resource")
		return errors.Wrap(errors.ErrCodeResource, "Failed to close resource", err)
	}

	delete(rm.resources, resourceID)
	rm.logger.WithField("resource_id", resourceID).Info("Resource unregistered")

	return nil
}

// Get retrieves a managed resource
func (rm *ResourceManager) Get(resourceID string) (ManagedResource, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	resource, exists := rm.resources[resourceID]
	if !exists {
		return nil, errors.NotFoundError("Resource: " + resourceID)
	}

	return resource, nil
}

// List returns all registered resources
func (rm *ResourceManager) List() map[string]ManagedResource {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	result := make(map[string]ManagedResource)
	for id, resource := range rm.resources {
		result[id] = resource
	}

	return result
}

// CloseAll closes all managed resources
func (rm *ResourceManager) CloseAll() error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	var allErrors []error

	for id, resource := range rm.resources {
		if err := resource.Close(); err != nil {
			rm.logger.WithError(err).WithField("resource_id", id).Error("Failed to close resource")
			allErrors = append(allErrors, fmt.Errorf("resource %s: %w", id, err))
		}
	}

	rm.resources = make(map[string]ManagedResource)

	if len(allErrors) > 0 {
		return errors.NewWithDetails(errors.ErrCodeResource, "Failed to close some resources", allErrors)
	}

	return nil
}

// Shutdown gracefully shuts down the resource manager
func (rm *ResourceManager) Shutdown(timeout time.Duration) error {
	rm.cancel()

	done := make(chan error, 1)
	go func() {
		done <- rm.CloseAll()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errors.TimeoutError("Resource manager shutdown")
	}
}

// backgroundMaintenance performs periodic health checks and cleanup
func (rm *ResourceManager) backgroundMaintenance(cfg *ResourceConfig) {
	ticker := time.NewTicker(cfg.HealthCheckRate)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performHealthChecks()
			rm.performMemoryCleanup()
		}
	}
}

// performHealthChecks checks the health of all resources
func (rm *ResourceManager) performHealthChecks() {
	rm.mutex.RLock()
	resources := make(map[string]ManagedResource)
	for id, resource := range rm.resources {
		resources[id] = resource
	}
	rm.mutex.RUnlock()

	for id, resource := range resources {
		if err := resource.HealthCheck(); err != nil {
			rm.logger.WithError(err).WithField("resource_id", id).Warn("Resource health check failed")

			// Try to remove unhealthy resource
			if err := rm.Unregister(id); err != nil {
				rm.logger.WithError(err).WithField("resource_id", id).Error("Failed to unregister unhealthy resource")
			}
		}
	}
}

// performMemoryCleanup forces garbage collection periodically
func (rm *ResourceManager) performMemoryCleanup() {
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	runtime.GC()

	runtime.ReadMemStats(&memAfter)

	if memBefore.Alloc > memAfter.Alloc {
		freed := memBefore.Alloc - memAfter.Alloc
		rm.logger.WithField("freed_bytes", freed).Debug("Memory cleanup performed")
	}
}

// GetStats returns resource manager statistics
func (rm *ResourceManager) GetStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_resources"] = len(rm.resources)

	resourceTypes := make(map[string]int)
	for _, resource := range rm.resources {
		resourceTypes[resource.Type()]++
	}
	stats["resource_types"] = resourceTypes

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	stats["memory_usage"] = map[string]interface{}{
		"alloc_bytes":     memStats.Alloc,
		"total_alloc":     memStats.TotalAlloc,
		"sys_bytes":       memStats.Sys,
		"num_gc":          memStats.NumGC,
		"gc_cpu_fraction": memStats.GCCPUFraction,
	}

	return stats
}
