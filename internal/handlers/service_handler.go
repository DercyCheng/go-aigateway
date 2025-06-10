package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Service represents a service in the system
type Service struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Model           string    `json:"model"`
	Provider        string    `json:"provider"`
	Status          string    `json:"status"`
	Requests        int64     `json:"requests"`
	AvgResponseTime float64   `json:"avgResponseTime"`
	SuccessRate     float64   `json:"successRate"`
	LastCheck       string    `json:"lastCheck"`
	Endpoint        string    `json:"endpoint"`
	RateLimit       int       `json:"rateLimit"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ServiceSource represents a service source configuration
type ServiceSource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Endpoint    string    `json:"endpoint"`
	APIKey      string    `json:"apiKey"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Route represents a routing rule
type Route struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Path       string                 `json:"path"`
	Method     string                 `json:"method"`
	Target     string                 `json:"target"`
	Priority   int                    `json:"priority"`
	Enabled    bool                   `json:"enabled"`
	Conditions map[string]interface{} `json:"conditions"`
	Actions    map[string]interface{} `json:"actions"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// ServiceHandler handles service-related requests
type ServiceHandler struct {
	services       []Service
	serviceSources []ServiceSource
	routes         []Route
}

// NewServiceHandler creates a new service handler
func NewServiceHandler() *ServiceHandler {
	now := time.Now()

	// Initialize with real data that represents actual services
	services := []Service{
		{
			ID:              "openai-gpt4",
			Name:            "GPT-4 Turbo",
			Model:           "gpt-4-turbo-preview",
			Provider:        "OpenAI",
			Status:          "healthy",
			Requests:        0,
			AvgResponseTime: 0,
			SuccessRate:     100.0,
			LastCheck:       now.Format("2006-01-02 15:04:05"),
			Endpoint:        "/v1/chat/completions",
			RateLimit:       10000,
			Description:     "Latest GPT-4 Turbo model with enhanced capabilities",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "anthropic-claude",
			Name:            "Claude-3 Opus",
			Model:           "claude-3-opus-20240229",
			Provider:        "Anthropic",
			Status:          "healthy",
			Requests:        0,
			AvgResponseTime: 0,
			SuccessRate:     100.0,
			LastCheck:       now.Format("2006-01-02 15:04:05"),
			Endpoint:        "/v1/messages",
			RateLimit:       5000,
			Description:     "Most capable Claude model for complex tasks",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	serviceSources := []ServiceSource{
		{
			ID:          "openai-source",
			Name:        "OpenAI GPT-4",
			Type:        "openai",
			Endpoint:    "https://api.openai.com/v1",
			APIKey:      "sk-***...***abc",
			Status:      "active",
			Description: "OpenAI GPT-4 API 服务",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "anthropic-source",
			Name:        "Claude API",
			Type:        "anthropic",
			Endpoint:    "https://api.anthropic.com/v1",
			APIKey:      "sk-ant-***...***xyz",
			Status:      "active",
			Description: "Anthropic Claude API 服务",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	routes := []Route{
		{
			ID:       "openai-route",
			Name:     "ChatGPT API Route",
			Path:     "/api/v1/chat/completions",
			Method:   "POST",
			Target:   "https://api.openai.com/v1/chat/completions",
			Priority: 1,
			Enabled:  true,
			Conditions: map[string]interface{}{
				"headers": map[string]string{"Authorization": "Bearer *"},
			},
			Actions: map[string]interface{}{
				"rateLimit": 100,
				"timeout":   30000,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	return &ServiceHandler{
		services:       services,
		serviceSources: serviceSources,
		routes:         routes,
	}
}

// GetServices returns all services
func (h *ServiceHandler) GetServices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"services": h.services,
		},
	})
}

// GetServiceSources returns all service sources
func (h *ServiceHandler) GetServiceSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.serviceSources,
	})
}

// CreateServiceSource creates a new service source
func (h *ServiceHandler) CreateServiceSource(c *gin.Context) {
	var req ServiceSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	now := time.Now()
	req.ID = generateID()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.Status = "active"

	h.serviceSources = append(h.serviceSources, req)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    req,
	})
}

// UpdateServiceSource updates an existing service source
func (h *ServiceHandler) UpdateServiceSource(c *gin.Context) {
	id := c.Param("id")
	var req ServiceSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	for i, source := range h.serviceSources {
		if source.ID == id {
			req.ID = id
			req.CreatedAt = source.CreatedAt
			req.UpdatedAt = time.Now()
			h.serviceSources[i] = req

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    req,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Service source not found",
		},
	})
}

// DeleteServiceSource deletes a service source
func (h *ServiceHandler) DeleteServiceSource(c *gin.Context) {
	id := c.Param("id")

	for i, source := range h.serviceSources {
		if source.ID == id {
			h.serviceSources = append(h.serviceSources[:i], h.serviceSources[i+1:]...)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Service source deleted successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Service source not found",
		},
	})
}

// ToggleServiceSourceStatus toggles the status of a service source
func (h *ServiceHandler) ToggleServiceSourceStatus(c *gin.Context) {
	id := c.Param("id")

	for i, source := range h.serviceSources {
		if source.ID == id {
			if source.Status == "active" {
				h.serviceSources[i].Status = "inactive"
			} else {
				h.serviceSources[i].Status = "active"
			}
			h.serviceSources[i].UpdatedAt = time.Now()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.serviceSources[i],
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Service source not found",
		},
	})
}

// GetRoutes returns all routes
func (h *ServiceHandler) GetRoutes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.routes,
	})
}

// CreateRoute creates a new route
func (h *ServiceHandler) CreateRoute(c *gin.Context) {
	var req Route
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	now := time.Now()
	req.ID = generateID()
	req.CreatedAt = now
	req.UpdatedAt = now

	h.routes = append(h.routes, req)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    req,
	})
}

// UpdateRoute updates an existing route
func (h *ServiceHandler) UpdateRoute(c *gin.Context) {
	id := c.Param("id")
	var req Route
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	for i, route := range h.routes {
		if route.ID == id {
			req.ID = id
			req.CreatedAt = route.CreatedAt
			req.UpdatedAt = time.Now()
			h.routes[i] = req

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    req,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Route not found",
		},
	})
}

// DeleteRoute deletes a route
func (h *ServiceHandler) DeleteRoute(c *gin.Context) {
	id := c.Param("id")

	for i, route := range h.routes {
		if route.ID == id {
			h.routes = append(h.routes[:i], h.routes[i+1:]...)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Route deleted successfully",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Route not found",
		},
	})
}

// ToggleRouteStatus toggles the status of a route
func (h *ServiceHandler) ToggleRouteStatus(c *gin.Context) {
	id := c.Param("id")

	for i, route := range h.routes {
		if route.ID == id {
			h.routes[i].Enabled = !route.Enabled
			h.routes[i].UpdatedAt = time.Now()

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    h.routes[i],
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Route not found",
		},
	})
}

// Helper function to generate unique IDs
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000))
}

// RegisterServiceRoutes registers all service-related routes
func RegisterServiceRoutes(r *gin.Engine, handler *ServiceHandler) {
	api := r.Group("/api/v1")

	// Services
	api.GET("/monitoring/services", handler.GetServices)

	// Service Sources
	api.GET("/service-sources", handler.GetServiceSources)
	api.POST("/service-sources", handler.CreateServiceSource)
	api.PUT("/service-sources/:id", handler.UpdateServiceSource)
	api.DELETE("/service-sources/:id", handler.DeleteServiceSource)
	api.POST("/service-sources/:id/toggle", handler.ToggleServiceSourceStatus)

	// Routes
	api.GET("/routes", handler.GetRoutes)
	api.POST("/routes", handler.CreateRoute)
	api.PUT("/routes/:id", handler.UpdateRoute)
	api.DELETE("/routes/:id", handler.DeleteRoute)
	api.POST("/routes/:id/toggle", handler.ToggleRouteStatus)
}
