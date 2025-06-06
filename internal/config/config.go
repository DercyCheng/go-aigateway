package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	GinMode     string
	TargetURL   string
	TargetKey   string
	GatewayKeys []string
	LogLevel    string
	LogFormat   string
	RateLimit   int
	HealthCheck bool

	// Service Discovery
	ServiceDiscovery ServiceDiscoveryConfig

	// Protocol Conversion
	ProtocolConversion ProtocolConversionConfig

	// RAM Authentication
	RAMAuth RAMAuthConfig

	// Cloud Integration
	CloudIntegration CloudIntegrationConfig
}

type ServiceDiscoveryConfig struct {
	Enabled     bool
	Type        string // consul, etcd, kubernetes, nacos
	Endpoints   []string
	Namespace   string
	RefreshRate time.Duration
}

type ProtocolConversionConfig struct {
	Enabled     bool
	HTTPSToRPC  bool
	GRPCSupport bool
	Protocols   []string
}

type RAMAuthConfig struct {
	Enabled         bool
	AccessKeyID     string
	AccessKeySecret string
	Region          string
	RoleArn         string
	PolicyDocument  string
	CacheExpiration time.Duration
}

type CloudIntegrationConfig struct {
	Enabled       bool
	CloudProvider string // aliyun, aws, azure, gcp
	Region        string
	Credentials   CloudCredentials
	Services      []string
}

type CloudCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
	SessionToken    string
}

func New() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "release"),
		TargetURL:   getEnv("TARGET_API_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		TargetKey:   getEnv("TARGET_API_KEY", ""),
		GatewayKeys: strings.Split(getEnv("GATEWAY_API_KEYS", ""), ","),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogFormat:   getEnv("LOG_FORMAT", "json"),
		RateLimit:   getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		HealthCheck: getEnvBool("HEALTH_CHECK_ENABLED", true),

		ServiceDiscovery: ServiceDiscoveryConfig{
			Enabled:     getEnvBool("SERVICE_DISCOVERY_ENABLED", false),
			Type:        getEnv("SERVICE_DISCOVERY_TYPE", "consul"),
			Endpoints:   strings.Split(getEnv("SERVICE_DISCOVERY_ENDPOINTS", ""), ","),
			Namespace:   getEnv("SERVICE_DISCOVERY_NAMESPACE", "default"),
			RefreshRate: getEnvDuration("SERVICE_DISCOVERY_REFRESH_RATE", 30*time.Second),
		},

		ProtocolConversion: ProtocolConversionConfig{
			Enabled:     getEnvBool("PROTOCOL_CONVERSION_ENABLED", false),
			HTTPSToRPC:  getEnvBool("HTTPS_TO_RPC_ENABLED", false),
			GRPCSupport: getEnvBool("GRPC_SUPPORT_ENABLED", false),
			Protocols:   strings.Split(getEnv("SUPPORTED_PROTOCOLS", "http,https"), ","),
		},

		RAMAuth: RAMAuthConfig{
			Enabled:         getEnvBool("RAM_AUTH_ENABLED", false),
			AccessKeyID:     getEnv("RAM_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("RAM_ACCESS_KEY_SECRET", ""),
			Region:          getEnv("RAM_REGION", "cn-hangzhou"),
			RoleArn:         getEnv("RAM_ROLE_ARN", ""),
			PolicyDocument:  getEnv("RAM_POLICY_DOCUMENT", ""),
			CacheExpiration: getEnvDuration("RAM_CACHE_EXPIRATION", 15*time.Minute),
		},

		CloudIntegration: CloudIntegrationConfig{
			Enabled:       getEnvBool("CLOUD_INTEGRATION_ENABLED", false),
			CloudProvider: getEnv("CLOUD_PROVIDER", "aliyun"),
			Region:        getEnv("CLOUD_REGION", "cn-hangzhou"),
			Credentials: CloudCredentials{
				AccessKeyID:     getEnv("CLOUD_ACCESS_KEY_ID", ""),
				AccessKeySecret: getEnv("CLOUD_ACCESS_KEY_SECRET", ""),
				SessionToken:    getEnv("CLOUD_SESSION_TOKEN", ""),
			},
			Services: strings.Split(getEnv("CLOUD_SERVICES", "ecs,rds,oss"), ","),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
