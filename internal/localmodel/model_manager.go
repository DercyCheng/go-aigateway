package localmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ModelManager manages the models for the Python model server
type ModelManager struct {
	modelPath     string
	pythonPath    string
	config        *config.LocalModelConfig
	mu            sync.Mutex
	downloadQueue map[string]bool
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        string `json:"size"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Downloaded  bool   `json:"downloaded"`
}

// NewModelManager creates a new model manager
func NewModelManager(modelPath, pythonPath string, cfg *config.LocalModelConfig) *ModelManager {
	return &ModelManager{
		modelPath:     modelPath,
		pythonPath:    pythonPath,
		config:        cfg,
		downloadQueue: make(map[string]bool),
	}
}

// ListModels returns a list of available models
func (mm *ModelManager) ListModels() ([]ModelInfo, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Define the available local models
	models := []ModelInfo{
		{
			ID:          "tiny-llama",
			Name:        "TinyLlama Chat",
			Type:        "chat",
			Size:        "small",
			Status:      "stopped",
			Description: "轻量对话模型，仅1.1B参数，适合本地部署",
			Downloaded:  mm.isModelDownloaded("TinyLlama/TinyLlama-1.1B-Chat-v1.0"),
		},
		{
			ID:          "phi-2",
			Name:        "Phi-2",
			Type:        "completion",
			Size:        "small",
			Status:      "stopped",
			Description: "微软研发的高性能小模型，性能优秀",
			Downloaded:  mm.isModelDownloaded("microsoft/phi-2"),
		},
		{
			ID:          "miniLM",
			Name:        "MiniLM Embeddings",
			Type:        "embedding",
			Size:        "small",
			Status:      "stopped",
			Description: "文本向量嵌入模型，适合本地部署",
			Downloaded:  mm.isModelDownloaded("sentence-transformers/all-MiniLM-L6-v2"),
		},
		{
			ID:          "mistral-7b",
			Name:        "Mistral-7B",
			Type:        "chat",
			Size:        "large",
			Status:      "stopped",
			Description: "高性能开源大模型，7B参数",
			Downloaded:  mm.isModelDownloaded("HuggingFaceH4/mistral-7b-instruct-v0.2"),
		},
	}

	// Add third-party models if enabled
	if mm.config != nil && mm.config.ThirdParty.Enabled {
		thirdPartyModels := []ModelInfo{
			{
				ID:          "qwen-turbo",
				Name:        "通义千问 Turbo",
				Type:        "chat",
				Size:        "medium",
				Status:      "available",
				Description: "阿里云通义千问模型，快速响应，适合日常对话",
				Downloaded:  true, // Third-party models are always "available"
			},
			{
				ID:          "qwen-plus",
				Name:        "通义千问 Plus",
				Type:        "chat",
				Size:        "large",
				Status:      "available",
				Description: "阿里云通义千问增强版，更强的推理能力",
				Downloaded:  true,
			},
			{
				ID:          "qwen-max",
				Name:        "通义千问 Max",
				Type:        "chat",
				Size:        "large",
				Status:      "available",
				Description: "阿里云通义千问旗舰版，最强性能",
				Downloaded:  true,
			}, {
				ID:          "text-embedding-v1",
				Name:        "阿里百炼 Embedding V1",
				Type:        "embedding",
				Size:        "medium",
				Status:      "available",
				Description: "阿里云文本嵌入模型V1版本",
				Downloaded:  true,
			},
			{
				ID:          "text-embedding-v2",
				Name:        "阿里百炼 Embedding V2",
				Type:        "embedding",
				Size:        "large",
				Status:      "available",
				Description: "阿里云文本嵌入模型V2版本，更高精度",
				Downloaded:  true,
			},
		}
		models = append(models, thirdPartyModels...)
	}

	return models, nil
}

// isModelDownloaded checks if a model has been downloaded
func (mm *ModelManager) isModelDownloaded(modelID string) bool {
	// The transformers library usually stores models in the cache directory
	// We'll check if the model directory exists in the cache
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// The path depends on the operating system
	// For Windows: %USERPROFILE%\.cache\huggingface\hub
	// For Linux/macOS: ~/.cache/huggingface/hub
	cachePath := filepath.Join(homeDir, ".cache", "huggingface", "hub", "models--"+strings.ReplaceAll(modelID, "/", "--"))

	_, err = os.Stat(cachePath)
	return err == nil
}

// DownloadModel downloads a model
func (mm *ModelManager) DownloadModel(ctx context.Context, modelID string) error {
	mm.mu.Lock()
	if mm.downloadQueue[modelID] {
		mm.mu.Unlock()
		return fmt.Errorf("model %s is already being downloaded", modelID)
	}
	mm.downloadQueue[modelID] = true
	mm.mu.Unlock()

	defer func() {
		mm.mu.Lock()
		delete(mm.downloadQueue, modelID)
		mm.mu.Unlock()
	}()

	// Map the model ID to the HuggingFace model ID
	var huggingfaceModelID string
	switch modelID {
	case "tiny-llama":
		huggingfaceModelID = "TinyLlama/TinyLlama-1.1B-Chat-v1.0"
	case "phi-2":
		huggingfaceModelID = "microsoft/phi-2"
	case "miniLM":
		huggingfaceModelID = "sentence-transformers/all-MiniLM-L6-v2"
	case "mistral-7b":
		huggingfaceModelID = "HuggingFaceH4/mistral-7b-instruct-v0.2"
	default:
		return fmt.Errorf("unknown model ID: %s", modelID)
	}

	// Create a temporary Python script to download the model
	scriptPath := filepath.Join(mm.modelPath, "download_model.py")
	script := fmt.Sprintf(`
import os
from transformers import AutoTokenizer, AutoModelForCausalLM, AutoModel

# Download the model
print("Downloading model: %s")
try:
    if "sentence-transformers" in "%s":
        AutoModel.from_pretrained("%s")
    else:
        AutoTokenizer.from_pretrained("%s")
        AutoModelForCausalLM.from_pretrained("%s")
    print("Model downloaded successfully")
except Exception as e:
    print(f"Error downloading model: {str(e)}")
    exit(1)
`, huggingfaceModelID, huggingfaceModelID, huggingfaceModelID, huggingfaceModelID, huggingfaceModelID)

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("failed to create download script: %w", err)
	}
	defer os.Remove(scriptPath)

	// Run the script
	cmd := exec.CommandContext(ctx, mm.pythonPath, scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}

	return nil
}

// GetModelStatus checks the status of a model
func (mm *ModelManager) GetModelStatus(modelID string) (string, error) {
	// For now, we'll just check if the model is being downloaded
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.downloadQueue[modelID] {
		return "loading", nil
	}

	// Check if the model is currently running by making a request to the health endpoint
	// This assumes the server is running on localhost:5000
	resp, err := http.Get("http://localhost:5000/health")
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return "running", nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	return "stopped", nil
}

// StartModel starts a model
func (mm *ModelManager) StartModel(ctx context.Context, modelID string, modelType, modelSize string) error {
	// Check if the model is already running
	status, err := mm.GetModelStatus(modelID)
	if err != nil {
		return err
	}
	if status == "running" {
		return nil
	}
	if status == "loading" {
		return fmt.Errorf("model %s is currently being downloaded", modelID)
	}

	// Start the model
	scriptPath := filepath.Join(mm.modelPath, "server.py")
	cmd := exec.CommandContext(ctx, mm.pythonPath, scriptPath,
		"--model-type", modelType,
		"--model-size", modelSize,
		"--host", "localhost",
		"--port", "5000")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start model: %w", err)
	}

	// Wait for the server to start
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:5000/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to connect to model server")
}

// StopModel stops a model
func (mm *ModelManager) StopModel(ctx context.Context) error {
	// Find the process running on port 5000 and kill it
	// This is a simplified approach and may not work in all environments
	// For a more robust solution, you would need to track the process ID when starting the model

	// For Windows
	cmd := exec.CommandContext(ctx, "taskkill", "/F", "/IM", "python.exe")
	if err := cmd.Run(); err != nil {
		// Try Linux/macOS approach
		cmd = exec.CommandContext(ctx, "pkill", "-f", "server.py")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop model: %w", err)
		}
	}

	return nil
}

// UpdateModelSettings updates the settings for a model
func (mm *ModelManager) UpdateModelSettings(modelID string, maxTokens int, temperature float64) error {
	// In a real implementation, you would store these settings in a database or config file
	// For now, we'll just log them
	logrus.WithFields(logrus.Fields{
		"modelID":     modelID,
		"maxTokens":   maxTokens,
		"temperature": temperature,
	}).Info("Updated model settings")

	return nil
}

// GetServerHealth checks if the server is running
func (mm *ModelManager) GetServerHealth() (bool, error) {
	resp, err := http.Get("http://localhost:5000/health")
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

// GetServerModels gets the models available from the server
func (mm *ModelManager) GetServerModels() (map[string]interface{}, error) {
	resp, err := http.Get("http://localhost:5000/v1/models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
