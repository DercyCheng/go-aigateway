package handlers

import (
	"net/http"

	"go-aigateway/internal/config"

	"github.com/gin-gonic/gin"
)

// TestAPIHandler provides a simple test endpoint to verify API functionality
func TestAPIHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verify that the basic proxy configuration is working
		if cfg.TargetURL == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "Gateway not properly configured",
					"type":    "configuration_error",
					"code":    "missing_target_url",
					"hint":    "Please set TARGET_URL environment variable",
				},
			})
			return
		}

		if len(cfg.GatewayKeys) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "No gateway API keys configured",
					"type":    "configuration_error",
					"code":    "missing_api_keys",
					"hint":    "Please set GATEWAY_API_KEYS environment variable",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "AI Gateway is properly configured and ready",
			"target_url":      cfg.TargetURL,
			"has_api_key":     cfg.TargetKey != "",
			"configured_keys": len(cfg.GatewayKeys),
			"endpoints": gin.H{
				"chat_completions": "/v1/chat/completions",
				"completions":      "/v1/completions",
				"models":           "/v1/models",
				"health":           "/health",
				"test":             "/test",
			},
			"authentication": gin.H{
				"method": "Bearer token in Authorization header",
				"keys":   cfg.GatewayKeys[:min(len(cfg.GatewayKeys), 3)], // Show first 3 keys for testing
			},
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
