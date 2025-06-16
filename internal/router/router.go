package router

import (
	"time"

	"go-aigateway/internal/cloud"
	"go-aigateway/internal/config"
	"go-aigateway/internal/handlers"
	"go-aigateway/internal/middleware"
	"go-aigateway/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config, localAuth *security.LocalAuthenticator) {
	// Health check endpoint (no auth required)
	if cfg.HealthCheck {
		r.GET("/health", handlers.HealthCheck)
		r.GET("/", handlers.HealthCheck)
	}

	// Test endpoint to verify configuration (no auth required)
	r.GET("/test", handlers.TestAPIHandler(cfg))

	// Metrics endpoint (no auth required)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Standardized API v1 group for management APIs
	apiV1 := r.Group("/api/v1")

	// Authentication endpoints (no auth required for login)
	auth := apiV1.Group("/auth")
	{
		auth.POST("/login", handlers.Login(localAuth))
		auth.POST("/refresh", handlers.RefreshToken(localAuth))
	}

	// API management endpoints (admin auth required)
	admin := apiV1.Group("/admin")
	admin.Use(middleware.LocalAuth(localAuth, "admin"))
	{
		admin.POST("/api-keys", handlers.CreateAPIKey(localAuth))
		admin.GET("/api-keys", handlers.ListAPIKeys(localAuth))
		admin.DELETE("/api-keys/:id", handlers.DeleteAPIKey(localAuth))
		admin.PUT("/api-keys/:id", handlers.UpdateAPIKey(localAuth))
	}

	// Backward compatibility - Legacy authentication endpoints (deprecated but supported)
	legacyAuth := r.Group("/auth")
	{
		legacyAuth.POST("/login", handlers.Login(localAuth))
		legacyAuth.POST("/refresh", handlers.RefreshToken(localAuth))
	}

	// Backward compatibility - Legacy admin endpoints (deprecated but supported)
	legacyAdmin := r.Group("/admin")
	legacyAdmin.Use(middleware.LocalAuth(localAuth, "admin"))
	{
		legacyAdmin.POST("/api-keys", handlers.CreateAPIKey(localAuth))
		legacyAdmin.GET("/api-keys", handlers.ListAPIKeys(localAuth))
		legacyAdmin.DELETE("/api-keys/:id", handlers.DeleteAPIKey(localAuth))
		legacyAdmin.PUT("/api-keys/:id", handlers.UpdateAPIKey(localAuth))
	}

	// OpenAI-compatible API routes with API key authentication for external clients
	api := r.Group("/v1")
	api.Use(middleware.APIKeyAuth(cfg))

	// Chat completions endpoint
	api.POST("/chat/completions", handlers.ChatCompletions(cfg))

	// Completions endpoint (legacy)
	api.POST("/completions", handlers.Completions(cfg))

	// Models endpoint
	api.GET("/models", handlers.Models(cfg))

	// Additional OpenAI-compatible endpoints
	api.POST("/engines/:engine/completions", handlers.Completions(cfg))
	api.POST("/engines/:engine/chat/completions", handlers.ChatCompletions(cfg))

	// Legacy API routes (for backward compatibility, no auth required for testing)
	legacy := r.Group("/api/v1")
	{
		legacy.POST("/chat", handlers.ChatCompletions(cfg))
		legacy.POST("/chat/completions", handlers.ChatCompletions(cfg))
		legacy.POST("/completions", handlers.Completions(cfg))
		legacy.GET("/models", handlers.Models(cfg))
	}
}

// SetupCloudRoutes sets up standardized cloud management routes
func SetupCloudRoutes(r *gin.Engine, integrator *cloud.CloudIntegrator) {
	if integrator == nil {
		return
	}

	// Define common handlers for reuse
	getServicesHandler := func(c *gin.Context) {
		services, err := integrator.GetServices()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"services": services})
	}

	getServiceHealthHandler := func(c *gin.Context) {
		serviceName := c.Param("name")
		health, err := integrator.GetServiceHealth(serviceName)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, health)
	}

	scaleServiceHandler := func(c *gin.Context) {
		serviceName := c.Param("name")
		var req struct {
			Replicas int `json:"replicas"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := integrator.ScaleService(serviceName, req.Replicas); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Service scaled successfully"})
	}

	getServiceMetricsHandler := func(c *gin.Context) {
		serviceName := c.Param("name")
		var timeRange cloud.TimeRange

		// Parse start time
		if startStr := c.Query("start"); startStr != "" {
			if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
				timeRange.Start = startTime
			} else {
				timeRange.Start = time.Now().Add(-1 * time.Hour)
			}
		} else {
			timeRange.Start = time.Now().Add(-1 * time.Hour)
		}

		// Parse end time
		if endStr := c.Query("end"); endStr != "" {
			if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
				timeRange.End = endTime
			} else {
				timeRange.End = time.Now()
			}
		} else {
			timeRange.End = time.Now()
		}

		metrics, err := integrator.GetMetrics(serviceName, timeRange)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, metrics)
	}

	getServiceLogsHandler := func(c *gin.Context) {
		serviceName := c.Param("name")
		var timeRange cloud.TimeRange

		// Parse start time
		if startStr := c.Query("start"); startStr != "" {
			if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
				timeRange.Start = startTime
			} else {
				timeRange.Start = time.Now().Add(-1 * time.Hour)
			}
		} else {
			timeRange.Start = time.Now().Add(-1 * time.Hour)
		}

		// Parse end time
		if endStr := c.Query("end"); endStr != "" {
			if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
				timeRange.End = endTime
			} else {
				timeRange.End = time.Now()
			}
		} else {
			timeRange.End = time.Now()
		}

		logs, err := integrator.GetLogs(serviceName, timeRange)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"logs": logs})
	}

	updateServiceConfigHandler := func(c *gin.Context) {
		serviceName := c.Param("name")
		var config map[string]interface{}
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := integrator.UpdateConfiguration(serviceName, config); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Configuration updated successfully"})
	}

	// Standardized cloud management routes under /api/v1
	cloudGroup := r.Group("/api/v1/cloud")
	{
		cloudGroup.GET("/services", getServicesHandler)
		cloudGroup.GET("/services/:name/health", getServiceHealthHandler)
		cloudGroup.POST("/services/:name/scale", scaleServiceHandler)
		cloudGroup.GET("/services/:name/metrics", getServiceMetricsHandler)
		cloudGroup.GET("/services/:name/logs", getServiceLogsHandler)
		cloudGroup.PUT("/services/:name/config", updateServiceConfigHandler)
	}

	// Backward compatibility - Legacy cloud routes (deprecated but supported)
	legacyCloudGroup := r.Group("/cloud")
	{
		legacyCloudGroup.GET("/services", getServicesHandler)
		legacyCloudGroup.GET("/services/:name/health", getServiceHealthHandler)
		legacyCloudGroup.POST("/services/:name/scale", scaleServiceHandler)
		legacyCloudGroup.GET("/services/:name/metrics", getServiceMetricsHandler)
		legacyCloudGroup.GET("/services/:name/logs", getServiceLogsHandler)
		legacyCloudGroup.PUT("/services/:name/config", updateServiceConfigHandler)
	}
}
