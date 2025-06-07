package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-aigateway/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// LocalAuthenticator provides local authentication without external dependencies
type LocalAuthenticator struct {
	config    *config.SecurityConfig
	apiKeys   map[string]*APIKeyInfo
	sessions  map[string]*SessionInfo
	users     map[string]*UserInfo
	mutex     sync.RWMutex
	jwtSecret []byte
}

// APIKeyInfo represents an API key
type APIKeyInfo struct {
	ID          string            `json:"id"`
	KeyHash     string            `json:"key_hash"`
	Name        string            `json:"name"`
	UserID      string            `json:"user_id"`
	Permissions []string          `json:"permissions"`
	RateLimit   int               `json:"rate_limit"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	LastUsed    *time.Time        `json:"last_used,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UserInfo represents a user
type UserInfo struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	LastLogin   *time.Time        `json:"last_login,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SessionInfo represents an active session
type SessionInfo struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	LastSeen  time.Time `json:"last_seen"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// Claims represents JWT claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// NewLocalAuthenticator creates a new local authenticator
func NewLocalAuthenticator(cfg *config.SecurityConfig) *LocalAuthenticator {
	jwtSecret := []byte(cfg.JWTSecret)
	if len(jwtSecret) == 0 {
		// Generate a random secret if none provided
		jwtSecret = make([]byte, 32)
		rand.Read(jwtSecret)
		logrus.Warn("No JWT secret provided, using randomly generated secret. This should not be used in production!")
	}

	auth := &LocalAuthenticator{
		config:    cfg,
		apiKeys:   make(map[string]*APIKeyInfo),
		sessions:  make(map[string]*SessionInfo),
		users:     make(map[string]*UserInfo),
		jwtSecret: jwtSecret,
	}

	// Initialize with default admin user if none exists
	auth.initializeDefaultUsers()

	return auth
}

// initializeDefaultUsers creates default users if none exist
func (la *LocalAuthenticator) initializeDefaultUsers() {
	// Create default admin user
	adminUser := &UserInfo{
		ID:          "admin",
		Username:    "admin",
		Email:       "admin@localhost",
		Roles:       []string{"admin", "user"},
		Permissions: []string{"*"}, // All permissions
		Active:      true,
		CreatedAt:   time.Now(),
		Metadata:    map[string]string{"type": "default"},
	}

	// Create default API user
	apiUser := &UserInfo{
		ID:          "api-user",
		Username:    "api-user",
		Email:       "api@localhost",
		Roles:       []string{"api-user"},
		Permissions: []string{"ai:chat", "ai:completion", "ai:models"},
		Active:      true,
		CreatedAt:   time.Now(),
		Metadata:    map[string]string{"type": "api"},
	}

	la.users[adminUser.ID] = adminUser
	la.users[apiUser.ID] = apiUser

	// Create default API keys
	la.createDefaultAPIKeys()
}

// createDefaultAPIKeys creates default API keys for initial setup
func (la *LocalAuthenticator) createDefaultAPIKeys() {
	// Default admin API key
	adminKey, err := la.GenerateAPIKey("admin", "Default Admin Key", []string{"*"}, 0)
	if err != nil {
		logrus.WithError(err).Error("Failed to create default admin API key")
	} else {
		logrus.WithField("key_prefix", adminKey[:10]+"...").Info("Created default admin API key")
	}

	// Default API user key
	userKey, err := la.GenerateAPIKey("api-user", "Default API User Key", []string{"ai:chat", "ai:completion", "ai:models"}, 100)
	if err != nil {
		logrus.WithError(err).Error("Failed to create default API user key")
	} else {
		logrus.WithField("key_prefix", userKey[:10]+"...").Info("Created default API user key")
	}
}

// GenerateAPIKey generates a new API key for a user
func (la *LocalAuthenticator) GenerateAPIKey(userID, name string, permissions []string, rateLimit int) (string, error) {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	// Check if user exists
	user, exists := la.users[userID]
	if !exists {
		return "", fmt.Errorf("user not found: %s", userID)
	}

	// Check API key limit
	userKeyCount := 0
	for _, key := range la.apiKeys {
		if key.UserID == userID {
			userKeyCount++
		}
	}

	if userKeyCount >= la.config.MaxAPIKeys {
		return "", fmt.Errorf("maximum API keys reached for user: %s", userID)
	}

	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	apiKey := la.config.APIKeyPrefix + hex.EncodeToString(keyBytes)
	keyHash := la.hashAPIKey(apiKey)

	// Create API key info
	keyInfo := &APIKeyInfo{
		ID:          generateID(),
		KeyHash:     keyHash,
		Name:        name,
		UserID:      userID,
		Permissions: permissions,
		RateLimit:   rateLimit,
		CreatedAt:   time.Now(),
		Metadata: map[string]string{
			"user_email": user.Email,
			"user_roles": strings.Join(user.Roles, ","),
		},
	}

	la.apiKeys[keyHash] = keyInfo

	logrus.WithFields(logrus.Fields{
		"user_id":     userID,
		"key_name":    name,
		"permissions": permissions,
	}).Info("Generated new API key")

	return apiKey, nil
}

// ValidateAPIKey validates an API key and returns user information
func (la *LocalAuthenticator) ValidateAPIKey(apiKey string) (*UserInfo, *APIKeyInfo, error) {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	keyHash := la.hashAPIKey(apiKey)
	keyInfo, exists := la.apiKeys[keyHash]
	if !exists {
		return nil, nil, fmt.Errorf("invalid API key")
	}

	// Check if key is expired
	if keyInfo.ExpiresAt != nil && time.Now().After(*keyInfo.ExpiresAt) {
		return nil, nil, fmt.Errorf("API key expired")
	}

	// Get user info
	user, exists := la.users[keyInfo.UserID]
	if !exists {
		return nil, nil, fmt.Errorf("user not found for API key")
	}

	// Check if user is active
	if !user.Active {
		return nil, nil, fmt.Errorf("user account is disabled")
	}

	// Update last used timestamp (do this in a separate goroutine to avoid blocking)
	go func() {
		la.mutex.Lock()
		now := time.Now()
		keyInfo.LastUsed = &now
		la.mutex.Unlock()
	}()

	return user, keyInfo, nil
}

// GenerateJWT generates a JWT token for a user
func (la *LocalAuthenticator) GenerateJWT(userID string) (string, error) {
	la.mutex.RLock()
	user, exists := la.users[userID]
	la.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("user not found: %s", userID)
	}

	// Create claims
	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(la.config.TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ai-gateway",
			Subject:   userID,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(la.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns claims
func (la *LocalAuthenticator) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return la.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	return claims, nil
}

// CheckPermission checks if a user has a specific permission
func (la *LocalAuthenticator) CheckPermission(userID, resource, action string) bool {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	user, exists := la.users[userID]
	if !exists || !user.Active {
		return false
	}

	// Check if user has admin role (full access)
	for _, role := range user.Roles {
		if role == "admin" {
			return true
		}
	}

	// Check specific permission
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)
	for _, permission := range user.Permissions {
		if permission == "*" || permission == requiredPermission || permission == resource+":*" {
			return true
		}
	}

	return false
}

// RevokeAPIKey revokes an API key
func (la *LocalAuthenticator) RevokeAPIKey(apiKey string) error {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	keyHash := la.hashAPIKey(apiKey)
	if _, exists := la.apiKeys[keyHash]; !exists {
		return fmt.Errorf("API key not found")
	}

	delete(la.apiKeys, keyHash)
	logrus.WithField("key_hash", keyHash[:10]+"...").Info("Revoked API key")

	return nil
}

// ListAPIKeys returns all API keys for a user
func (la *LocalAuthenticator) ListAPIKeys(userID string) []*APIKeyInfo {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	var keys []*APIKeyInfo
	for _, key := range la.apiKeys {
		if key.UserID == userID {
			// Don't include the actual key hash in the response
			keyCopy := *key
			keyCopy.KeyHash = keyCopy.KeyHash[:10] + "..." // Show only prefix
			keys = append(keys, &keyCopy)
		}
	}

	return keys
}

// hashAPIKey creates a hash of the API key for storage
func (la *LocalAuthenticator) hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CleanupExpiredSessions removes expired sessions
func (la *LocalAuthenticator) CleanupExpiredSessions() {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	now := time.Now()
	for id, session := range la.sessions {
		if now.After(session.ExpiresAt) {
			delete(la.sessions, id)
		}
	}
}

// StartCleanupTask starts a background task to clean up expired sessions
func (la *LocalAuthenticator) StartCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			la.CleanupExpiredSessions()
		}
	}
}

// AuthenticateUser authenticates a user with username and password
func (la *LocalAuthenticator) AuthenticateUser(username, password string) (*UserInfo, error) {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	// For now, we'll use simple hardcoded authentication
	// In a real implementation, you would hash and compare passwords
	var user *UserInfo
	for _, u := range la.users {
		if u.Username == username && u.Active {
			// Simple password check - in production, use proper password hashing
			if (username == "admin" && password == "admin123") ||
				(username == "apiuser" && password == "api123") {
				user = u
				break
			}
		}
	}

	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// CreateAPIKey creates a new API key for a user with enhanced options
func (la *LocalAuthenticator) CreateAPIKey(userID, name string, permissions map[string]bool, rateLimit int, expiresAt *int64) (string, error) {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	// Check if user exists
	_, exists := la.users[userID]
	if !exists {
		return "", fmt.Errorf("user not found: %s", userID)
	}

	// Convert permissions map to slice
	permSlice := make([]string, 0, len(permissions))
	for perm, enabled := range permissions {
		if enabled {
			permSlice = append(permSlice, perm)
		}
	}

	// Generate API key
	apiKey, err := la.GenerateAPIKey(userID, name, permSlice, rateLimit)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}
