package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go-aigateway/internal/providers"

	"github.com/gin-gonic/gin"
)

// AIHandler AI服务处理器
type AIHandler struct {
	manager *providers.Manager
}

// NewAIHandler 创建AI服务处理器
func NewAIHandler(manager *providers.Manager) *AIHandler {
	return &AIHandler{
		manager: manager,
	}
}

// ChatCompletions 聊天补全接口
// @Summary 聊天补全
// @Description 发送聊天消息并获取AI回复
// @Tags AI
// @Accept json
// @Produce json
// @Param request body providers.ChatRequest true "聊天请求"
// @Success 200 {object} providers.ChatResponse "聊天响应"
// @Failure 400 {object} map[string]interface{} "请求错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /v1/chat/completions [post]
func (h *AIHandler) ChatCompletions(c *gin.Context) {
	var req providers.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format",
				"details": err.Error(),
			},
		})
		return
	}

	// 验证请求
	if err := h.validateChatRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request parameters",
				"details": err.Error(),
			},
		})
		return
	}

	// 处理流式响应
	if req.Stream {
		h.handleStreamingChat(c, &req)
		return
	}

	// 非流式响应
	response, err := h.manager.Chat(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to process chat request",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// handleStreamingChat 处理流式聊天
func (h *AIHandler) handleStreamingChat(c *gin.Context, req *providers.ChatRequest) {
	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// 获取流式响应通道
	responseChan, err := h.manager.ChatStream(c.Request.Context(), req)
	if err != nil {
		c.SSEvent("error", gin.H{
			"message": "Failed to start streaming",
			"details": err.Error(),
		})
		return
	}

	// 发送流式数据
	c.Stream(func(w io.Writer) bool {
		select {
		case response, ok := <-responseChan:
			if !ok {
				// 通道关闭，发送结束标记
				c.SSEvent("data", "[DONE]")
				return false
			}

			if response.Error != nil {
				c.SSEvent("error", gin.H{
					"message": "Streaming error",
					"details": response.Error.Error(),
				})
				return false
			}

			if response.Done {
				c.SSEvent("data", "[DONE]")
				return false
			}

			// 发送数据
			data, _ := json.Marshal(response)
			c.SSEvent("data", string(data))
			return true

		case <-c.Request.Context().Done():
			// 客户端断开连接
			return false
		}
	})
}

// validateChatRequest 验证聊天请求
func (h *AIHandler) validateChatRequest(req *providers.ChatRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages is required and cannot be empty")
	}

	// 验证消息格式
	for i, msg := range req.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message[%d].role is required", i)
		}
		if msg.Role != "system" && msg.Role != "user" && msg.Role != "assistant" && msg.Role != "function" {
			return fmt.Errorf("message[%d].role must be one of: system, user, assistant, function", i)
		}
		if msg.Content == "" && msg.Role != "function" {
			return fmt.Errorf("message[%d].content is required for role %s", i, msg.Role)
		}
	}

	// 验证参数范围
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if req.TopP != nil && (*req.TopP < 0 || *req.TopP > 1) {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	return nil
}

// GetModels 获取可用模型列表
// @Summary 获取模型列表
// @Description 获取所有可用的AI模型列表
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{} "模型列表"
// @Router /v1/models [get]
func (h *AIHandler) GetModels(c *gin.Context) {
	models := make(map[string]interface{})

	// 获取所有健康的提供商
	providers := h.manager.GetHealthyProviders()

	for _, provider := range providers {
		providerName := provider.GetName()
		providerModels := provider.GetModels()

		modelList := make([]map[string]interface{}, 0, len(providerModels))
		for _, model := range providerModels {
			modelList = append(modelList, map[string]interface{}{
				"id":                 model.Name,
				"object":             "model",
				"owned_by":           providerName,
				"max_tokens":         model.MaxTokens,
				"supports_streaming": model.SupportsStreaming,
				"rate_limit":         model.RateLimit,
			})
		}

		models[providerName] = map[string]interface{}{
			"provider": providerName,
			"models":   modelList,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   models,
	})
}

