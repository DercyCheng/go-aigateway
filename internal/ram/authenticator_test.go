package ram

import (
	"context"
	"encoding/base64"
	"fmt"
	"go-aigateway/internal/config"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRAMAuthenticator(t *testing.T) {
	t.Run("enabled config", func(t *testing.T) {
		cfg := &config.RAMAuthConfig{
			Enabled:         true,
			AccessKeySecret: "test-secret",
			Region:          "us-west-1",
			CacheExpiration: time.Hour,
		}

		auth := NewRAMAuthenticator(cfg)
		assert.NotNil(t, auth)
		assert.Equal(t, cfg, auth.config)
		assert.NotNil(t, auth.cache)
	})

	t.Run("disabled config", func(t *testing.T) {
		cfg := &config.RAMAuthConfig{
			Enabled: false,
		}

		auth := NewRAMAuthenticator(cfg)
		assert.Nil(t, auth)
	})
}

func TestRAMAuthenticator_Authenticate(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("nil authenticator", func(t *testing.T) {
		var nilAuth *RAMAuthenticator

		req := &AuthRequest{
			AccessKeyID: "test-key",
			Signature:   "test-signature",
			Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		}

		resp, err := nilAuth.Authenticate(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("valid authentication", func(t *testing.T) {
		accessKeyID := "LTAI4test123456"
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		req := &AuthRequest{
			AccessKeyID:     accessKeyID,
			Timestamp:       timestamp,
			Nonce:           "test-nonce",
			Method:          "POST",
			URI:             "/api/v1/chat",
			Headers:         map[string]string{"Content-Type": "application/json"},
			QueryParameters: map[string]string{},
		}

		// Calculate valid signature
		canonicalString := auth.buildCanonicalString(req)
		signature := auth.calculateSignature(canonicalString)
		req.Signature = signature

		resp, err := auth.Authenticate(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Authenticated)
		assert.NotNil(t, resp.UserInfo)
		assert.Equal(t, fmt.Sprintf("user-%s", accessKeyID[len(accessKeyID)-8:]), resp.UserInfo.UserID)
	})

	t.Run("invalid signature", func(t *testing.T) {
		req := &AuthRequest{
			AccessKeyID:     "LTAI4invalid123456",
			Signature:       "invalid-signature",
			Timestamp:       strconv.FormatInt(time.Now().Unix(), 10),
			Nonce:           "test-nonce",
			Method:          "POST",
			URI:             "/api/v1/chat",
			Headers:         map[string]string{"Content-Type": "application/json"},
			QueryParameters: map[string]string{},
		}

		resp, err := auth.Authenticate(context.Background(), req)
		require.NoError(t, err)
		assert.False(t, resp.Authenticated)
		assert.Equal(t, "Invalid signature", resp.Error)
	})

	t.Run("expired timestamp", func(t *testing.T) {
		// Use timestamp from 10 minutes ago
		oldTimestamp := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)

		req := &AuthRequest{
			AccessKeyID:     "LTAI4expired123456",
			Signature:       "test-signature",
			Timestamp:       oldTimestamp,
			Nonce:           "test-nonce",
			Method:          "POST",
			URI:             "/api/v1/chat",
			Headers:         map[string]string{"Content-Type": "application/json"},
			QueryParameters: map[string]string{},
		}

		resp, err := auth.Authenticate(context.Background(), req)
		require.NoError(t, err)
		assert.False(t, resp.Authenticated)
		assert.Equal(t, "Request timestamp expired", resp.Error)
	})

	t.Run("cached authentication", func(t *testing.T) {
		accessKeyID := "LTAI4cached123"

		// Set up cache entry
		userInfo := &UserInfo{
			UserID:   "cached-user",
			UserName: "cached_user",
			Roles:    []string{"ai-gateway-user"},
		}

		auth.setCache(accessKeyID, &CacheEntry{
			UserInfo:  userInfo,
			ExpiresAt: time.Now().Add(time.Hour),
		})

		req := &AuthRequest{
			AccessKeyID: accessKeyID,
			Signature:   "any-signature",
			Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		}

		resp, err := auth.Authenticate(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Authenticated)
		assert.Equal(t, "cached-user", resp.UserInfo.UserID)
	})
}

func TestRAMAuthenticator_validateSignature(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	req := &AuthRequest{
		AccessKeyID:     "LTAI4test123456",
		Method:          "POST",
		URI:             "/api/v1/chat",
		Headers:         map[string]string{"Content-Type": "application/json"},
		QueryParameters: map[string]string{"param1": "value1"},
		Timestamp:       strconv.FormatInt(time.Now().Unix(), 10),
	}

	t.Run("valid signature", func(t *testing.T) {
		canonicalString := auth.buildCanonicalString(req)
		validSignature := auth.calculateSignature(canonicalString)
		req.Signature = validSignature

		valid := auth.validateSignature(req)
		assert.True(t, valid)
	})

	t.Run("invalid signature", func(t *testing.T) {
		req.Signature = "invalid-signature"

		valid := auth.validateSignature(req)
		assert.False(t, valid)
	})
}

func TestRAMAuthenticator_buildCanonicalString(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	req := &AuthRequest{
		Method: "POST",
		URI:    "/api/v1/chat",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "test-value",
		},
		QueryParameters: map[string]string{
			"param2": "value2",
			"param1": "value1",
		},
		Timestamp: "1234567890",
	}

	canonicalString := auth.buildCanonicalString(req)

	// Verify the canonical string contains expected components
	assert.Contains(t, canonicalString, "POST")
	assert.Contains(t, canonicalString, "/api/v1/chat")
	assert.Contains(t, canonicalString, "1234567890")

	// Headers and query parameters should be sorted
	assert.Contains(t, canonicalString, "content-type:application/json")
	assert.Contains(t, canonicalString, "x-custom:test-value")
	assert.Contains(t, canonicalString, "param1=value1&param2=value2")
}

func TestRAMAuthenticator_calculateSignature(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	canonicalString := "POST\n/api/v1/chat\ncontent-type:application/json\nparam1=value1\n1234567890"

	signature := auth.calculateSignature(canonicalString)

	// Verify signature is base64 encoded
	_, err := base64.StdEncoding.DecodeString(signature)
	assert.NoError(t, err)

	// Verify signature is deterministic
	signature2 := auth.calculateSignature(canonicalString)
	assert.Equal(t, signature, signature2)

	// Verify signature changes with different input
	signature3 := auth.calculateSignature("different-string")
	assert.NotEqual(t, signature, signature3)
}

func TestRAMAuthenticator_validateTimestamp(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("valid current timestamp", func(t *testing.T) {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		valid := auth.validateTimestamp(timestamp)
		assert.True(t, valid)
	})

	t.Run("timestamp within allowed skew", func(t *testing.T) {
		// 2 minutes ago (within 5 minute limit)
		timestamp := strconv.FormatInt(time.Now().Add(-2*time.Minute).Unix(), 10)
		valid := auth.validateTimestamp(timestamp)
		assert.True(t, valid)
	})

	t.Run("timestamp too old", func(t *testing.T) {
		// 10 minutes ago (beyond 5 minute limit)
		timestamp := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		valid := auth.validateTimestamp(timestamp)
		assert.False(t, valid)
	})

	t.Run("timestamp too far in future", func(t *testing.T) {
		// 10 minutes in the future (beyond 5 minute limit)
		timestamp := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)
		valid := auth.validateTimestamp(timestamp)
		assert.False(t, valid)
	})

	t.Run("invalid timestamp format", func(t *testing.T) {
		valid := auth.validateTimestamp("invalid-timestamp")
		assert.False(t, valid)
	})
}

func TestRAMAuthenticator_getUserInfo(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("regular user", func(t *testing.T) {
		accessKeyID := "LTAI4test123456"

		userInfo, err := auth.getUserInfo(context.Background(), accessKeyID)
		require.NoError(t, err)
		assert.Equal(t, "user-st123456", userInfo.UserID)
		assert.Equal(t, "user_st123456", userInfo.UserName)
		assert.Contains(t, userInfo.Roles, "ai-gateway-user")
		assert.Contains(t, userInfo.Permissions, "ai:chat")
	})

	t.Run("admin user", func(t *testing.T) {
		accessKeyID := "LTAI4admintest123"

		userInfo, err := auth.getUserInfo(context.Background(), accessKeyID)
		require.NoError(t, err)
		assert.Contains(t, userInfo.Roles, "ai-gateway-admin")
		assert.Contains(t, userInfo.Permissions, "ai:admin")
	})
}

func TestRAMAuthenticator_CheckPermission(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("nil user info", func(t *testing.T) {
		hasPermission := auth.CheckPermission(nil, "ai", "chat")
		assert.False(t, hasPermission)
	})

	t.Run("admin user has all permissions", func(t *testing.T) {
		userInfo := &UserInfo{
			UserID: "admin-user",
			Roles:  []string{"ai-gateway-admin"},
		}

		hasPermission := auth.CheckPermission(userInfo, "ai", "chat")
		assert.True(t, hasPermission)

		hasPermission = auth.CheckPermission(userInfo, "admin", "delete")
		assert.True(t, hasPermission)
	})

	t.Run("user with specific permission", func(t *testing.T) {
		userInfo := &UserInfo{
			UserID:      "regular-user",
			Roles:       []string{"ai-gateway-user"},
			Permissions: []string{"ai:chat", "ai:completion"},
		}

		hasPermission := auth.CheckPermission(userInfo, "ai", "chat")
		assert.True(t, hasPermission)

		hasPermission = auth.CheckPermission(userInfo, "ai", "admin")
		assert.False(t, hasPermission)
	})

	t.Run("user with wildcard permission", func(t *testing.T) {
		userInfo := &UserInfo{
			UserID:      "power-user",
			Roles:       []string{"ai-gateway-user"},
			Permissions: []string{"ai*"},
		}

		hasPermission := auth.CheckPermission(userInfo, "ai", "chat")
		assert.True(t, hasPermission)

		hasPermission = auth.CheckPermission(userInfo, "admin", "delete")
		assert.False(t, hasPermission)
	})
}

func TestRAMAuthenticator_Cache(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	accessKeyID := "test-key"
	userInfo := &UserInfo{
		UserID:   "test-user",
		UserName: "test_user",
	}

	t.Run("set and get cache", func(t *testing.T) {
		entry := &CacheEntry{
			UserInfo:  userInfo,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		auth.setCache(accessKeyID, entry)

		cached := auth.getFromCache(accessKeyID)
		require.NotNil(t, cached)
		assert.Equal(t, userInfo.UserID, cached.UserInfo.UserID)
	})

	t.Run("expired cache entry", func(t *testing.T) {
		entry := &CacheEntry{
			UserInfo:  userInfo,
			ExpiresAt: time.Now().Add(-time.Hour), // Expired
		}

		auth.setCache(accessKeyID, entry)

		cached := auth.getFromCache(accessKeyID)
		assert.Nil(t, cached) // Should return nil for expired entries
	})

	t.Run("clear cache", func(t *testing.T) {
		entry := &CacheEntry{
			UserInfo:  userInfo,
			ExpiresAt: time.Now().Add(time.Hour),
		}

		auth.setCache(accessKeyID, entry)

		// Verify entry exists
		cached := auth.getFromCache(accessKeyID)
		assert.NotNil(t, cached)

		// Clear cache
		auth.ClearCache()

		// Verify entry is gone
		cached = auth.getFromCache(accessKeyID)
		assert.Nil(t, cached)
	})
}

func TestRAMAuthenticator_ValidateRequest(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("nil authenticator", func(t *testing.T) {
		var nilAuth *RAMAuthenticator

		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
		valid, err := nilAuth.ValidateRequest(req, "test-key", "signature", "123456789")

		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("invalid timestamp format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
		valid, err := auth.ValidateRequest(req, "test-key", "signature", "invalid")

		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("expired timestamp", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", nil)
		oldTimestamp := strconv.FormatInt(time.Now().Add(-20*time.Minute).Unix(), 10)

		valid, err := auth.ValidateRequest(req, "test-key", "signature", oldTimestamp)

		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "timestamp expired")
	})

	t.Run("valid request with correct signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat?param1=value1", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Custom", "test-value")

		timestamp := strconv.FormatInt(time.Now().Unix(), 10)

		// Build auth request to calculate correct signature
		authReq := &AuthRequest{
			AccessKeyID: "test-key",
			Timestamp:   timestamp,
			Method:      req.Method,
			URI:         req.URL.Path,
			Headers: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "test-value",
			},
			QueryParameters: map[string]string{
				"param1": "value1",
			},
		}

		canonicalString := auth.buildCanonicalString(authReq)
		signature := auth.calculateSignature(canonicalString)

		valid, err := auth.ValidateRequest(req, "test-key", signature, timestamp)

		assert.NoError(t, err)
		assert.True(t, valid)
	})
}

func TestRAMAuthenticator_extractAuthRequest(t *testing.T) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)
	require.NotNil(t, auth)

	t.Run("request with RAM auth headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat?param1=value1", nil)
		req.Header.Set("X-Ca-Key", "test-access-key")
		req.Header.Set("X-Ca-Signature", "test-signature")
		req.Header.Set("X-Ca-Timestamp", "1234567890")
		req.Header.Set("X-Ca-Nonce", "test-nonce")
		req.Header.Set("Content-Type", "application/json")

		authReq := auth.extractAuthRequest(req)

		require.NotNil(t, authReq)
		assert.Equal(t, "test-access-key", authReq.AccessKeyID)
		assert.Equal(t, "test-signature", authReq.Signature)
		assert.Equal(t, "1234567890", authReq.Timestamp)
		assert.Equal(t, "test-nonce", authReq.Nonce)
		assert.Equal(t, "POST", authReq.Method)
		assert.Equal(t, "/api/v1/chat", authReq.URI)
		assert.Equal(t, "application/json", authReq.Headers["Content-Type"])
		assert.Equal(t, "value1", authReq.QueryParameters["param1"])
	})

	t.Run("request without RAM auth headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

		authReq := auth.extractAuthRequest(req)
		assert.Nil(t, authReq)
	})
}

