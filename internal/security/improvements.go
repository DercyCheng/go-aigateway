package security

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityImprovements provides enhanced security features
type SecurityImprovements struct {
	logger          *logrus.Logger
	bannedIPs       map[string]time.Time
	loginAttempts   map[string]int
	rateLimitWindow time.Duration
}

// NewSecurityImprovements creates enhanced security middleware
func NewSecurityImprovements() *SecurityImprovements {
	return &SecurityImprovements{
		logger:          logrus.New(),
		bannedIPs:       make(map[string]time.Time),
		loginAttempts:   make(map[string]int),
		rateLimitWindow: 15 * time.Minute,
	}
}

// EnhancedAPIKeyValidation provides secure API key validation
func (si *SecurityImprovements) EnhancedAPIKeyValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key
		apiKey := c.GetHeader("Authorization")
		if apiKey == "" {
			apiKey = c.GetHeader("X-API-Key")
		}

		if apiKey == "" {
			si.logSecurityEvent(c, "missing_api_key", "No API key provided")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key required",
					"code":    "missing_api_key",
				},
			})
			c.Abort()
			return
		}

		// Remove Bearer prefix if present
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")

		// Validate API key length and format
		if len(apiKey) < 32 || len(apiKey) > 512 {
			si.logSecurityEvent(c, "invalid_api_key_length", fmt.Sprintf("API key length: %d", len(apiKey)))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key format",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		// Check for suspicious patterns
		if si.containsSuspiciousPatterns(apiKey) {
			si.logSecurityEvent(c, "suspicious_api_key", "API key contains suspicious patterns")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecureCompareAPIKey provides constant-time API key comparison
func (si *SecurityImprovements) SecureCompareAPIKey(provided, stored string) bool {
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(provided), []byte(stored)) == 1
}

// BruteForceProtection protects against brute force attacks
func (si *SecurityImprovements) BruteForceProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Check if IP is temporarily banned
		if banTime, exists := si.bannedIPs[clientIP]; exists {
			if time.Now().Before(banTime) {
				si.logSecurityEvent(c, "banned_ip_access", fmt.Sprintf("IP %s is banned", clientIP))
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": "Too many failed attempts. Please try again later.",
						"code":    "rate_limited",
					},
				})
				c.Abort()
				return
			} else {
				// Ban expired, remove from banned list
				delete(si.bannedIPs, clientIP)
				delete(si.loginAttempts, clientIP)
			}
		}

		c.Next()

		// Check for authentication failures
		if c.Writer.Status() == http.StatusUnauthorized {
			si.loginAttempts[clientIP]++

			// Ban after 5 failed attempts
			if si.loginAttempts[clientIP] >= 5 {
				si.bannedIPs[clientIP] = time.Now().Add(si.rateLimitWindow)
				si.logSecurityEvent(c, "ip_banned", fmt.Sprintf("IP %s banned for %v", clientIP, si.rateLimitWindow))
			}
		} else if c.Writer.Status() == http.StatusOK {
			// Reset attempts on successful authentication
			delete(si.loginAttempts, clientIP)
		}
	}
}

// RequestSizeLimit with configurable limits per endpoint
func (si *SecurityImprovements) RequestSizeLimit(limits map[string]int64, defaultLimit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		limit := defaultLimit

		// Check for endpoint-specific limits
		for pattern, endpointLimit := range limits {
			if strings.Contains(path, pattern) {
				limit = endpointLimit
				break
			}
		}

		if c.Request.ContentLength > limit {
			si.logSecurityEvent(c, "request_too_large", fmt.Sprintf("Content-Length: %d, Limit: %d", c.Request.ContentLength, limit))
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"message": fmt.Sprintf("Request body too large. Maximum size: %d bytes", limit),
					"code":    "request_too_large",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// containsSuspiciousPatterns checks for suspicious patterns in API keys
func (si *SecurityImprovements) containsSuspiciousPatterns(apiKey string) bool {
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"data:",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"expression(",
		"../",
		"..\\",
		"%2e%2e",
		"%252e%252e",
	}

	lowerKey := strings.ToLower(apiKey)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// logSecurityEvent logs security-related events
func (si *SecurityImprovements) logSecurityEvent(c *gin.Context, eventType, details string) {
	si.logger.WithFields(logrus.Fields{
		"event_type":   eventType,
		"client_ip":    c.ClientIP(),
		"user_agent":   c.GetHeader("User-Agent"),
		"request_path": c.Request.URL.Path,
		"method":       c.Request.Method,
		"details":      details,
		"timestamp":    time.Now().UTC(),
	}).Warn("Security event detected")
}
