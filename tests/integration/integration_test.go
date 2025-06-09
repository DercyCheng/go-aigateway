package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"go-aigateway/internal/handlers"
	"go-aigateway/internal/middleware"
	"go-aigateway/internal/security"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", handlers.HealthHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.NotNil(t, response["timestamp"])
}

func TestChatEndpointWithAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		GatewayKeys: []string{"test-api-key-123"},
	}
	router := gin.New()
	router.Use(middleware.APIKeyAuth(cfg))
	router.POST("/api/v1/chat", handlers.ChatHandler(cfg))

	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hello, how are you?"},
		},
		"model": "gpt-3.5-turbo",
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	t.Run("valid API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-api-key-123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should not return 401 (authentication error)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		errorInfo := response["error"].(map[string]interface{})
		assert.Equal(t, "authentication_error", errorInfo["type"])
		assert.Equal(t, "api_key_required", errorInfo["code"])
	})

	t.Run("invalid API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid-api-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		errorInfo := response["error"].(map[string]interface{})
		assert.Equal(t, "authentication_error", errorInfo["type"])
		assert.Equal(t, "invalid_api_key", errorInfo["code"])
	})
}

func TestSecurityMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	securityConfig := &security.Config{
		MaxRequestSize:    1024,
		RateLimitEnabled:  true,
		RateLimitRequests: 5,
		RateLimitWindow:   time.Minute,
		CSRFProtection:    false, // Disable for testing
		XSSProtection:     true,
		SecureHeaders:     true,
		AuditLogging:      false, // Disable for testing
	}

	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	router := gin.New()
	router.Use(securityMiddleware.Handler())
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("security headers applied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader([]byte(`{"test": "data"}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Check security headers
		assert.Contains(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
		assert.Contains(t, w.Header().Get("X-Frame-Options"), "DENY")
		assert.Contains(t, w.Header().Get("X-XSS-Protection"), "1; mode=block")
	})

	t.Run("request size limit", func(t *testing.T) {
		largeData := bytes.Repeat([]byte("x"), 2048) // Larger than 1024 limit
		req := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader(largeData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("rate limiting", func(t *testing.T) {
		// Make multiple requests from the same IP
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewReader([]byte(`{"test": "data"}`)))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, w.Code)
			} else {
				// 6th request should be rate limited
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AllowedOrigins: []string{"http://localhost:3000"},
		GinMode:        "test",
	}

	router := gin.New()
	router.Use(middleware.CORS(cfg))
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("CORS headers added", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRateLimiterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RateLimiter(3)) // 3 requests per minute
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Make requests from the same IP
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if i < 3 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			// 4th and 5th requests should be rate limited
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Add a handler that triggers different types of errors
	router.POST("/api/v1/error/:type", func(c *gin.Context) {
		errorType := c.Param("type")

		switch errorType {
		case "validation":
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"message": "Invalid input data",
					"type":    "validation_error",
					"code":    "invalid_input",
				},
			})
		case "auth":
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Authentication required",
					"type":    "authentication_error",
					"code":    "auth_required",
				},
			})
		case "server":
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Internal server error",
					"type":    "server_error",
					"code":    "internal_error",
				},
			})
		default:
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		}
	})

	tests := []struct {
		errorType    string
		expectedCode int
		expectedType string
	}{
		{"validation", http.StatusBadRequest, "validation_error"},
		{"auth", http.StatusUnauthorized, "authentication_error"},
		{"server", http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("error type %s", tt.errorType), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/error/%s", tt.errorType), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			errorInfo := response["error"].(map[string]interface{})
			assert.Equal(t, tt.expectedType, errorInfo["type"])
		})
	}
}

func TestFullStackAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up configuration
	cfg := &config.Config{
		GatewayKeys: []string{"test-api-key-123"},
	}

	securityConfig := &security.Config{
		MaxRequestSize:    1024 * 1024,
		RateLimitEnabled:  true,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		CSRFProtection:    false,
		XSSProtection:     true,
		SecureHeaders:     true,
		AuditLogging:      false,
	}
	// Set up middleware
	securityMiddleware := security.NewSecurityMiddleware(securityConfig)

	router := gin.New()
	router.Use(middleware.CORS(cfg))
	router.Use(securityMiddleware.Handler())

	// Public endpoints (no auth required)
	router.GET("/health", handlers.HealthHandler)
	// API key protected endpoints
	apiKeyGroup := router.Group("/api/v1")
	apiKeyGroup.Use(middleware.APIKeyAuth(cfg))
	{
		apiKeyGroup.POST("/chat", handlers.ChatHandler(cfg))
		apiKeyGroup.POST("/completions", handlers.CompletionHandler(cfg))
	}

	t.Run("public endpoint accessible", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("API key endpoint with valid key", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-api-key-123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should not return authentication error
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)

		// Should have security headers
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
	})
}

func BenchmarkFullRequestFlow(b *testing.B) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		GatewayKeys: []string{"test-api-key-123"},
	}

	securityConfig := &security.Config{
		MaxRequestSize:   1024 * 1024,
		RateLimitEnabled: false, // Disable for benchmarking
		CSRFProtection:   false,
		XSSProtection:    true,
		SecureHeaders:    true,
		AuditLogging:     false,
	}
	securityMiddleware := security.NewSecurityMiddleware(securityConfig)
	router := gin.New()
	router.Use(middleware.CORS(cfg))
	router.Use(securityMiddleware.Handler())
	router.Use(middleware.APIKeyAuth(cfg))
	router.POST("/api/v1/chat", handlers.ChatHandler(cfg))

	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hello, how are you?"},
		},
		"model": "gpt-3.5-turbo",
	}

	jsonBody, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-api-key-123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
	}
}