func BenchmarkRAMAuthenticator_validateSignature(b *testing.B) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)

	req := &AuthRequest{
		AccessKeyID:     "LTAI4test123456",
		Method:          "POST",
		URI:             "/api/v1/chat",
		Headers:         map[string]string{"Content-Type": "application/json"},
		QueryParameters: map[string]string{"param1": "value1"},
		Timestamp:       strconv.FormatInt(time.Now().Unix(), 10),
	}

	canonicalString := auth.buildCanonicalString(req)
	req.Signature = auth.calculateSignature(canonicalString)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.validateSignature(req)
	}
}

func BenchmarkRAMAuthenticator_buildCanonicalString(b *testing.B) {
	cfg := &config.RAMAuthConfig{
		Enabled:         true,
		AccessKeySecret: "test-secret-key",
		Region:          "us-west-1",
		CacheExpiration: time.Hour,
	}

	auth := NewRAMAuthenticator(cfg)

	req := &AuthRequest{
		Method: "POST",
		URI:    "/api/v1/chat",
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"X-Custom":      "test-value",
			"Authorization": "Bearer token",
		},
		QueryParameters: map[string]string{
			"param1": "value1",
			"param2": "value2",
			"param3": "value3",
		},
		Timestamp: "1234567890",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.buildCanonicalString(req)
	}
}
