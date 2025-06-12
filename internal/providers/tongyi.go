package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TongyiProvider 通义千问提供商
type TongyiProvider struct {
	config *ProviderConfig
	client *http.Client
	name   string
}

// NewTongyiProvider 创建通义千问提供商
func NewTongyiProvider(config *ProviderConfig) *TongyiProvider {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &TongyiProvider{
		config: config,
		client: client,
		name:   "tongyi",
	}
}

// GetName 获取提供商名称
func (p *TongyiProvider) GetName() string {
	return p.name
}

// GetModels 获取支持的模型列表
func (p *TongyiProvider) GetModels() []Model {
	return p.config.Models
}

// GetConfig 获取配置
func (p *TongyiProvider) GetConfig() *ProviderConfig {
	return p.config
}

// tongyiChatRequest 通义千问聊天请求格式
type tongyiChatRequest struct {
	Model      string            `json:"model"`
	Input      tongyiInput       `json:"input"`
	Parameters *tongyiParameters `json:"parameters,omitempty"`
}

type tongyiInput struct {
	Messages []Message `json:"messages"`
}

type tongyiParameters struct {
	Temperature       *float64 `json:"temperature,omitempty"`
	TopP              *float64 `json:"top_p,omitempty"`
	TopK              *int     `json:"top_k,omitempty"`
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	Stop              []string `json:"stop,omitempty"`
	IncrementalOutput bool     `json:"incremental_output,omitempty"`
}

// tongyiChatResponse 通义千问聊天响应格式
type tongyiChatResponse struct {
	StatusCode int          `json:"status_code"`
	RequestID  string       `json:"request_id"`
	Code       string       `json:"code"`
	Message    string       `json:"message"`
	Output     tongyiOutput `json:"output"`
	Usage      tongyiUsage  `json:"usage"`
}

type tongyiOutput struct {
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type tongyiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Chat 聊天补全
func (p *TongyiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 转换请求格式
	tongyiReq := &tongyiChatRequest{
		Model: req.Model,
		Input: tongyiInput{
			Messages: req.Messages,
		},
	}

	if req.Temperature != nil || req.TopP != nil || req.TopK != nil || req.MaxTokens != nil || len(req.Stop) > 0 {
		tongyiReq.Parameters = &tongyiParameters{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			TopK:        req.TopK,
			MaxTokens:   req.MaxTokens,
			Stop:        req.Stop,
		}
	}

	// 序列化请求
	reqBody, err := json.Marshal(tongyiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("X-DashScope-SSE", "disable")

	// 发送请求
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var tongyiResp tongyiChatResponse
	if err := json.Unmarshal(respBody, &tongyiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查API错误
	if tongyiResp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s - %s", tongyiResp.Code, tongyiResp.Message)
	}

	// 转换响应格式
	response := &ChatResponse{
		ID:       tongyiResp.RequestID,
		Object:   "chat.completion",
		Created:  time.Now().Unix(),
		Model:    req.Model,
		Provider: p.name,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: tongyiResp.Output.Text,
				},
				FinishReason: tongyiResp.Output.FinishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     tongyiResp.Usage.InputTokens,
			CompletionTokens: tongyiResp.Usage.OutputTokens,
			TotalTokens:      tongyiResp.Usage.TotalTokens,
		},
	}

	return response, nil
}

// ChatStream 流式聊天补全
func (p *TongyiProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamResponse, error) {
	// TODO: 实现流式聊天
	// 通义千问支持SSE流式响应，需要解析SSE格式
	responseChan := make(chan *ChatStreamResponse, 1)

	go func() {
		defer close(responseChan)

		// 这里应该实现SSE流式处理
		// 暂时返回错误
		responseChan <- &ChatStreamResponse{
			Error: fmt.Errorf("streaming not implemented yet for Tongyi provider"),
			Done:  true,
		}
	}()

	return responseChan, nil
}

// Embeddings 文本嵌入
func (p *TongyiProvider) Embeddings(ctx context.Context, req *EmbeddingsRequest) (*EmbeddingsResponse, error) {
	// TODO: 实现文本嵌入API
	return nil, fmt.Errorf("embeddings not implemented yet for Tongyi provider")
}

// HealthCheck 健康检查
func (p *TongyiProvider) HealthCheck(ctx context.Context) error {
	// 创建一个简单的测试请求
	testReq := &ChatRequest{
		Model: p.config.Models[0].Name,
		Messages: []Message{
			{
				Role:    "user",
				Content: "测试连接",
			},
		},
		MaxTokens: func() *int { i := 1; return &i }(),
	}

	// 发送测试请求
	_, err := p.Chat(ctx, testReq)
	return err
}
