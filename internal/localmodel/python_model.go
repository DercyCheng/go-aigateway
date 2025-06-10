package localmodel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PythonModelServer handles interactions with a local Python model server
type PythonModelServer struct {
	config        *config.LocalModelConfig
	serverProcess *os.Process
	serverRunning bool
	mu            sync.Mutex
	httpClient    *http.Client
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completions API
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatCompletionResponse represents a response from the chat completions API
type ChatCompletionResponse struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	Created           int64  `json:"created"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Choices           []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CompletionRequest represents a request to the completions API
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// CompletionResponse represents a response from the completions API
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbeddingRequest represents a request to the embeddings API
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents a response from the embeddings API
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// APIModelInfo represents a model for API responses (OpenAI compatible)
type APIModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents a response from the models API
type ModelsResponse struct {
	Object string         `json:"object"`
	Data   []APIModelInfo `json:"data"`
}

// NewPythonModelServer creates a new instance of the Python model server
func NewPythonModelServer(cfg *config.LocalModelConfig) *PythonModelServer {
	return &PythonModelServer{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Start launches the Python model server
func (pms *PythonModelServer) Start(ctx context.Context) error {
	pms.mu.Lock()
	defer pms.mu.Unlock()

	if pms.serverRunning {
		return nil
	}

	// Ensure model directory exists
	if err := os.MkdirAll(pms.config.ModelPath, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Create Python server script
	scriptPath := filepath.Join(pms.config.ModelPath, "server.py")
	if err := pms.createServerScript(scriptPath); err != nil {
		return fmt.Errorf("failed to create server script: %w", err)
	}

	// Create requirements.txt
	requirementsPath := filepath.Join(pms.config.ModelPath, "requirements.txt")
	if err := pms.createRequirementsFile(requirementsPath); err != nil {
		return fmt.Errorf("failed to create requirements file: %w", err)
	}

	// Install requirements
	logrus.Info("Installing Python dependencies...")
	cmd := exec.Command(pms.config.PythonPath, "-m", "pip", "install", "-r", requirementsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Python dependencies: %w", err)
	}
	// Start the Python server
	logrus.WithFields(logrus.Fields{
		"host":                pms.config.ServerHost,
		"port":                pms.config.ServerPort,
		"third_party_enabled": pms.config.ThirdParty.Enabled,
	}).Info("Starting Python model server...")

	// Build command arguments
	cmdArgs := []string{
		scriptPath,
		"--host", pms.config.ServerHost,
		"--port", fmt.Sprintf("%d", pms.config.ServerPort),
		"--model-type", pms.config.ModelType,
		"--model-size", pms.config.ModelSize,
	}

	// Add third-party flag if enabled
	if pms.config.ThirdParty.Enabled {
		cmdArgs = append(cmdArgs, "--use-third-party")
	}

	cmd = exec.Command(pms.config.PythonPath, cmdArgs...)

	// Set environment variables for third-party configuration
	cmd.Env = os.Environ()
	if pms.config.ThirdParty.Enabled && pms.config.ThirdParty.APIKey != "" {
		cmd.Env = append(cmd.Env, "BAILIAN_API_KEY="+pms.config.ThirdParty.APIKey)
		cmd.Env = append(cmd.Env, "USE_THIRD_PARTY_MODEL=true")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Python server: %w", err)
	}

	pms.serverProcess = cmd.Process
	pms.serverRunning = true

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Check if server is running
	serverURL := fmt.Sprintf("http://%s:%d/health", pms.config.ServerHost, pms.config.ServerPort)
	for i := 0; i < 10; i++ {
		resp, err := http.Get(serverURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			logrus.Info("Python model server started successfully")
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to connect to Python server")
}

// Stop stops the Python model server
func (pms *PythonModelServer) Stop() error {
	pms.mu.Lock()
	defer pms.mu.Unlock()

	if !pms.serverRunning {
		return nil
	}

	logrus.Info("Stopping Python model server...")
	if err := pms.serverProcess.Kill(); err != nil {
		return fmt.Errorf("failed to stop Python server: %w", err)
	}

	pms.serverRunning = false
	return nil
}

// ChatCompletion sends a request to the chat completions API
func (pms *PythonModelServer) ChatCompletion(ctx context.Context, request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if request.MaxTokens == 0 {
		request.MaxTokens = pms.config.MaxTokens
	}
	if request.Temperature == 0 {
		request.Temperature = pms.config.Temperature
	}

	serverURL := fmt.Sprintf("http://%s:%d/v1/chat/completions", pms.config.ServerHost, pms.config.ServerPort)
	result, err := pms.retryRequest(ctx, serverURL, request, &ChatCompletionResponse{})
	if err != nil {
		return nil, err
	}
	return result.(*ChatCompletionResponse), nil
}

// Completion sends a request to the completions API
func (pms *PythonModelServer) Completion(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if request.MaxTokens == 0 {
		request.MaxTokens = pms.config.MaxTokens
	}
	if request.Temperature == 0 {
		request.Temperature = pms.config.Temperature
	}

	serverURL := fmt.Sprintf("http://%s:%d/v1/completions", pms.config.ServerHost, pms.config.ServerPort)
	result, err := pms.retryRequest(ctx, serverURL, request, &CompletionResponse{})
	if err != nil {
		return nil, err
	}
	return result.(*CompletionResponse), nil
}

// Embedding sends a request to the embeddings API
func (pms *PythonModelServer) Embedding(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	serverURL := fmt.Sprintf("http://%s:%d/v1/embeddings", pms.config.ServerHost, pms.config.ServerPort)
	result, err := pms.retryRequest(ctx, serverURL, request, &EmbeddingResponse{})
	if err != nil {
		return nil, err
	}
	return result.(*EmbeddingResponse), nil
}

// Models gets a list of available models
func (pms *PythonModelServer) Models(ctx context.Context) (*ModelsResponse, error) {
	serverURL := fmt.Sprintf("http://%s:%d/v1/models", pms.config.ServerHost, pms.config.ServerPort)
	result, err := pms.retryRequest(ctx, serverURL, nil, &ModelsResponse{})
	if err != nil {
		return nil, err
	}
	return result.(*ModelsResponse), nil
}

// retryRequest retries a request to the server with backoff
func (pms *PythonModelServer) retryRequest(ctx context.Context, url string, requestBody interface{}, responseBody interface{}) (interface{}, error) {
	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; attempt < pms.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			logrus.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"url":     url,
			}).Info("Retrying request to Python model server...")
			time.Sleep(pms.config.RetryDelay)
		}

		// Prepare request
		var reqBody io.Reader
		if requestBody != nil {
			jsonData, err := json.Marshal(requestBody)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request: %w", err)
			}

			if pms.config.LogRequests {
				logrus.WithField("request", string(jsonData)).Debug("Sending request to Python model server")
			}

			reqBody = bytes.NewBuffer(jsonData)
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err = pms.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		// If this is the last attempt and we got an error
		if attempt == pms.config.RetryAttempts-1 && err != nil {
			return nil, fmt.Errorf("failed to send request after %d attempts: %w", pms.config.RetryAttempts, err)
		}
	}

	// If we get here without a valid response, all attempts failed
	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server request failed after %d attempts", pms.config.RetryAttempts)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if pms.config.LogResponses {
		logrus.WithField("response", string(respData)).Debug("Received response from Python model server")
	}

	if err := json.Unmarshal(respData, responseBody); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return responseBody, nil
}

// createServerScript creates the Python server script
func (pms *PythonModelServer) createServerScript(scriptPath string) error {
	script := `
import argparse
import json
import time
import logging
from typing import List, Dict, Any, Optional
import os
import sys

import torch
from flask import Flask, request, jsonify
from transformers import AutoTokenizer, AutoModelForCausalLM, pipeline

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Global variables
model = None
tokenizer = None
embedding_model = None
model_type = "chat"
model_size = "small"

# Model selection based on size
MODEL_MAP = {
    "small": {
        "chat": "TinyLlama/TinyLlama-1.1B-Chat-v1.0",
        "completion": "microsoft/phi-2",
        "embedding": "sentence-transformers/all-MiniLM-L6-v2"
    },
    "medium": {
        "chat": "TinyLlama/TinyLlama-1.1B-Chat-v1.0",
        "completion": "microsoft/phi-2",
        "embedding": "sentence-transformers/all-MiniLM-L6-v2" 
    },
    "large": {
        "chat": "HuggingFaceH4/mistral-7b-instruct-v0.2",
        "completion": "google/gemma-2b",
        "embedding": "intfloat/e5-large-v2"
    }
}

def initialize_model():
    global model, tokenizer, embedding_model, model_type, model_size
    
    model_id = MODEL_MAP[model_size][model_type]
    logger.info(f"Initializing {model_type} model: {model_id}")
    
    device = "cuda" if torch.cuda.is_available() else "cpu"
    logger.info(f"Using device: {device}")
    
    if model_type in ["chat", "completion"]:
        tokenizer = AutoTokenizer.from_pretrained(model_id)
        model = AutoModelForCausalLM.from_pretrained(
            model_id,
            torch_dtype=torch.float16 if device == "cuda" else torch.float32,
            low_cpu_mem_usage=True,
            device_map=device
        )
    
    if model_type == "embedding" or model_size == "large":
        embedding_model_id = MODEL_MAP[model_size]["embedding"]
        logger.info(f"Initializing embedding model: {embedding_model_id}")
        embedding_model = pipeline("feature-extraction", model=embedding_model_id, device=device)

@app.route('/health', methods=['GET'])
def health_check():
    return jsonify({"status": "healthy"}), 200

@app.route('/v1/chat/completions', methods=['POST'])
def chat_completions():
    try:
        data = request.json
        messages = data.get('messages', [])
        max_tokens = data.get('max_tokens', 1024)
        temperature = data.get('temperature', 0.7)
        model_name = data.get('model', MODEL_MAP[model_size][model_type])
        
        logger.info(f"Chat completion request with {len(messages)} messages")
        
        # Format the conversation for the model
        prompt = ""
        for msg in messages:
            role = msg['role']
            content = msg['content']
            if role == 'system':
                prompt += f"<|system|>\n{content}\n"
            elif role == 'user':
                prompt += f"<|user|>\n{content}\n"
            elif role == 'assistant':
                prompt += f"<|assistant|>\n{content}\n"
        
        prompt += "<|assistant|>\n"
        
        inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
        
        # Generate response
        outputs = model.generate(
            inputs["input_ids"],
            max_new_tokens=max_tokens,
            temperature=temperature,
            do_sample=temperature > 0,
        )
        
        response_text = tokenizer.decode(outputs[0][inputs["input_ids"].shape[1]:], skip_special_tokens=True)
        
        # Create response
        response = {
            "id": f"chatcmpl-{int(time.time())}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": model_name,
            "system_fingerprint": "local-python-model",
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": response_text.strip()
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": inputs["input_ids"].shape[1],
                "completion_tokens": outputs.shape[1] - inputs["input_ids"].shape[1],
                "total_tokens": outputs.shape[1]
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        logger.error(f"Error in chat completions: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/v1/completions', methods=['POST'])
def completions():
    try:
        data = request.json
        prompt = data.get('prompt', '')
        max_tokens = data.get('max_tokens', 1024)
        temperature = data.get('temperature', 0.7)
        model_name = data.get('model', MODEL_MAP[model_size][model_type])
        
        logger.info(f"Completion request with prompt length: {len(prompt)}")
        
        inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
        
        # Generate response
        outputs = model.generate(
            inputs["input_ids"],
            max_new_tokens=max_tokens,
            temperature=temperature,
            do_sample=temperature > 0,
        )
        
        response_text = tokenizer.decode(outputs[0][inputs["input_ids"].shape[1]:], skip_special_tokens=True)
        
        # Create response
        response = {
            "id": f"cmpl-{int(time.time())}",
            "object": "text_completion",
            "created": int(time.time()),
            "model": model_name,
            "choices": [
                {
                    "text": response_text.strip(),
                    "index": 0,
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": inputs["input_ids"].shape[1],
                "completion_tokens": outputs.shape[1] - inputs["input_ids"].shape[1],
                "total_tokens": outputs.shape[1]
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        logger.error(f"Error in completions: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/v1/embeddings', methods=['POST'])
def embeddings():
    try:
        if embedding_model is None:
            return jsonify({"error": "Embedding model not initialized"}), 500
            
        data = request.json
        input_texts = data.get('input', [])
        model_name = data.get('model', MODEL_MAP[model_size]["embedding"])
        
        if isinstance(input_texts, str):
            input_texts = [input_texts]
            
        logger.info(f"Embedding request with {len(input_texts)} inputs")
        
        # Generate embeddings
        embeddings = []
        token_count = 0
        
        for i, text in enumerate(input_texts):
            # Get embedding
            embedding_output = embedding_model(text)
            # Average across tokens to get a single vector per text
            embedding_vector = torch.mean(torch.tensor(embedding_output[0]), dim=0).tolist()
            
            embeddings.append({
                "object": "embedding",
                "embedding": embedding_vector,
                "index": i
            })
            
            # Approximate token count
            token_count += len(text.split())
        
        # Create response
        response = {
            "object": "list",
            "data": embeddings,
            "model": model_name,
            "usage": {
                "prompt_tokens": token_count,
                "total_tokens": token_count
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        logger.error(f"Error in embeddings: {str(e)}")
        return jsonify({"error": str(e)}), 500

@app.route('/v1/models', methods=['GET'])
def list_models():
    models = [
        {
            "id": MODEL_MAP["small"]["chat"],
            "object": "model",
            "created": int(time.time()) - 10000,
            "owned_by": "local"
        },
        {
            "id": MODEL_MAP["small"]["completion"],
            "object": "model",
            "created": int(time.time()) - 10000,
            "owned_by": "local"
        },
        {
            "id": MODEL_MAP["small"]["embedding"],
            "object": "model",
            "created": int(time.time()) - 10000,
            "owned_by": "local"
        }
    ]
    
    if model_size != "small":
        for model_type in ["chat", "completion", "embedding"]:
            models.append({
                "id": MODEL_MAP[model_size][model_type],
                "object": "model",
                "created": int(time.time()) - 10000,
                "owned_by": "local"
            })
    
    response = {
        "object": "list",
        "data": models
    }
    
    return jsonify(response)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Local AI Model Server')
    parser.add_argument('--host', type=str, default='localhost', help='Host to bind the server to')
    parser.add_argument('--port', type=int, default=5000, help='Port to bind the server to')
    parser.add_argument('--model-type', type=str, default='chat', choices=['chat', 'completion', 'embedding'], help='Type of model to use')
    parser.add_argument('--model-size', type=str, default='small', choices=['small', 'medium', 'large'], help='Size of model to use')
    
    args = parser.parse_args()
    
    model_type = args.model_type
    model_size = args.model_size
    
    logger.info(f"Starting server with model type: {model_type}, size: {model_size}")
    initialize_model()
    
    app.run(host=args.host, port=args.port)
`
	return os.WriteFile(scriptPath, []byte(script), 0644)
}

// createRequirementsFile creates the requirements.txt file
func (pms *PythonModelServer) createRequirementsFile(requirementsPath string) error {
	requirements := `
flask==2.3.3
transformers==4.36.2
torch==2.1.0
sentencepiece==0.1.99
accelerate==0.25.0
`
	return os.WriteFile(requirementsPath, []byte(requirements), 0644)
}
