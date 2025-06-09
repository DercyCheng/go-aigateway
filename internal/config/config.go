package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	GinMode        string
	TargetURL      string
	TargetKey      string
	GatewayKeys    []string
	LogLevel       string
	LogFormat      string
	RateLimit      int
	HealthCheck    bool
	AllowedOrigins []string // CORS allowed origins

	// Security Configuration
	Security SecurityConfig

	// Redis Configuration
	Redis RedisConfig

	// Service Discovery
	ServiceDiscovery ServiceDiscoveryConfig

	// Protocol Conversion
	ProtocolConversion ProtocolConversionConfig

	// RAM Authentication
	RAMAuth RAMAuthConfig

	// Cloud Integration
	CloudIntegration CloudIntegrationConfig

	// Auto Scaling
	AutoScaling AutoScalingConfig

	// Monitoring
	Monitoring MonitoringConfig

	// Local Model with Python
	LocalModel LocalModelConfig
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	EnableLocalAuth bool          // Use local authentication instead of RAM
	JWTSecret       string        // JWT secret for token generation
	TokenExpiration time.Duration // JWT token expiration
	RequireHTTPS    bool          // Force HTTPS in production
	APIKeyPrefix    string        // Prefix for API keys
	MaxAPIKeys      int           // Maximum number of API keys per user
}

type ServiceDiscoveryConfig struct {
	Enabled     bool
	Type        string // consul, etcd, kubernetes, nacos
	Endpoints   []string
	Namespace   string
	RefreshRate time.Duration
}

type RedisConfig struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type AutoScalingConfig struct {
	Enabled           bool
	MinReplicas       int
	MaxReplicas       int
	TargetCPU         float64
	TargetQPS         int
	ScaleUpCooldown   time.Duration
	ScaleDownCooldown time.Duration
}