// GetProviderStatus 获取提供商状态
// @Summary 获取提供商状态
// @Description 获取所有AI服务提供商的健康状态和指标
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{} "提供商状态"
// @Router /v1/providers/status [get]
func (h *AIHandler) GetProviderStatus(c *gin.Context) {
	metrics := h.manager.GetMetrics()

	status := make(map[string]interface{})
	for providerName, metric := range metrics {
		status[providerName] = gin.H{
			"status":            metric.Status,
			"request_count":     metric.RequestCount,
			"error_count":       metric.ErrorCount,
			"error_rate":        h.calculateErrorRate(metric),
			"average_latency":   metric.AverageLatency.String(),
			"last_request_time": metric.LastRequestTime.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": status,
		"timestamp": gin.H{
			"unix": gin.H{
				"seconds": gin.H{},
			},
		},
	})
}

// calculateErrorRate 计算错误率
func (h *AIHandler) calculateErrorRate(metric *providers.ProviderMetrics) float64 {
	if metric.RequestCount == 0 {
		return 0.0
	}
	return float64(metric.ErrorCount) / float64(metric.RequestCount)
}

// GetProviderMetrics 获取详细的提供商指标
// @Summary 获取提供商指标
// @Description 获取指定提供商的详细指标信息
// @Tags AI
// @Param provider path string true "提供商名称"
// @Produce json
// @Success 200 {object} map[string]interface{} "提供商指标"
// @Failure 404 {object} map[string]interface{} "提供商不存在"
// @Router /v1/providers/{provider}/metrics [get]
func (h *AIHandler) GetProviderMetrics(c *gin.Context) {
	providerName := c.Param("provider")

	provider, exists := h.manager.GetProvider(providerName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message":  "Provider not found",
				"provider": providerName,
			},
		})
		return
	}

	metrics := h.manager.GetMetrics()
	metric, exists := metrics[providerName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message":  "Metrics not found for provider",
				"provider": providerName,
			},
		})
		return
	}

	response := gin.H{
		"provider":          providerName,
		"status":            metric.Status,
		"request_count":     metric.RequestCount,
		"error_count":       metric.ErrorCount,
		"error_rate":        h.calculateErrorRate(metric),
		"average_latency":   metric.AverageLatency.String(),
		"last_request_time": metric.LastRequestTime.Format("2006-01-02 15:04:05"),
		"models":            provider.GetModels(),
		"config": gin.H{
			"base_url":    provider.GetConfig().BaseURL,
			"timeout":     provider.GetConfig().Timeout.String(),
			"retry_count": provider.GetConfig().RetryCount,
			"retry_delay": provider.GetConfig().RetryDelay.String(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// TestProvider 测试提供商连接
// @Summary 测试提供商
// @Description 测试指定提供商的连接状态
// @Tags AI
// @Param provider path string true "提供商名称"
// @Produce json
// @Success 200 {object} map[string]interface{} "测试结果"
// @Failure 404 {object} map[string]interface{} "提供商不存在"
// @Failure 500 {object} map[string]interface{} "测试失败"
// @Router /v1/providers/{provider}/test [post]
func (h *AIHandler) TestProvider(c *gin.Context) {
	providerName := c.Param("provider")

	provider, exists := h.manager.GetProvider(providerName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message":  "Provider not found",
				"provider": providerName,
			},
		})
		return
	}

	// 执行健康检查
	err := provider.HealthCheck(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"provider": providerName,
			"status":   "unhealthy",
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": providerName,
		"status":   "healthy",
		"message":  "Provider is working correctly",
	})
}

// RegisterAIRoutes 注册AI服务相关路由
func RegisterAIRoutes(r *gin.RouterGroup, handler *AIHandler) {
	// OpenAI兼容的API
	r.POST("/chat/completions", handler.ChatCompletions)
	r.GET("/models", handler.GetModels)

	// 提供商管理API
	providers := r.Group("/providers")
	{
		providers.GET("/status", handler.GetProviderStatus)
		providers.GET("/:provider/metrics", handler.GetProviderMetrics)
		providers.POST("/:provider/test", handler.TestProvider)
	}
}
