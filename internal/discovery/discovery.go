package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ServiceInstance struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Protocol string            `json:"protocol"`
	Tags     []string          `json:"tags"`
	Meta     map[string]string `json:"meta"`
	Health   string            `json:"health"`
}

type ServiceDiscovery interface {
	Register(instance *ServiceInstance) error
	Deregister(instanceID string) error
	Discover(serviceName string) ([]*ServiceInstance, error)
	Watch(serviceName string, callback func([]*ServiceInstance)) error
	Close() error
}

type Manager struct {
	config    *config.ServiceDiscoveryConfig
	discovery ServiceDiscovery
	services  map[string][]*ServiceInstance
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewManager(cfg *config.ServiceDiscoveryConfig) (*Manager, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:   cfg,
		services: make(map[string][]*ServiceInstance),
		ctx:      ctx,
		cancel:   cancel,
	}

	var err error
	switch cfg.Type {
	case "consul":
		manager.discovery, err = NewConsulDiscovery(cfg)
	case "etcd":
		manager.discovery, err = NewEtcdDiscovery(cfg)
	case "kubernetes":
		manager.discovery, err = NewKubernetesDiscovery(cfg)
	case "nacos":
		manager.discovery, err = NewNacosDiscovery(cfg)
	default:
		return nil, fmt.Errorf("unsupported service discovery type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create service discovery: %w", err)
	}

	// Start background refresh
	go manager.backgroundRefresh()

	return manager, nil
}

func (m *Manager) GetServices(serviceName string) []*ServiceInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := m.services[serviceName]
	result := make([]*ServiceInstance, len(services))
	copy(result, services)
	return result
}

func (m *Manager) RegisterService(instance *ServiceInstance) error {
	if m.discovery == nil {
		return fmt.Errorf("service discovery not enabled")
	}

	return m.discovery.Register(instance)
}

func (m *Manager) DeregisterService(instanceID string) error {
	if m.discovery == nil {
		return fmt.Errorf("service discovery not enabled")
	}

	return m.discovery.Deregister(instanceID)
}

func (m *Manager) backgroundRefresh() {
	ticker := time.NewTicker(m.config.RefreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.refreshServices()
		}
	}
}

func (m *Manager) refreshServices() {
	// This would be implemented based on your service names
	// For now, we'll just log that refresh is happening
	logrus.Debug("Refreshing service discovery cache")
}

func (m *Manager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}

	if m.discovery != nil {
		return m.discovery.Close()
	}

	return nil
}

// Consul implementation
type ConsulDiscovery struct {
	config *config.ServiceDiscoveryConfig
}

func NewConsulDiscovery(cfg *config.ServiceDiscoveryConfig) (*ConsulDiscovery, error) {
	return &ConsulDiscovery{config: cfg}, nil
}

func (c *ConsulDiscovery) Register(instance *ServiceInstance) error {
	logrus.WithField("instance", instance.ID).Info("Registering service with Consul")

	// Build Consul service registration payload
	registration := map[string]interface{}{
		"ID":      instance.ID,
		"Name":    instance.Name,
		"Address": instance.Address,
		"Port":    instance.Port,
		"Tags":    instance.Tags,
		"Meta":    instance.Meta,
		"Check": map[string]interface{}{
			"HTTP":                           fmt.Sprintf("%s://%s:%d/health", instance.Protocol, instance.Address, instance.Port),
			"Interval":                       "10s",
			"Timeout":                        "3s",
			"DeregisterCriticalServiceAfter": "30s",
		},
	}

	// Make HTTP request to Consul API
	jsonData, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %w", err)
	}

	for _, endpoint := range c.config.Endpoints {
		url := fmt.Sprintf("%s/v1/agent/service/register", endpoint)
		req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonData))
		if err != nil {
			logrus.WithError(err).Warnf("Failed to create request for Consul endpoint %s", endpoint)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to register with Consul endpoint %s", endpoint)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logrus.WithField("instance", instance.ID).Info("Successfully registered service with Consul")
			return nil
		}
	}

	return fmt.Errorf("failed to register service with any Consul endpoint")
}

func (c *ConsulDiscovery) Deregister(instanceID string) error {
	logrus.WithField("instance", instanceID).Info("Deregistering service from Consul")

	for _, endpoint := range c.config.Endpoints {
		url := fmt.Sprintf("%s/v1/agent/service/deregister/%s", endpoint, instanceID)
		req, err := http.NewRequest("PUT", url, nil)
		if err != nil {
			continue
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to deregister from Consul endpoint %s", endpoint)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logrus.WithField("instance", instanceID).Info("Successfully deregistered service from Consul")
			return nil
		}
	}

	return fmt.Errorf("failed to deregister service from any Consul endpoint")
}

