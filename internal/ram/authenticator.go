package ram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-aigateway/internal/config"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type RAMAuthenticator struct {
	config *config.RAMAuthConfig
	cache  map[string]*CacheEntry
	mutex  sync.RWMutex
}

type CacheEntry struct {
	UserInfo  *UserInfo
	ExpiresAt time.Time
}

type UserInfo struct {
	UserID       string            `json:"user_id"`
	UserName     string            `json:"user_name"`
	Roles        []string          `json:"roles"`
	Permissions  []string          `json:"permissions"`
	Policies     []string          `json:"policies"`
	Attributes   map[string]string `json:"attributes"`
}

type AuthRequest struct {
	AccessKeyID     string            `json:"access_key_id"`
	Signature       string            `json:"signature"`
	Timestamp       string            `json:"timestamp"`
	Nonce           string            `json:"nonce"`
	Method          string            `json:"method"`
	URI             string            `json:"uri"`
	Headers         map[string]string `json:"headers"`
	QueryParameters map[string]string `json:"query_parameters"`
}

type AuthResponse struct {
	Authenticated bool      `json:"authenticated"`
	UserInfo      *UserInfo `json:"user_info,omitempty"`
	Error         string    `json:"error,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

func NewRAMAuthenticator(cfg *config.RAMAuthConfig) *RAMAuthenticator {
	if !cfg.Enabled {
		return nil
	}

	return &RAMAuthenticator{
		config: cfg,
		cache:  make(map[string]*CacheEntry),
	}
}

func (ra *RAMAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	if ra == nil {
		return nil, fmt.Errorf("RAM authentication not enabled")
	}

	// Check cache first
	if cached := ra.getFromCache(req.AccessKeyID); cached != nil {
		logrus.WithField("access_key_id", req.AccessKeyID).Debug("Using cached authentication")
		return &AuthResponse{
			Authenticated: true,
			UserInfo:      cached.UserInfo,
			ExpiresAt:     cached.ExpiresAt,
		}, nil
	}

	// Validate signature
	if !ra.validateSignature(req) {
		return &AuthResponse{
			Authenticated: false,
			Error:         "Invalid signature",
		}, nil
	}

	// Validate timestamp (prevent replay attacks)
	if !ra.validateTimestamp(req.Timestamp) {
		return &AuthResponse{
			Authenticated: false,
			Error:         "Request timestamp expired",
		}, nil
	}

	// Get user info from RAM
	userInfo, err := ra.getUserInfo(ctx, req.AccessKeyID)
	if err != nil {
		return &AuthResponse{
			Authenticated: false,
			Error:         fmt.Sprintf("Failed to get user info: %v", err),
		}, nil
	}

	// Cache the result
	expiresAt := time.Now().Add(ra.config.CacheExpiration)
	ra.setCache(req.AccessKeyID, &CacheEntry{
		UserInfo:  userInfo,
		ExpiresAt: expiresAt,
	})

	return &AuthResponse{
		Authenticated: true,
		UserInfo:      userInfo,
		ExpiresAt:     expiresAt,
	}, nil
}

func (ra *RAMAuthenticator) validateSignature(req *AuthRequest) bool {
	// Build canonical string
	canonicalString := ra.buildCanonicalString(req)
	
	// Calculate expected signature
	expectedSignature := ra.calculateSignature(canonicalString)
	
	// Compare signatures
	return hmac.Equal([]byte(req.Signature), []byte(expectedSignature))
}

func (ra *RAMAuthenticator) buildCanonicalString(req *AuthRequest) string {
	var parts []string
	
	// HTTP method
	parts = append(parts, strings.ToUpper(req.Method))
	
	// URI
	parts = append(parts, req.URI)
	
	// Canonical query string
	if len(req.QueryParameters) > 0 {
		var queryParts []string
		for k, v := range req.QueryParameters {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
		}
		sort.Strings(queryParts)
		parts = append(parts, strings.Join(queryParts, "&"))
	} else {
		parts = append(parts, "")
	}
	
	// Canonical headers
	if len(req.Headers) > 0 {
		var headerParts []string
		for k, v := range req.Headers {
			headerParts = append(headerParts, fmt.Sprintf("%s:%s", strings.ToLower(k), strings.TrimSpace(v)))
		}
		sort.Strings(headerParts)
		parts = append(parts, strings.Join(headerParts, "\n"))
	} else {
		parts = append(parts, "")
	}
	
	// Timestamp and nonce
	parts = append(parts, req.Timestamp)
	parts = append(parts, req.Nonce)
	
	return strings.Join(parts, "\n")
}

func (ra *RAMAuthenticator) calculateSignature(canonicalString string) string {
	h := hmac.New(sha256.New, []byte(ra.config.AccessKeySecret))
	h.Write([]byte(canonicalString))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (ra *RAMAuthenticator) validateTimestamp(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	
	requestTime := time.Unix(ts, 0)
	now := time.Now()
	
	// Allow 5 minutes clock skew
	return now.Sub(requestTime) <= 5*time.Minute && requestTime.Sub(now) <= 5*time.Minute
}

func (ra *RAMAuthenticator) getUserInfo(ctx context.Context, accessKeyID string) (*UserInfo, error) {
	// In a real implementation, this would make a call to Aliyun RAM API
	// For now, we'll return mock data based on the access key
	
	logrus.WithField("access_key_id", accessKeyID).Info("Fetching user info from RAM")
	
	// Mock user info based on access key pattern
	userInfo := &UserInfo{
		UserID:   fmt.Sprintf("user-%s", accessKeyID[len(accessKeyID)-8:]),
		UserName: fmt.Sprintf("user_%s", accessKeyID[len(accessKeyID)-8:]),
		Roles:    []string{"ai-gateway-user"},
		Permissions: []string{
			"ai:chat",
			"ai:completion",
			"ai:models",
		},
		Policies: []string{
			"AIGatewayUserPolicy",
		},
		Attributes: map[string]string{
			"region":      ra.config.Region,
			"access_key":  accessKeyID,
			"auth_method": "ram",
		},
	}
	
	// Add admin permissions for specific access keys
	if strings.HasPrefix(accessKeyID, "LTAI") && strings.Contains(accessKeyID, "admin") {
		userInfo.Roles = append(userInfo.Roles, "ai-gateway-admin")
		userInfo.Permissions = append(userInfo.Permissions, "ai:admin", "ai:metrics")
		userInfo.Policies = append(userInfo.Policies, "AIGatewayAdminPolicy")
	}
	
	return userInfo, nil
}

func (ra *RAMAuthenticator) getFromCache(accessKeyID string) *CacheEntry {
	ra.mutex.RLock()
	defer ra.mutex.RUnlock()
	
	entry, exists := ra.cache[accessKeyID]
	if !exists {
		return nil
	}
	
	if time.Now().After(entry.ExpiresAt) {
		delete(ra.cache, accessKeyID)
		return nil
	}
	
	return entry
}

func (ra *RAMAuthenticator) setCache(accessKeyID string, entry *CacheEntry) {
	ra.mutex.Lock()
	defer ra.mutex.Unlock()
	
	ra.cache[accessKeyID] = entry
}

func (ra *RAMAuthenticator) ClearCache() {
	ra.mutex.Lock()
	defer ra.mutex.Unlock()
	
	ra.cache = make(map[string]*CacheEntry)
}

func (ra *RAMAuthenticator) CheckPermission(userInfo *UserInfo, resource, action string) bool {
	if userInfo == nil {
		return false
	}
	
	// Check if user has admin role
	for _, role := range userInfo.Roles {
		if role == "ai-gateway-admin" {
			return true
		}
	}
	
	// Check specific permissions
	requiredPermission := fmt.Sprintf("%s:%s", resource, action)
	for _, permission := range userInfo.Permissions {
		if permission == requiredPermission || permission == resource+"*" {
			return true
		}
	}
	
	return false
}

// Middleware function to integrate with Gin
func (ra *RAMAuthenticator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ra == nil {
				next.ServeHTTP(w, r)
				return
			}
			
			// Extract RAM authentication info from request
			authReq := ra.extractAuthRequest(r)
			if authReq == nil {
				// No RAM auth info, continue with next handler
				next.ServeHTTP(w, r)
				return
			}
			
			// Perform authentication
			authResp, err := ra.Authenticate(r.Context(), authReq)
			if err != nil {
				http.Error(w, fmt.Sprintf("Authentication error: %v", err), http.StatusInternalServerError)
				return
			}
			
			if !authResp.Authenticated {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"message": authResp.Error,
						"type":    "authentication_error",
						"code":    "ram_auth_failed",
					},
				})
				return
			}
			
			// Add user info to request context
			ctx := context.WithValue(r.Context(), "ram_user_info", authResp.UserInfo)
			r = r.WithContext(ctx)
			
			next.ServeHTTP(w, r)
		})
	}
}

func (ra *RAMAuthenticator) extractAuthRequest(r *http.Request) *AuthRequest {
	// Check for RAM authentication headers
	accessKeyID := r.Header.Get("X-Ram-Access-Key-Id")
	signature := r.Header.Get("X-Ram-Signature")
	timestamp := r.Header.Get("X-Ram-Timestamp")
	nonce := r.Header.Get("X-Ram-Nonce")
	
	if accessKeyID == "" || signature == "" || timestamp == "" || nonce == "" {
		return nil
	}
	
	// Convert headers to map
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	
	// Convert query parameters to map
	queryParams := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}
	
	return &AuthRequest{
		AccessKeyID:     accessKeyID,
		Signature:       signature,
		Timestamp:       timestamp,
		Nonce:           nonce,
		Method:          r.Method,
		URI:             r.URL.Path,
		Headers:         headers,
		QueryParameters: queryParams,
	}
}

// ValidateRequest validates an HTTP request using RAM authentication
func (ra *RAMAuthenticator) ValidateRequest(r *http.Request, accessKeyID, signature, timestamp string) (bool, error) {
	if ra == nil {
		return false, fmt.Errorf("RAM authentication not enabled")
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp format")
	}

	// Check timestamp validity (within 15 minutes)
	now := time.Now().Unix()
	if abs(now-ts) > 900 {
		return false, fmt.Errorf("timestamp expired")
	}

	// Build auth request
	authReq := &AuthRequest{
		AccessKeyID: accessKeyID,
		Signature:   signature,
		Timestamp:   timestamp,
		Method:      r.Method,
		URI:         r.URL.Path,
		Headers:     make(map[string]string),
		QueryParameters: make(map[string]string),
	}

	// Convert headers to map
	for k, v := range r.Header {
		if len(v) > 0 {
			authReq.Headers[k] = v[0]
		}
	}

	// Convert query parameters to map
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			authReq.QueryParameters[k] = v[0]
		}
	}

	// Validate signature
	return ra.validateSignature(authReq), nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
