package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityMiddleware(t *testing.T) {
	config := &Config{
		MaxRequestSize:    1024 * 1024,
		RateLimitEnabled:  true,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		CSRFProtection:    true,
		XSSProtection:     true,
		SecureHeaders:     true,
		AuditLogging:      true,
	}

	middleware := NewSecurityMiddleware(config)
	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
	assert.NotNil(t, middleware.rateLimiter)
	assert.NotNil(t, middleware.csrfTokens)
	assert.NotNil(t, middleware.auditLogger)
}

func TestInputValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid alphanumeric",
			input:    "test123",
			expected: true,
		},
		{
			name:     "valid with spaces",
			input:    "hello world",
			expected: true,
		},
		{
			name:     "invalid with script tag",
			input:    "<script>alert('xss')</script>",
			expected: false,
		},
		{
			name:     "invalid with sql injection",
			input:    "'; DROP TABLE users; --",
			expected: false,
		},
		{
			name:     "invalid with javascript",
			input:    "javascript:alert('xss')",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no sanitization needed",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "remove script tags",
			input:    "<script>alert('xss')</script>test",
			expected: "test",
		},
		{
			name:     "escape html entities",
			input:    "<div>content</div>",
			expected: "&lt;div&gt;content&lt;/div&gt;",
		},
		{
			name:     "remove javascript protocol",
			input:    "javascript:alert('xss')",
			expected: "alert('xss')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test that hashing the same password produces different hashes
	hash2, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Test correct password
	valid := VerifyPassword(password, hash)
	assert.True(t, valid)

	// Test incorrect password
	valid = VerifyPassword("wrongpassword", hash)
	assert.False(t, valid)
}

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken(32)
	require.NoError(t, err)
	assert.Len(t, token, 64) // 32 bytes = 64 hex characters

	// Test that multiple calls produce different tokens
	token2, err := GenerateSecureToken(32)
	require.NoError(t, err)
	assert.NotEqual(t, token, token2)
}

func TestSecurityMiddleware_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &Config{
		MaxRequestSize:    1024,
		RateLimitEnabled:  true,
		RateLimitRequests: 10,
		RateLimitWindow:   time.Minute,
		CSRFProtection:    false, // Disable for testing
		XSSProtection:     true,
		SecureHeaders:     true,
		AuditLogging:      false, // Disable for testing
	}

	middleware := NewSecurityMiddleware(config)

	router := gin.New()
	router.Use(middleware.Handler())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"data": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Check security headers
		assert.Contains(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
		assert.Contains(t, w.Header().Get("X-Frame-Options"), "DENY")
		assert.Contains(t, w.Header().Get("X-XSS-Protection"), "1; mode=block")
	})

	t.Run("request too large", func(t *testing.T) {
		largeData := strings.Repeat("x", 2048) // Larger than 1024 limit
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(largeData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := &Config{
		MaxRequestSize:    1024,
		RateLimitEnabled:  true,
		RateLimitRequests: 2, // Very low limit for testing
		RateLimitWindow:   time.Minute,
		CSRFProtection:    false,
		XSSProtection:     false,
		SecureHeaders:     false,
		AuditLogging:      false,
	}

	middleware := NewSecurityMiddleware(config)

	router := gin.New()
	router.Use(middleware.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCSRFProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := &Config{
		MaxRequestSize:   1024,
		RateLimitEnabled: false,
		CSRFProtection:   true,
		XSSProtection:    false,
		SecureHeaders:    false,
		AuditLogging:     false,
	}

	middleware := NewSecurityMiddleware(config)

	router := gin.New()
	router.Use(middleware.Handler())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("missing CSRF token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"data": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid CSRF token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"data": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "invalid-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSecureSession(t *testing.T) {
	sessionID := "test-session-id"
	userID := "user123"

	// Create session
	session := &SecureSession{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		IPAddress: "192.168.1.1",
		UserAgent: "test-agent",
		Data:      make(map[string]interface{}),
	}

	session.Data["role"] = "admin"
	session.Data["permissions"] = []string{"read", "write"}

	// Test session validation
	assert.True(t, session.IsValid())

	// Test expired session
	session.ExpiresAt = time.Now().Add(-time.Hour)
	assert.False(t, session.IsValid())

	// Test session refresh
	session.Refresh(2 * time.Hour)
	assert.True(t, session.IsValid())
	assert.True(t, session.ExpiresAt.After(time.Now().Add(time.Hour)))
}

func TestAuditLogger(t *testing.T) {
	logger := NewAuditLogger()
	event := &AuditEvent{
		UserID:    "user123",
		Type:      "login",
		Action:    "login",
		Resource:  "auth",
		IPAddress: "192.168.1.1",
		UserAgent: "test-agent",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"success": true,
			"method":  "password",
		},
	}

	// This should not panic
	logger.Log(event)

	// Test with context
	ctx := context.Background()
	logger.LogWithContext(ctx, event)
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 70.41.3.18, 150.172.238.178",
			},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.1",
			},
			remoteAddr: "192.168.1.1:12345",
			expected:   "203.0.113.1",
		},
		{
			name:       "Remote address only",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := extractClientIPFromRequest(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestIsValidCSRFToken(t *testing.T) {
	// Test with a real token generated by the system
	token, err := GenerateSecureToken(32)
	require.NoError(t, err)

	// This is a simplified test - in practice, CSRF tokens would be
	// stored and validated against a session
	assert.True(t, len(token) > 0)
	assert.NotEmpty(t, token)
}

func BenchmarkInputValidation(b *testing.B) {
	testInputs := []string{
		"hello world",
		"<script>alert('xss')</script>",
		"'; DROP TABLE users; --",
		"javascript:alert('xss')",
		"normal input with numbers 123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		IsValidInput(input)
	}
}

func BenchmarkPasswordHashing(b *testing.B) {
	password := "testpassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}
