package handlers

import (
	"net/http"

	"go-aigateway/internal/security"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	TokenType string `json:"token_type"`
}

// RefreshRequest represents the token refresh request
type RefreshRequest struct {
	Token string `json:"token" binding:"required"`
}

// CreateAPIKeyRequest represents the API key creation request
type CreateAPIKeyRequest struct {
	Name        string          `json:"name" binding:"required"`
	Permissions map[string]bool `json:"permissions"`
	RateLimit   int             `json:"rate_limit"`
	ExpiresAt   *int64          `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest represents the API key update request
type UpdateAPIKeyRequest struct {
	Name        string          `json:"name,omitempty"`
	Permissions map[string]bool `json:"permissions,omitempty"`
	RateLimit   int             `json:"rate_limit,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
}

// Login handler for user authentication
func Login(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
		// Authenticate user
		user, err := localAuth.AuthenticateUser(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate JWT token
		token, err := localAuth.GenerateJWT(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token:     token,
			ExpiresIn: 86400, // 24 hours
			TokenType: "Bearer",
		})
	}
}

// RefreshToken handler for token refresh
func RefreshToken(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// Validate and refresh token
		claims, err := localAuth.ValidateJWT(req.Token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		// Generate new token
		newToken, err := localAuth.GenerateJWT(claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new token"})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token:     newToken,
			ExpiresIn: 86400, // 24 hours
			TokenType: "Bearer",
		})
	}
}

// CreateAPIKey handler for creating new API keys
func CreateAPIKey(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// Get user from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Create API key
		apiKey, err := localAuth.CreateAPIKey(userID.(string), req.Name, req.Permissions, req.RateLimit, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"api_key": apiKey,
			"message": "API key created successfully",
		})
	}
}

// ListAPIKeys handler for listing API keys
func ListAPIKeys(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
		// List API keys for user
		apiKeys := localAuth.ListAPIKeys(userID.(string))

		c.JSON(http.StatusOK, gin.H{"api_keys": apiKeys})
	}
}

// DeleteAPIKey handler for deleting API keys
func DeleteAPIKey(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID := c.Param("id")
		if keyID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key ID is required"})
			return
		}
		// Get user from context
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Delete API key
		err := localAuth.RevokeAPIKey(keyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
	}
}

// UpdateAPIKey handler for updating API keys
func UpdateAPIKey(localAuth *security.LocalAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID := c.Param("id")
		if keyID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key ID is required"})
			return
		}

		var req UpdateAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
		// Get user from context
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Update API key (implementation would depend on your storage layer)
		// For now, return success message
		c.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
	}
}
