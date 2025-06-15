package handlers

import (
	"time"
)

// generateID generates a unique ID based on timestamp
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000))
}

// GetThirdPartyModelInfo returns information about third-party models (阿里百炼)
func GetThirdPartyModelInfo() map[string]ThirdPartyModelInfo {
	return map[string]ThirdPartyModelInfo{
		// 通义千问大语言模型系列
		"qwen-turbo": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-Turbo",
			ModelType:   "chat",
			MaxTokens:   8192,
		},
		"qwen-plus": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-Plus",
			ModelType:   "chat",
			MaxTokens:   32768,
		},
		"qwen-max": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-Max",
			ModelType:   "chat",
			MaxTokens:   8192,
		},
		"qwen-max-longcontext": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-Max长文本版",
			ModelType:   "chat",
			MaxTokens:   30000,
		},
		"qwen2-72b-instruct": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问2-72B",
			ModelType:   "chat",
			MaxTokens:   32768,
		},
		"qwen2-7b-instruct": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问2-7B",
			ModelType:   "chat",
			MaxTokens:   32768,
		},
		"qwen2-1.5b-instruct": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问2-1.5B",
			ModelType:   "chat",
			MaxTokens:   32768,
		},
		"qwen2-0.5b-instruct": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问2-0.5B",
			ModelType:   "chat",
			MaxTokens:   32768,
		},

		// 文本嵌入模型系列
		"text-embedding-v1": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通用文本向量 Small",
			ModelType:   "embedding",
			MaxTokens:   2048,
		},
		"text-embedding-v2": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通用文本向量 Large",
			ModelType:   "embedding",
			MaxTokens:   2048,
		},
		"text-embedding-v3": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通用文本向量 Large v3",
			ModelType:   "embedding",
			MaxTokens:   8192,
		},

		// 多模态模型
		"qwen-vl-plus": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-VL-Plus",
			ModelType:   "multimodal",
			MaxTokens:   8192,
		},
		"qwen-vl-max": {
			Provider:    "alibaba-dashscope",
			ChineseName: "通义千问-VL-Max",
			ModelType:   "multimodal",
			MaxTokens:   8192,
		},

		// 音频模型
		"paraformer-realtime-v1": {
			Provider:    "alibaba-dashscope",
			ChineseName: "实时语音识别",
			ModelType:   "speech-to-text",
			MaxTokens:   0, // N/A for audio models
		},
		"cosyvoice-v1": {
			Provider:    "alibaba-dashscope",
			ChineseName: "语音合成",
			ModelType:   "text-to-speech",
			MaxTokens:   0, // N/A for audio models
		},
	}
}

// ThirdPartyModelInfo contains information about a third-party model
type ThirdPartyModelInfo struct {
	Provider    string // The actual provider (e.g., "alibaba-dashscope")
	ChineseName string // Chinese name of the model
	ModelType   string // Type: "chat", "embedding", "multimodal", "speech-to-text", "text-to-speech"
	MaxTokens   int    // Maximum tokens supported
}

// IsThirdPartyModel checks if a model ID belongs to third-party providers (阿里百炼)
func IsThirdPartyModel(modelID string) bool {
	_, exists := GetThirdPartyModelInfo()[modelID]
	return exists
}
