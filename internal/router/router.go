package router

import (
	"time"

	"go-aigateway/internal/config"
	"go-aigateway/internal/handlers"
	"go-aigateway/internal/middleware"
	"go-aigateway/internal/cloud"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config) {
	// Health check endpoint (no auth required)
	if cfg.HealthCheck {
		r.GET("/health", handlers.HealthCheck)
		r.GET("/", handlers.HealthCheck)
	}

	// Metrics endpoint (no auth required)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes with authentication
	api := r.Group("/v1")
	api.Use(middleware.APIKeyAuth(cfg))

	// Chat completions endpoint
	api.POST("/chat/completions", handlers.ChatCompletions(cfg))

	// Completions endpoint (legacy)
	api.POST("/completions", handlers.Completions(cfg))

	// Models endpoint
	api.GET("/models", handlers.Models(cfg))
}

// SetupCloudRoutes sets up cloud management routes
func SetupCloudRoutes(r *gin.Engine, integrator *cloud.CloudIntegrator) {
	if integrator == nil {
		return
	}

	// Cloud management routes
	cloudGroup := r.Group("/cloud")
	{
		// Service management
		cloudGroup.GET("/services", func(c *gin.Context) {
			services, err := integrator.GetServices()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"services": services})
		})

		cloudGroup.GET("/services/:name/health", func(c *gin.Context) {
			serviceName := c.Param("name")
			health, err := integrator.GetServiceHealth(serviceName)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, health)
		})

		cloudGroup.POST("/services/:name/scale", func(c *gin.Context) {
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
		})

		cloudGroup.GET("/services/:name/metrics", func(c *gin.Context) {
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
		})

		cloudGroup.GET("/services/:name/logs", func(c *gin.Context) {
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
		})

		cloudGroup.PUT("/services/:name/config", func(c *gin.Context) {
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
		})
	}
}