func (c *ConsulDiscovery) Discover(serviceName string) ([]*ServiceInstance, error) {
	logrus.WithField("service", serviceName).Info("Discovering services from Consul")

	for _, endpoint := range c.config.Endpoints {
		url := fmt.Sprintf("%s/v1/health/service/%s?passing=true", endpoint, serviceName)
		resp, err := http.Get(url)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to discover services from Consul endpoint %s", endpoint)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		var consulServices []struct {
			Service struct {
				ID      string            `json:"ID"`
				Service string            `json:"Service"`
				Address string            `json:"Address"`
				Port    int               `json:"Port"`
				Tags    []string          `json:"Tags"`
				Meta    map[string]string `json:"Meta"`
			} `json:"Service"`
			Checks []struct {
				Status string `json:"Status"`
			} `json:"Checks"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&consulServices); err != nil {
			continue
		}

		var instances []*ServiceInstance
		for _, cs := range consulServices {
			health := "unknown"
			for _, check := range cs.Checks {
				if check.Status == "passing" {
					health = "healthy"
					break
				} else if check.Status == "critical" {
					health = "unhealthy"
				}
			}

			instances = append(instances, &ServiceInstance{
				ID:       cs.Service.ID,
				Name:     cs.Service.Service,
				Address:  cs.Service.Address,
				Port:     cs.Service.Port,
				Protocol: "http", // Default to http
				Tags:     cs.Service.Tags,
				Meta:     cs.Service.Meta,
				Health:   health,
			})
		}

		return instances, nil
	}

	return nil, fmt.Errorf("failed to discover services from any Consul endpoint")
}

func (c *ConsulDiscovery) Watch(serviceName string, callback func([]*ServiceInstance)) error {
	logrus.WithField("service", serviceName).Info("Watching service changes in Consul")

	go func() {
		ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
		defer ticker.Stop()

		var lastInstances []*ServiceInstance

		for {
			select {
			case <-ticker.C:
				instances, err := c.Discover(serviceName)
				if err != nil {
					logrus.WithError(err).Error("Failed to discover services during watch")
					continue
				}

				// Check if instances have changed
				if !instancesEqual(lastInstances, instances) {
					lastInstances = instances
					callback(instances)
				}
			}
		}
	}()

	return nil
}

func (c *ConsulDiscovery) Close() error {
	return nil
}

// Etcd implementation
type EtcdDiscovery struct {
	config *config.ServiceDiscoveryConfig
}

func NewEtcdDiscovery(cfg *config.ServiceDiscoveryConfig) (*EtcdDiscovery, error) {
	return &EtcdDiscovery{config: cfg}, nil
}

func (e *EtcdDiscovery) Register(instance *ServiceInstance) error {
	logrus.WithField("instance", instance.ID).Info("Registering service with etcd")
	return nil
}

func (e *EtcdDiscovery) Deregister(instanceID string) error {
	logrus.WithField("instance", instanceID).Info("Deregistering service from etcd")
	return nil
}

func (e *EtcdDiscovery) Discover(serviceName string) ([]*ServiceInstance, error) {
	logrus.WithField("service", serviceName).Info("Discovering services from etcd")
	return nil, nil
}

func (e *EtcdDiscovery) Watch(serviceName string, callback func([]*ServiceInstance)) error {
	logrus.WithField("service", serviceName).Info("Watching service changes in etcd")
	return nil
}

func (e *EtcdDiscovery) Close() error {
	return nil
}

// Kubernetes implementation
type KubernetesDiscovery struct {
	config *config.ServiceDiscoveryConfig
}

func NewKubernetesDiscovery(cfg *config.ServiceDiscoveryConfig) (*KubernetesDiscovery, error) {
	return &KubernetesDiscovery{config: cfg}, nil
}

func (k *KubernetesDiscovery) Register(instance *ServiceInstance) error {
	logrus.WithField("instance", instance.ID).Info("Registering service with Kubernetes")
	return nil
}

func (k *KubernetesDiscovery) Deregister(instanceID string) error {
	logrus.WithField("instance", instanceID).Info("Deregistering service from Kubernetes")
	return nil
}

func (k *KubernetesDiscovery) Discover(serviceName string) ([]*ServiceInstance, error) {
	logrus.WithField("service", serviceName).Info("Discovering services from Kubernetes")
	return nil, nil
}

func (k *KubernetesDiscovery) Watch(serviceName string, callback func([]*ServiceInstance)) error {
	logrus.WithField("service", serviceName).Info("Watching service changes in Kubernetes")
	return nil
}

func (k *KubernetesDiscovery) Close() error {
	return nil
}

// Nacos implementation
type NacosDiscovery struct {
	config *config.ServiceDiscoveryConfig
}

func NewNacosDiscovery(cfg *config.ServiceDiscoveryConfig) (*NacosDiscovery, error) {
	return &NacosDiscovery{config: cfg}, nil
}

func (n *NacosDiscovery) Register(instance *ServiceInstance) error {
	logrus.WithField("instance", instance.ID).Info("Registering service with Nacos")
	return nil
}

func (n *NacosDiscovery) Deregister(instanceID string) error {
	logrus.WithField("instance", instanceID).Info("Deregistering service from Nacos")
	return nil
}

func (n *NacosDiscovery) Discover(serviceName string) ([]*ServiceInstance, error) {
	logrus.WithField("service", serviceName).Info("Discovering services from Nacos")
	return nil, nil
}

func (n *NacosDiscovery) Watch(serviceName string, callback func([]*ServiceInstance)) error {
	logrus.WithField("service", serviceName).Info("Watching service changes in Nacos")
	return nil
}

func (n *NacosDiscovery) Close() error {
	return nil
}

// Helper function to compare service instances
func instancesEqual(a, b []*ServiceInstance) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].ID != b[i].ID ||
			a[i].Name != b[i].Name ||
			a[i].Address != b[i].Address ||
			a[i].Port != b[i].Port ||
			a[i].Health != b[i].Health {
			return false
		}
	}

	return true
}
