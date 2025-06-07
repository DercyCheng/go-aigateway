package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigNew(t *testing.T) {
	// Setup test environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("GIN_MODE", "test")
	os.Setenv("TARGET_URL", "http://localhost:8000")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("GIN_MODE")
		os.Unsetenv("TARGET_URL")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := New()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "test", cfg.GinMode)
	assert.Equal(t, "http://localhost:8000", cfg.TargetURL)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestConfigDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg := New()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "release", cfg.GinMode)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 60, cfg.RateLimit)
	assert.True(t, cfg.HealthCheck)
}

func TestRedisConfig(t *testing.T) {
	os.Setenv("REDIS_ENABLED", "true")
	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_PASSWORD", "secret")
	os.Setenv("REDIS_DB", "1")
	defer func() {
		os.Unsetenv("REDIS_ENABLED")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_DB")
	}()

	cfg := New()

	assert.True(t, cfg.Redis.Enabled)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Equal(t, "secret", cfg.Redis.Password)
	assert.Equal(t, 1, cfg.Redis.DB)
}

func TestServiceDiscoveryConfig(t *testing.T) {
	os.Setenv("SERVICE_DISCOVERY_ENABLED", "true")
	os.Setenv("SERVICE_DISCOVERY_TYPE", "consul")
	os.Setenv("SERVICE_DISCOVERY_ENDPOINTS", "consul1:8500,consul2:8500")
	os.Setenv("SERVICE_DISCOVERY_REFRESH_RATE", "30s")
	defer func() {
		os.Unsetenv("SERVICE_DISCOVERY_ENABLED")
		os.Unsetenv("SERVICE_DISCOVERY_TYPE")
		os.Unsetenv("SERVICE_DISCOVERY_ENDPOINTS")
		os.Unsetenv("SERVICE_DISCOVERY_REFRESH_RATE")
	}()

	cfg := New()

	assert.True(t, cfg.ServiceDiscovery.Enabled)
	assert.Equal(t, "consul", cfg.ServiceDiscovery.Type)
	assert.Contains(t, cfg.ServiceDiscovery.Endpoints, "consul1:8500")
	assert.Contains(t, cfg.ServiceDiscovery.Endpoints, "consul2:8500")
	assert.Equal(t, 30*time.Second, cfg.ServiceDiscovery.RefreshRate)
}

func TestInvalidRefreshRate(t *testing.T) {
	os.Setenv("SERVICE_DISCOVERY_REFRESH_RATE", "invalid")
	defer os.Unsetenv("SERVICE_DISCOVERY_REFRESH_RATE")

	cfg := New()

	// Should fall back to default
	assert.Equal(t, 30*time.Second, cfg.ServiceDiscovery.RefreshRate)
}

func TestCloudIntegrationConfig(t *testing.T) {
	os.Setenv("CLOUD_INTEGRATION_ENABLED", "true")
	os.Setenv("CLOUD_INTEGRATION_PROVIDER", "aws")
	os.Setenv("CLOUD_INTEGRATION_REGION", "us-west-2")
	defer func() {
		os.Unsetenv("CLOUD_INTEGRATION_ENABLED")
		os.Unsetenv("CLOUD_INTEGRATION_PROVIDER")
		os.Unsetenv("CLOUD_INTEGRATION_REGION")
	}()

	cfg := New()
	assert.True(t, cfg.CloudIntegration.Enabled)
	assert.Equal(t, "aws", cfg.CloudIntegration.CloudProvider)
	assert.Equal(t, "us-west-2", cfg.CloudIntegration.Region)
}

func TestLocalModelConfig(t *testing.T) {
	os.Setenv("LOCAL_MODEL_ENABLED", "true")
	os.Setenv("LOCAL_MODEL_TYPE", "chat")
	os.Setenv("LOCAL_MODEL_SIZE", "medium")
	os.Setenv("LOCAL_MODEL_PORT", "8001")
	defer func() {
		os.Unsetenv("LOCAL_MODEL_ENABLED")
		os.Unsetenv("LOCAL_MODEL_TYPE")
		os.Unsetenv("LOCAL_MODEL_SIZE")
		os.Unsetenv("LOCAL_MODEL_PORT")
	}()

	cfg := New()
	assert.True(t, cfg.LocalModel.Enabled)
	assert.Equal(t, "chat", cfg.LocalModel.ModelType)
	assert.Equal(t, "medium", cfg.LocalModel.ModelSize)
	assert.Equal(t, "8001", cfg.LocalModel.Port)
}

func TestGatewayKeysConfig(t *testing.T) {
	os.Setenv("GATEWAY_API_KEYS", "key1,key2,key3")
	defer os.Unsetenv("GATEWAY_API_KEYS")

	cfg := New()

	assert.Len(t, cfg.GatewayKeys, 3)
	assert.Contains(t, cfg.GatewayKeys, "key1")
	assert.Contains(t, cfg.GatewayKeys, "key2")
	assert.Contains(t, cfg.GatewayKeys, "key3")
}

func TestEmptyGatewayKeys(t *testing.T) {
	os.Setenv("GATEWAY_API_KEYS", "")
	defer os.Unsetenv("GATEWAY_API_KEYS")

	cfg := New()

	assert.Empty(t, cfg.GatewayKeys)
}
