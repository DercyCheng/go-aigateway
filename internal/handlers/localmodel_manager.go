package handlers

import (
	"go-aigateway/internal/config"
	"go-aigateway/internal/localmodel"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LocalModelManagerHandler handles requests for managing local models
type LocalModelManagerHandler struct {
	manager      *localmodel.Manager
	modelManager *localmodel.ModelManager
	config       *config.LocalModelConfig
}

// NewLocalModelManagerHandler creates a new local model manager handler
func NewLocalModelManagerHandler(manager *localmodel.Manager, modelManager *localmodel.ModelManager, cfg *config.LocalModelConfig) *LocalModelManagerHandler {
	return &LocalModelManagerHandler{
		manager:      manager,
		modelManager: modelManager,
		config:       cfg,
	}
}

// ListModels returns a list of available models
func (h *LocalModelManagerHandler) ListModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models, err := h.modelManager.ListModels()
		if err != nil {
			logrus.WithError(err).Error("Failed to list models")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list models",
			})
			return
		}

		// Check which models are running
		for i := range models {
			status, err := h.modelManager.GetModelStatus(models[i].ID)
			if err != nil {
				logrus.WithError(err).WithField("modelID", models[i].ID).Error("Failed to get model status")
				continue
			}
			models[i].Status = status
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"models": models,
			},
		})
	}
}

// DownloadModel downloads a model
func (h *LocalModelManagerHandler) DownloadModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("id")
		if modelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Model ID is required",
			})
			return
		}

		// Check if this is a third-party model (阿里百炼)
		if h.isThirdPartyModel(modelID) {
			// Third-party models (阿里百炼) don't need to be downloaded - they run on Alibaba Cloud
			c.JSON(http.StatusOK, gin.H{
				"message": "Third-party model (阿里百炼) is already available",
			})
			return
		}

		// Start the download in a goroutine so we don't block the request
		go func() {
			if err := h.modelManager.DownloadModel(c.Request.Context(), modelID); err != nil {
				logrus.WithError(err).WithField("modelID", modelID).Error("Failed to download model")
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Model download started",
		})
	}
}

// StartModel starts a model
func (h *LocalModelManagerHandler) StartModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("id")
		if modelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Model ID is required",
			})
			return
		}

		// Check if this is a third-party model (阿里百炼)
		if h.isThirdPartyModel(modelID) {
			// For third-party models (阿里百炼), we don't need to "start" them as they're cloud-based
			// Just return success to indicate the model is available
			c.JSON(http.StatusOK, gin.H{
				"message": "Third-party model (阿里百炼) is available",
			})
			return
		}

		// Get the model type and size for local models
		var modelType, modelSize string
		switch modelID {
		case "tiny-llama":
			modelType = "chat"
			modelSize = "small"
		case "phi-2":
			modelType = "completion"
			modelSize = "small"
		case "miniLM":
			modelType = "embedding"
			modelSize = "small"
		case "mistral-7b":
			modelType = "chat"
			modelSize = "large"
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unknown model ID",
			})
			return
		}

		// Start the model in a goroutine so we don't block the request
		go func() {
			if err := h.modelManager.StartModel(c.Request.Context(), modelID, modelType, modelSize); err != nil {
				logrus.WithError(err).WithField("modelID", modelID).Error("Failed to start model")
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Model start requested",
		})
	}
}

// StopModel stops a model
func (h *LocalModelManagerHandler) StopModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("id")
		if modelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Model ID is required",
			})
			return
		}

		// Check if this is a third-party model (阿里百炼)
		if h.isThirdPartyModel(modelID) {
			// For third-party models (阿里百炼), we don't need to "stop" them as they're cloud-based
			// Just return success
			c.JSON(http.StatusOK, gin.H{
				"message": "Third-party model (阿里百炼) is always available",
			})
			return
		}

		// Stop the model in a goroutine so we don't block the request
		go func() {
			if err := h.modelManager.StopModel(c.Request.Context()); err != nil {
				logrus.WithError(err).WithField("modelID", modelID).Error("Failed to stop model")
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message": "Model stop requested",
		})
	}
}

// UpdateModelSettings updates the settings for a model
func (h *LocalModelManagerHandler) UpdateModelSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("id")
		if modelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Model ID is required",
			})
			return
		}

		// Parse the settings from the request
		maxTokensStr := c.PostForm("maxTokens")
		temperatureStr := c.PostForm("temperature")

		maxTokens, err := strconv.Atoi(maxTokensStr)
		if err != nil {
			maxTokens = h.config.MaxTokens
		}

		temperature, err := strconv.ParseFloat(temperatureStr, 64)
		if err != nil {
			temperature = h.config.Temperature
		}

		// Update the settings
		if err := h.modelManager.UpdateModelSettings(modelID, maxTokens, temperature); err != nil {
			logrus.WithError(err).WithField("modelID", modelID).Error("Failed to update model settings")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update model settings",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Model settings updated",
		})
	}
}

// GetModelStatus returns the status of a model
func (h *LocalModelManagerHandler) GetModelStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("id")
		if modelID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Model ID is required",
			})
			return
		}

		// Check if this is a third-party model (阿里百炼)
		if h.isThirdPartyModel(modelID) {
			// Third-party models (阿里百炼) are always "available" when third-party is enabled
			if h.config.ThirdParty.Enabled {
				c.JSON(http.StatusOK, gin.H{
					"status": "available",
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"status": "unavailable",
				})
			}
			return
		}

		status, err := h.modelManager.GetModelStatus(modelID)
		if err != nil {
			logrus.WithError(err).WithField("modelID", modelID).Error("Failed to get model status")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get model status",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": status,
		})
	}
}

// isThirdPartyModel checks if the given model ID is a third-party model (阿里百炼)
func (h *LocalModelManagerHandler) isThirdPartyModel(modelID string) bool {
	// Use the centralized third-party model information
	return IsThirdPartyModel(modelID)
}

// RegisterLocalModelManagerRoutes registers the local model manager routes
func RegisterLocalModelManagerRoutes(r *gin.Engine, handler *LocalModelManagerHandler) {
	// Local model manager routes
	localModelManager := r.Group("/api/local/models")
	localModelManager.GET("", handler.ListModels())
	localModelManager.POST("/:id/download", handler.DownloadModel())
	localModelManager.POST("/:id/start", handler.StartModel())
	localModelManager.POST("/:id/stop", handler.StopModel())
	localModelManager.POST("/:id/settings", handler.UpdateModelSettings())
	localModelManager.GET("/:id/status", handler.GetModelStatus())
}
