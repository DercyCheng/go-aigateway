package providers

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config AI服务提供商总配置
type Config struct {
	Tongyi   *ProviderConfig `yaml:"tongyi"`
	OpenAI   *ProviderConfig `yaml:"openai"`
	Wenxin   *ProviderConfig `yaml:"wenxin"`
	Zhipu    *ProviderConfig `yaml:"zhipu"`
	Hunyuan  *ProviderConfig `yaml:"hunyuan"`
	Moonshot *ProviderConfig `yaml:"moonshot"`
	Global   *GlobalConfig   `yaml:"global"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	DefaultTimeout      time.Duration       `yaml:"default_timeout"`
	DefaultRetryCount   int                 `yaml:"default_retry_count"`
	DefaultRetryDelay   time.Duration       `yaml:"default_retry_delay"`
	MaxIdleConns        int                 `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int                 `yaml:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration       `yaml:"idle_conn_timeout"`
	MaxRequestSize      string              `yaml:"max_request_size"`
	MaxResponseSize     string              `yaml:"max_response_size"`
	CacheEnabled        bool                `yaml:"cache_enabled"`
	CacheTTL            time.Duration       `yaml:"cache_ttl"`
	CacheMaxSize        int                 `yaml:"cache_max_size"`
	MetricsEnabled      bool                `yaml:"metrics_enabled"`
	TracingEnabled      bool                `yaml:"tracing_enabled"`
	LoadBalanceStrategy LoadBalanceStrategy `yaml:"load_balance_strategy"`
	HealthCheckEnabled  bool                `yaml:"health_check_enabled"`
	HealthCheckInterval time.Duration       `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration       `yaml:"health_check_timeout"`
	RateLimitEnabled    bool                `yaml:"rate_limit_enabled"`
	AuthRequired        bool                `yaml:"auth_required"`
	TLSVerify           bool                `yaml:"tls_verify"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 替换环境变量
	configContent := expandEnvVars(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(configContent), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用默认值
	applyDefaults(&config)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// expandEnvVars 展开环境变量
func expandEnvVars(content string) string {
	return os.ExpandEnv(content)
}

// applyDefaults 应用默认配置
func applyDefaults(config *Config) {
	// 全局默认值
	if config.Global == nil {
		config.Global = &GlobalConfig{}
	}

	global := config.Global
	if global.DefaultTimeout == 0 {
		global.DefaultTimeout = 30 * time.Second
	}
	if global.DefaultRetryCount == 0 {
		global.DefaultRetryCount = 3
	}
	if global.DefaultRetryDelay == 0 {
		global.DefaultRetryDelay = 1 * time.Second
	}
	if global.MaxIdleConns == 0 {
		global.MaxIdleConns = 100
	}
	if global.MaxIdleConnsPerHost == 0 {
		global.MaxIdleConnsPerHost = 10
	}
	if global.IdleConnTimeout == 0 {
		global.IdleConnTimeout = 90 * time.Second
	}
	if global.LoadBalanceStrategy == "" {
		global.LoadBalanceStrategy = LoadBalanceRoundRobin
	}
	if global.HealthCheckInterval == 0 {
		global.HealthCheckInterval = 30 * time.Second
	}
	if global.HealthCheckTimeout == 0 {
		global.HealthCheckTimeout = 5 * time.Second
	}

	// 为每个提供商应用默认值
	providers := []*ProviderConfig{
		config.Tongyi, config.OpenAI, config.Wenxin,
		config.Zhipu, config.Hunyuan, config.Moonshot,
	}

	for _, provider := range providers {
		if provider != nil {
			applyProviderDefaults(provider, global)
		}
	}
}

// applyProviderDefaults 为提供商应用默认值
func applyProviderDefaults(provider *ProviderConfig, global *GlobalConfig) {
	if provider.Timeout == 0 {
		provider.Timeout = global.DefaultTimeout
	}
	if provider.RetryCount == 0 {
		provider.RetryCount = global.DefaultRetryCount
	}
	if provider.RetryDelay == 0 {
		provider.RetryDelay = global.DefaultRetryDelay
	}
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 检查至少有一个提供商启用
	hasEnabledProvider := false
	providers := map[string]*ProviderConfig{
		"tongyi":   config.Tongyi,
		"openai":   config.OpenAI,
		"wenxin":   config.Wenxin,
		"zhipu":    config.Zhipu,
		"hunyuan":  config.Hunyuan,
		"moonshot": config.Moonshot,
	}

	for name, provider := range providers {
		if provider != nil && provider.Enabled {
			if err := validateProviderConfig(name, provider); err != nil {
				return err
			}
			hasEnabledProvider = true
		}
	}

	if !hasEnabledProvider {
		return fmt.Errorf("no providers enabled")
	}

	return nil
}

// validateProviderConfig 验证提供商配置
func validateProviderConfig(name string, config *ProviderConfig) error {
	if config.BaseURL == "" {
		return fmt.Errorf("provider %s: base_url is required", name)
	}

	if config.APIKey == "" {
		return fmt.Errorf("provider %s: api_key is required", name)
	}

	if len(config.Models) == 0 {
		return fmt.Errorf("provider %s: at least one model must be configured", name)
	}

	// 验证模型配置
	for i, model := range config.Models {
		if model.Name == "" {
			return fmt.Errorf("provider %s: model[%d].name is required", name, i)
		}
		if model.MaxTokens <= 0 {
			return fmt.Errorf("provider %s: model[%d].max_tokens must be positive", name, i)
		}
		if model.RateLimit <= 0 {
			return fmt.Errorf("provider %s: model[%d].rate_limit must be positive", name, i)
		}
	}

	return nil
}

// CreateProviderFromConfig 从配置创建提供商
func CreateProviderFromConfig(providerType ProviderType, config *ProviderConfig) (Provider, error) {
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("provider %s is not enabled", providerType)
	}

	switch providerType {
	case ProviderTypeTongyi:
		return NewTongyiProvider(config), nil
	case ProviderTypeOpenAI:
		// TODO: 实现OpenAI提供商
		return nil, fmt.Errorf("OpenAI provider not implemented yet")
	case ProviderTypeWenxin:
		// TODO: 实现百度文心一言提供商
		return nil, fmt.Errorf("Wenxin provider not implemented yet")
	case ProviderTypeZhipu:
		// TODO: 实现智谱AI提供商
		return nil, fmt.Errorf("Zhipu provider not implemented yet")
	case ProviderTypeHunyuan:
		// TODO: 实现腾讯混元提供商
		return nil, fmt.Errorf("Hunyuan provider not implemented yet")
	case ProviderTypeMoonshot:
		// TODO: 实现月之暗面提供商
		return nil, fmt.Errorf("Moonshot provider not implemented yet")
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// LoadSecretFromFile 从文件加载密钥
func LoadSecretFromFile(filename string) (string, error) {
	if filename == "" {
		return "", nil
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %s: %w", filename, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// LoadSecretsFromEnvFiles 从环境变量指定的文件加载密钥
func LoadSecretsFromEnvFiles(config *Config) error {
	// 处理通义千问
	if config.Tongyi != nil && config.Tongyi.Enabled {
		if keyFile := os.Getenv("TONGYI_API_KEY_FILE"); keyFile != "" {
			key, err := LoadSecretFromFile(keyFile)
			if err != nil {
				return err
			}
			config.Tongyi.APIKey = key
		}
	}

	// 处理OpenAI
	if config.OpenAI != nil && config.OpenAI.Enabled {
		if keyFile := os.Getenv("OPENAI_API_KEY_FILE"); keyFile != "" {
			key, err := LoadSecretFromFile(keyFile)
			if err != nil {
				return err
			}
			config.OpenAI.APIKey = key
		}
	}

	// 处理其他提供商...
	// TODO: 为其他提供商添加类似的逻辑

	return nil
}
