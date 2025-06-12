package providers

import (
	"context"
	"time"
)

// Provider 定义AI服务提供商接口
type Provider interface {
	// GetName 获取提供商名称
	GetName() string

	// GetModels 获取支持的模型列表
	GetModels() []Model

	// Chat 聊天补全
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 流式聊天补全
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamResponse, error)

	// Embeddings 文本嵌入
	Embeddings(ctx context.Context, req *EmbeddingsRequest) (*EmbeddingsResponse, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// GetConfig 获取配置
	GetConfig() *ProviderConfig
}

// Model 模型信息
type Model struct {
	Name              string `json:"name" yaml:"name"`
	MaxTokens         int    `json:"max_tokens" yaml:"max_tokens"`
	SupportsStreaming bool   `json:"supports_streaming" yaml:"supports_streaming"`
	RateLimit         int    `json:"rate_limit" yaml:"rate_limit"`
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	BaseURL    string        `json:"base_url" yaml:"base_url"`
	APIKey     string        `json:"api_key" yaml:"api_key"`
	Models     []Model       `json:"models" yaml:"models"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	RetryCount int           `json:"retry_count" yaml:"retry_count"`
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Temperature *float64   `json:"temperature,omitempty"`
	MaxTokens   *int       `json:"max_tokens,omitempty"`
	Stream      bool       `json:"stream,omitempty"`
	Stop        []string   `json:"stop,omitempty"`
	TopP        *float64   `json:"top_p,omitempty"`
	TopK        *int       `json:"top_k,omitempty"`
	User        string     `json:"user,omitempty"`
	Functions   []Function `json:"functions,omitempty"`
	Tools       []Tool     `json:"tools,omitempty"`
}

// Message 消息
type Message struct {
	Role       string `json:"role"` // system, user, assistant, function
	Content    string `json:"content"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Function 函数定义
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Tool 工具定义
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID       string   `json:"id"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Model    string   `json:"model"`
	Choices  []Choice `json:"choices"`
	Usage    Usage    `json:"usage"`
	Provider string   `json:"provider"`
}

// Choice 选择
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	FinishReason string   `json:"finish_reason"`
	Delta        *Message `json:"delta,omitempty"`
}

// Usage 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatStreamResponse 流式聊天响应
type ChatStreamResponse struct {
	ID       string   `json:"id"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Model    string   `json:"model"`
	Choices  []Choice `json:"choices"`
	Provider string   `json:"provider"`
	Done     bool     `json:"done"`
	Error    error    `json:"error,omitempty"`
}

// EmbeddingsRequest 嵌入请求
type EmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	User  string   `json:"user,omitempty"`
}

// EmbeddingsResponse 嵌入响应
type EmbeddingsResponse struct {
	Object   string      `json:"object"`
	Model    string      `json:"model"`
	Data     []Embedding `json:"data"`
	Usage    Usage       `json:"usage"`
	Provider string      `json:"provider"`
}

// Embedding 嵌入向量
type Embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ProviderType 提供商类型
type ProviderType string

const (
	ProviderTypeTongyi   ProviderType = "tongyi"
	ProviderTypeOpenAI   ProviderType = "openai"
	ProviderTypeWenxin   ProviderType = "wenxin"
	ProviderTypeZhipu    ProviderType = "zhipu"
	ProviderTypeHunyuan  ProviderType = "hunyuan"
	ProviderTypeMoonshot ProviderType = "moonshot"
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin       LoadBalanceStrategy = "round_robin"
	LoadBalanceRandom           LoadBalanceStrategy = "random"
	LoadBalanceLeastConnections LoadBalanceStrategy = "least_connections"
	LoadBalanceWeighted         LoadBalanceStrategy = "weighted"
)

// ProviderStatus 提供商状态
type ProviderStatus string

const (
	ProviderStatusHealthy   ProviderStatus = "healthy"
	ProviderStatusUnhealthy ProviderStatus = "unhealthy"
	ProviderStatusDisabled  ProviderStatus = "disabled"
)

// ProviderMetrics 提供商指标
type ProviderMetrics struct {
	RequestCount    int64          `json:"request_count"`
	ErrorCount      int64          `json:"error_count"`
	AverageLatency  time.Duration  `json:"average_latency"`
	LastRequestTime time.Time      `json:"last_request_time"`
	Status          ProviderStatus `json:"status"`
}
