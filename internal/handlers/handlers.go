package handlers

import (
	"bytes"
	"encoding/json"
	"go-aigateway/internal/config"
	"go-aigateway/internal/middleware"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthCheck handler
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "ai-gateway",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// ChatCompletions handler
func ChatCompletions(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxyRequest(c, cfg, "/chat/completions")
	}
}

// Completions handler (legacy)
func Completions(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxyRequest(c, cfg, "/completions")
	}
}

// Models handler
func Models(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxyRequest(c, cfg, "/models")
	}
}

// Generic proxy handler
func proxyRequest(c *gin.Context, cfg *config.Config, endpoint string) {
	start := time.Now()

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

	// Create target URL
	targetURL := strings.TrimSuffix(cfg.TargetURL, "/") + endpoint

	// Create new request
	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewBuffer(body))
	if err != nil {
		logrus.WithError(err).Error("Failed to create proxy request")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Internal server error",
				"type":    "internal_server_error",
				"code":    "proxy_error",
			},
		})
		return
	}

	// Copy headers from original request
	for key, values := range c.Request.Header {
		// Skip Authorization header as we'll set our own
		if strings.ToLower(key) == "authorization" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set target API authorization
	if cfg.TargetKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.TargetKey)
	}

	// Set content type if not present
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Copy query parameters
	req.URL.RawQuery = c.Request.URL.RawQuery

	// Log request
	logrus.WithFields(logrus.Fields{
		"method":     req.Method,
		"url":        req.URL.String(),
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
	}).Info("Proxying request")

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		duration := time.Since(start)
		middleware.RecordProxyRequest(endpoint, http.StatusBadGateway, duration)

		logrus.WithError(err).Error("Failed to execute proxy request")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "Failed to connect to target API",
				"type":    "api_connection_error",
				"code":    "connection_error",
			},
		})
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		duration := time.Since(start)
		middleware.RecordProxyRequest(endpoint, http.StatusBadGateway, duration)

		logrus.WithError(err).Error("Failed to read response body")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "Failed to read target API response",
				"type":    "api_response_error",
				"code":    "response_error",
			},
		})
		return
	}

	// Record successful proxy request metrics
	duration := time.Since(start)
	middleware.RecordProxyRequest(endpoint, resp.StatusCode, duration)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Log response
	logrus.WithFields(logrus.Fields{
		"status_code":   resp.StatusCode,
		"response_size": len(respBody),
		"duration_ms":   duration.Milliseconds(),
	}).Info("Received response from target API")

	// Handle streaming responses
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(resp.StatusCode)
		c.Writer.Write(respBody)
		return
	}

	// For JSON responses, we might want to modify the response
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var jsonResp map[string]interface{}
		if err := json.Unmarshal(respBody, &jsonResp); err == nil {
			// Modify response if needed (e.g., add gateway info)
			c.JSON(resp.StatusCode, jsonResp)
			return
		}
	}

	// Return raw response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}
