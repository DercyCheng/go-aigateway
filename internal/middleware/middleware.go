package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"go-aigateway/internal/config"
	"go-aigateway/internal/ram"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// API Key authentication middleware
func APIKeyAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key required",
					"type":    "authentication_error",
					"code":    "api_key_required",
				},
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key format",
					"type":    "authentication_error",
					"code":    "invalid_api_key_format",
				},
			})
			c.Abort()
			return
		}

		// Validate API key
		valid := false
		for _, key := range cfg.GatewayKeys {
			if strings.TrimSpace(key) == token {
				valid = true
				// Record API key usage for metrics
				keyPrefix := token
				if len(token) > 10 {
					keyPrefix = token[:10] + "..."
				}
				RecordAPIKeyUsage(keyPrefix)
				break
			}
		}

		if !valid {
			logrus.WithField("token", token[:min(len(token), 10)]+"...").Warn("Invalid API key attempt")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RAM authentication middleware
func RAMAuth(authenticator *ram.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and metrics endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Extract RAM authentication headers
		accessKeyID := c.GetHeader("X-Ca-Key")
		signature := c.GetHeader("X-Ca-Signature")
		timestamp := c.GetHeader("X-Ca-Timestamp")

		if accessKeyID == "" || signature == "" || timestamp == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "RAM authentication required",
					"type":    "authentication_error",
					"code":    "ram_auth_required",
				},
			})
			c.Abort()
			return
		}

		// Validate signature
		valid, err := authenticator.ValidateRequest(c.Request, accessKeyID, signature, timestamp)
		if err != nil {
			logrus.WithError(err).Error("RAM authentication validation error")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "RAM authentication validation failed",
					"type":    "authentication_error",
					"code":    "ram_auth_invalid",
				},
			})
			c.Abort()
			return
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid RAM signature",
					"type":    "authentication_error",
					"code":    "ram_signature_invalid",
				},
			})
			c.Abort()
			return
		}

		// Store access key ID in context for later use
		c.Set("ram_access_key_id", accessKeyID)
		c.Next()
	}
}

// Rate limiter middleware
type rateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
	}
}

func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	limiter := newRateLimiter(requestsPerMinute)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !limiter.allow(clientIP) {
			// Record rate limit hit for metrics
			RecordRateLimitHit(clientIP)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
					"code":    "rate_limit_exceeded",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *rateLimiter) allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Clean old requests
	if requests, exists := rl.requests[clientIP]; exists {
		validRequests := make([]time.Time, 0)
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[clientIP] = validRequests
	}

	// Check if under limit
	if len(rl.requests[clientIP]) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests[clientIP] = append(rl.requests[clientIP], now)
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
