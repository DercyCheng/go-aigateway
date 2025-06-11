package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-aigateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", HealthCheck)

	// Test
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "ai-gateway", response["service"])
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "version")
}

// TestChatCompletions tests the chat completions endpoint
func TestChatCompletions(t *testing.T) {
	// Create a mock server to simulate the target API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a successful response
		response := gin.H{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1677652288,
			"choices": []gin.H{
				{
					"message": gin.H{
						"role":    "assistant",
						"content": "Hello! How can I help you today?",
					},
					"finish_reason": "stop",
					"index":         0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		setupMock      func(*httptest.Server)
	}{
		{
			name: "Valid Request",
			requestBody: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello"},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Empty Request Body",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusOK, // Empty body should still proxy to target
		},
		{
			name: "Invalid Message Format",
			requestBody: map[string]interface{}{
				"model":    "gpt-3.5-turbo",
				"messages": "invalid",
			},
			expectedStatus: http.StatusOK, // Invalid format should still proxy to target
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			router := gin.New()

			cfg := &config.Config{
				TargetURL: mockServer.URL,
				TargetKey: "test-key",
			}

			router.POST("/chat/completions", ChatCompletions(cfg))

			// Prepare request
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/chat/completions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-api-key")

			// Test
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Contains(t, errorResponse, "error")
			}
		})
	}
}

// TestAPIKeyValidation tests API key validation
func TestAPIKeyValidation(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			apiKey:         "valid-test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid API Key",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing API Key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			router := gin.New()

			cfg := &config.Config{
				GatewayKeys: []string{"valid-test-key"},
			} // Add middleware
			router.Use(func(c *gin.Context) {
				// Mock API key validation
				authHeader := c.GetHeader("Authorization")

				// Check if Authorization header is missing
				if authHeader == "" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
					c.Abort()
					return
				}

				// Check if Authorization header has proper Bearer format
				if !strings.HasPrefix(authHeader, "Bearer ") {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
					c.Abort()
					return
				}

				token := authHeader[7:] // Remove "Bearer "

				valid := false
				for _, key := range cfg.GatewayKeys {
					if key == token {
						valid = true
						break
					}
				}

				if !valid {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
					c.Abort()
					return
				}
				c.Next()
			})

			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Prepare request
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("Authorization", "Bearer "+tt.apiKey)
			}

			// Test
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", HealthCheck)

	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// TestProxyRequestErrorHandling tests error handling in proxy requests
func TestProxyRequestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		targetURL      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid Target URL",
			targetURL:      "invalid-url",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "invalid_target",
		},
		{
			name:           "Empty Target URL",
			targetURL:      "",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "invalid_target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			router := gin.New()

			cfg := &config.Config{
				TargetURL: tt.targetURL,
				TargetKey: "test-key",
			}

			router.POST("/chat/completions", ChatCompletions(cfg))

			// Prepare request
			body := map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)
			req, _ := http.NewRequest("POST", "/chat/completions", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Test
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			var errorResponse map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			require.NoError(t, err)

			errorObj, ok := errorResponse["error"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, tt.expectedError, errorObj["code"])
		})
	}
}
