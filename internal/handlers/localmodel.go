package handlers

import (
	"encoding/json"
	"go-aigateway/internal/config"
	"go-aigateway/internal/localmodel"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LocalModelHandler handles requests to the local model
type LocalModelHandler struct {
	manager *localmodel.Manager
	config  *config.LocalModelConfig
}

// NewLocalModelHandler creates a new local model handler
func NewLocalModelHandler(manager *localmodel.Manager, cfg *config.LocalModelConfig) *LocalModelHandler {
	return &LocalModelHandler{
		manager: manager,
		config:  cfg,
	}
}

// LocalChatCompletions handles requests to the local chat completions API
func (h *LocalModelHandler) LocalChatCompletions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.WithError(err).Error("Failed to read request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to read request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Parse request
		var request localmodel.ChatCompletionRequest
		if err := json.Unmarshal(body, &request); err != nil {
			logrus.WithError(err).Error("Failed to parse request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to parse request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Set default values if not provided
		if request.MaxTokens == 0 {
			request.MaxTokens = h.config.MaxTokens
		}
		if request.Temperature == 0 {
			request.Temperature = h.config.Temperature
		}

		// Call local model
		response, err := h.manager.GetServer().ChatCompletion(c.Request.Context(), &request)
		if err != nil {
			logrus.WithError(err).Error("Failed to call local model")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to call local model",
					"type":    "internal_server_error",
					"code":    "local_model_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// LocalCompletions handles requests to the local completions API
func (h *LocalModelHandler) LocalCompletions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.WithError(err).Error("Failed to read request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to read request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Parse request
		var request localmodel.CompletionRequest
		if err := json.Unmarshal(body, &request); err != nil {
			logrus.WithError(err).Error("Failed to parse request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to parse request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Set default values if not provided
		if request.MaxTokens == 0 {
			request.MaxTokens = h.config.MaxTokens
		}
		if request.Temperature == 0 {
			request.Temperature = h.config.Temperature
		}

		// Call local model
		response, err := h.manager.GetServer().Completion(c.Request.Context(), &request)
		if err != nil {
			logrus.WithError(err).Error("Failed to call local model")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to call local model",
					"type":    "internal_server_error",
					"code":    "local_model_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// LocalEmbeddings handles requests to the local embeddings API
func (h *LocalModelHandler) LocalEmbeddings() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logrus.WithError(err).Error("Failed to read request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to read request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Parse request
		var request localmodel.EmbeddingRequest
		if err := json.Unmarshal(body, &request); err != nil {
			logrus.WithError(err).Error("Failed to parse request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Failed to parse request body",
					"type":    "invalid_request_error",
					"code":    "bad_request",
				},
			})
			return
		}

		// Call local model
		response, err := h.manager.GetServer().Embedding(c.Request.Context(), &request)
		if err != nil {
			logrus.WithError(err).Error("Failed to call local model")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to call local model",
					"type":    "internal_server_error",
					"code":    "local_model_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// LocalModels handles requests to the local models API
func (h *LocalModelHandler) LocalModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Call local model
		response, err := h.manager.GetServer().Models(c.Request.Context())
		if err != nil {
			logrus.WithError(err).Error("Failed to call local model")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to call local model",
					"type":    "internal_server_error",
					"code":    "local_model_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// RegisterLocalModelRoutes registers the local model routes
func RegisterLocalModelRoutes(r *gin.Engine, handler *LocalModelHandler) {
	// Local model routes
	localModel := r.Group("/local")
	localModel.POST("/chat/completions", handler.LocalChatCompletions())
	localModel.POST("/completions", handler.LocalCompletions())
	localModel.POST("/embeddings", handler.LocalEmbeddings())
	localModel.GET("/models", handler.LocalModels())
}