type MonitoringConfig struct {
	Enabled          bool
	AlertsEnabled    bool
	MetricsRetention time.Duration
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
	Provider      string // aliyun, aws, azure, gcp (alias for CloudProvider)
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

// LocalModelConfig represents the configuration for local models using Python
type LocalModelConfig struct {
	Enabled       bool
	PythonPath    string
	ModelPath     string
	ServerHost    string
	ServerPort    int
	Port          string // String version of ServerPort for compatibility
	Timeout       time.Duration
	MaxTokens     int
	Temperature   float64
	Type          string // "chat", "completion", "embedding" (alias for ModelType)
	ModelType     string // "chat", "completion", "embedding"
	Size          string // "small", "medium", "large" (alias for ModelSize)
	ModelSize     string // "small", "medium", "large"
	RetryAttempts int
	RetryDelay    time.Duration
	LogRequests   bool
	LogResponses  bool

	// Third-party model support
	ThirdParty ThirdPartyModelConfig
}

// ThirdPartyModelConfig represents configuration for third-party AI models
type ThirdPartyModelConfig struct {
	Enabled      bool   // Enable third-party models
	Provider     string // Provider name: "dashscope", "openai", "anthropic"
	APIKey       string // API key for the third-party service
	BaseURL      string // Custom base URL for the API
	DefaultModel string // Default model to use for requests
}

func New() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		GinMode:   getEnv("GIN_MODE", "release"),
		TargetURL: getEnv("TARGET_URL", getEnv("TARGET_API_URL", "")),
		TargetKey: getEnv("TARGET_KEY", getEnv("TARGET_API_KEY", "")),
		GatewayKeys: func() []string {
			keys := getEnv("GATEWAY_API_KEYS", "")
			if keys == "" {
				return []string{}
			}
			return strings.Split(keys, ",")
		}(),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "json"),
		RateLimit:      getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		HealthCheck:    getEnvBool("HEALTH_CHECK_ENABLED", true),
		AllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),

		// Security Configuration
		Security: SecurityConfig{
			EnableLocalAuth: getEnvBool("ENABLE_LOCAL_AUTH", true),
			JWTSecret:       getEnv("JWT_SECRET", ""),
			TokenExpiration: getEnvDuration("TOKEN_EXPIRATION", 24*time.Hour),
			RequireHTTPS:    getEnvBool("REQUIRE_HTTPS", false),
			APIKeyPrefix:    getEnv("API_KEY_PREFIX", "gw-"),
			MaxAPIKeys:      getEnvInt("MAX_API_KEYS_PER_USER", 10),
		},

		Redis: RedisConfig{
			Enabled:  getEnvBool("REDIS_ENABLED", true),
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},

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
		}, CloudIntegration: CloudIntegrationConfig{
			Enabled:       getEnvBool("CLOUD_INTEGRATION_ENABLED", false),
			Provider:      getEnv("CLOUD_INTEGRATION_PROVIDER", getEnv("CLOUD_PROVIDER", "aws")),
			CloudProvider: getEnv("CLOUD_PROVIDER", "aws"),
			Region:        getEnv("CLOUD_INTEGRATION_REGION", getEnv("CLOUD_REGION", "us-west-2")),
			Credentials: CloudCredentials{
				AccessKeyID:     getEnv("CLOUD_ACCESS_KEY_ID", ""),
				AccessKeySecret: getEnv("CLOUD_ACCESS_KEY_SECRET", ""),
				SessionToken:    getEnv("CLOUD_SESSION_TOKEN", ""),
			},
			Services: strings.Split(getEnv("CLOUD_SERVICES", "ecs,rds,oss"), ","),
		},

		AutoScaling: AutoScalingConfig{
			Enabled:           getEnvBool("AUTO_SCALING_ENABLED", false),
			MinReplicas:       getEnvInt("AUTO_SCALING_MIN_REPLICAS", 1),
			MaxReplicas:       getEnvInt("AUTO_SCALING_MAX_REPLICAS", 10),
			TargetCPU:         getEnvFloat("AUTO_SCALING_TARGET_CPU", 70.0),
			TargetQPS:         getEnvInt("AUTO_SCALING_TARGET_QPS", 1000),
			ScaleUpCooldown:   getEnvDuration("AUTO_SCALING_UP_COOLDOWN", 3*time.Minute),
			ScaleDownCooldown: getEnvDuration("AUTO_SCALING_DOWN_COOLDOWN", 5*time.Minute),
		},
		Monitoring: MonitoringConfig{
			Enabled:          getEnvBool("MONITORING_ENABLED", true),
			AlertsEnabled:    getEnvBool("MONITORING_ALERTS_ENABLED", true),
			MetricsRetention: getEnvDuration("MONITORING_METRICS_RETENTION", 24*time.Hour),
		}, LocalModel: LocalModelConfig{
			Enabled:       getEnvBool("LOCAL_MODEL_ENABLED", false),
			PythonPath:    getEnv("PYTHON_PATH", "python"),
			ModelPath:     getEnv("MODEL_PATH", "./python/model"),
			ServerHost:    getEnv("LOCAL_MODEL_HOST", "localhost"),
			ServerPort:    getEnvInt("LOCAL_MODEL_PORT", 5000),
			Port:          getEnv("LOCAL_MODEL_PORT", "5000"),
			Timeout:       getEnvDuration("LOCAL_MODEL_TIMEOUT", 30*time.Second),
			MaxTokens:     getEnvInt("LOCAL_MODEL_MAX_TOKENS", 1024),
			Temperature:   getEnvFloat("LOCAL_MODEL_TEMPERATURE", 0.7),
			Type:          getEnv("LOCAL_MODEL_TYPE", "chat"),
			ModelType:     getEnv("LOCAL_MODEL_TYPE", "chat"),
			Size:          getEnv("LOCAL_MODEL_SIZE", "small"),
			ModelSize:     getEnv("LOCAL_MODEL_SIZE", "small"),
			RetryAttempts: getEnvInt("LOCAL_MODEL_RETRY_ATTEMPTS", 3),
			RetryDelay:    getEnvDuration("LOCAL_MODEL_RETRY_DELAY", 1*time.Second),
			LogRequests:   getEnvBool("LOCAL_MODEL_LOG_REQUESTS", true),
			LogResponses:  getEnvBool("LOCAL_MODEL_LOG_RESPONSES", true),

			// Third-party model configuration
			ThirdParty: ThirdPartyModelConfig{
				Enabled:      getEnvBool("THIRD_PARTY_MODEL_ENABLED", false),
				Provider:     getEnv("THIRD_PARTY_MODEL_PROVIDER", "dashscope"),
				APIKey:       getEnv("THIRD_PARTY_MODEL_API_KEY", getEnv("DASHSCOPE_API_KEY", "")),
				BaseURL:      getEnv("THIRD_PARTY_MODEL_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
				DefaultModel: getEnv("THIRD_PARTY_MODEL_DEFAULT", "qwen-turbo"),
			},
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
