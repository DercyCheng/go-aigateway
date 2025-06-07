package resources

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock resource for testing
type mockResource struct {
	id      string
	resType string
	healthy bool
	closed  bool
	mutex   sync.RWMutex
}

func (m *mockResource) ID() string {
	return m.id
}

func (m *mockResource) Type() string {
	return m.resType
}

func (m *mockResource) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closed = true
	return nil
}

func (m *mockResource) HealthCheck() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.healthy {
		return fmt.Errorf("resource unhealthy")
	}
	return nil
}

func (m *mockResource) IsClosed() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.closed
}

func (m *mockResource) SetHealthy(healthy bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.healthy = healthy
}

func TestNewResourceManager(t *testing.T) {
	config := &ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	}

	manager := NewResourceManager(config)
	require.NotNil(t, manager)
	assert.NotNil(t, manager.resources)
}

func TestResourceRegistration(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	resource := &mockResource{
		id:      "test-resource-1",
		resType: "test",
		healthy: true,
	}

	// Test successful registration
	err := manager.Register(resource)
	assert.NoError(t, err)

	// Test duplicate registration
	err = manager.Register(resource)
	assert.Error(t, err)
}

func TestResourceUnregistration(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	resource := &mockResource{
		id:      "test-resource-1",
		resType: "test",
		healthy: true,
	}

	// Register first
	err := manager.Register(resource)
	require.NoError(t, err)

	// Test successful unregistration
	err = manager.Unregister("test-resource-1")
	assert.NoError(t, err)
	assert.True(t, resource.IsClosed())

	// Test unregistering non-existent resource
	err = manager.Unregister("non-existent")
	assert.Error(t, err)
}

func TestResourceRetrieval(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	resource := &mockResource{
		id:      "test-resource-1",
		resType: "test",
		healthy: true,
	}

	err := manager.Register(resource)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := manager.Get("test-resource-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, resource.ID(), retrieved.ID())

	// Test retrieving non-existent resource
	retrieved, err = manager.Get("non-existent")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestResourceList(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	resource1 := &mockResource{id: "resource-1", resType: "test", healthy: true}
	resource2 := &mockResource{id: "resource-2", resType: "test", healthy: false}

	err := manager.Register(resource1)
	require.NoError(t, err)
	err = manager.Register(resource2)
	require.NoError(t, err)

	resources := manager.List()
	assert.Len(t, resources, 2)
	assert.Contains(t, resources, "resource-1")
	assert.Contains(t, resources, "resource-2")
}

func TestResourceCleanup(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     100 * time.Millisecond,
		HealthCheckRate: 50 * time.Millisecond,
		CleanupTimeout:  1 * time.Second,
	})

	resource := &mockResource{id: "test", resType: "test", healthy: true}

	err := manager.Register(resource)
	require.NoError(t, err)

	// Test CloseAll
	err = manager.CloseAll()
	assert.NoError(t, err)
	assert.True(t, resource.IsClosed())

	// Verify resources map is empty
	resources := manager.List()
	assert.Empty(t, resources)
}

func TestResourceShutdown(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  1 * time.Second,
	})

	resource := &mockResource{id: "test", resType: "test", healthy: true}
	err := manager.Register(resource)
	require.NoError(t, err)

	// Test shutdown with timeout
	err = manager.Shutdown(5 * time.Second)
	assert.NoError(t, err)
	assert.True(t, resource.IsClosed())
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	const numGoroutines = 10
	const resourcesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent resource registration
	for i := 0; i < numGoroutines; i++ {
		go func(base int) {
			defer wg.Done()
			for j := 0; j < resourcesPerGoroutine; j++ {
				resource := &mockResource{
					id:      fmt.Sprintf("resource-%d-%d", base, j),
					resType: "test",
					healthy: true,
				}
				err := manager.Register(resource)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	resources := manager.List()
	assert.Equal(t, numGoroutines*resourcesPerGoroutine, len(resources))

	// Cleanup all resources
	err := manager.CloseAll()
	assert.NoError(t, err)

	resources = manager.List()
	assert.Empty(t, resources)
}

func BenchmarkResourceRegistration(b *testing.B) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resource := &mockResource{
			id:      fmt.Sprintf("resource-%d", i),
			resType: "test",
			healthy: true,
		}
		manager.Register(resource)
	}
}

func BenchmarkResourceRetrieval(b *testing.B) {
	manager := NewResourceManager(&ResourceConfig{
		MaxIdleTime:     5 * time.Minute,
		HealthCheckRate: 30 * time.Second,
		CleanupTimeout:  10 * time.Second,
	})

	// Pre-populate with resources
	for i := 0; i < 1000; i++ {
		resource := &mockResource{
			id:      fmt.Sprintf("resource-%d", i),
			resType: "test",
			healthy: true,
		}
		manager.Register(resource)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Get(fmt.Sprintf("resource-%d", i%1000))
	}
}
