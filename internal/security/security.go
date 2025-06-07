package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"go-aigateway/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// RateLimiter represents a simple rate limiter
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// IsAllowed checks if a request is allowed
func (rl *RateLimiter) IsAllowed(clientIP string) bool {
	now := time.Now()

	// Clean old entries
	if requests, exists := rl.requests[clientIP]; exists {
		filtered := make([]time.Time, 0)
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.window {
				filtered = append(filtered, reqTime)
			}
		}
		rl.requests[clientIP] = filtered
	}

	// Check if limit exceeded
	if len(rl.requests[clientIP]) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests[clientIP] = append(rl.requests[clientIP], now)
	return true
}

// Config represents security configuration
type Config struct {
	MaxRequestSize     int64
	CSRFProtection     bool
	CSRFEnabled        bool
	XSSProtection      bool
	ContentTypeNoSniff bool
	SecureHeaders      bool
	AuditLogging       bool
	HSTSMaxAge         int
	RateLimitEnabled   bool
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	SessionTimeout     time.Duration
	SessionSecure      bool
	SessionSameSite    http.SameSite
}

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	config      *Config
	logger      *logrus.Logger
	rateLimiter *RateLimiter
	csrfTokens  map[string]time.Time
	auditLogger *AuditLogger
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *Config) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:      config,
		logger:      logrus.New(),
		rateLimiter: NewRateLimiter(config.RateLimitRequests, config.RateLimitWindow),
		csrfTokens:  make(map[string]time.Time),
		auditLogger: NewAuditLogger(),
	}
}

// Middleware returns the Gin middleware function
func (sm *SecurityMiddleware) Middleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Add security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		if sm.config.HSTSMaxAge > 0 {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", sm.config.HSTSMaxAge))
		}

		// Content Security Policy
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'"
		c.Header("Content-Security-Policy", csp)

		c.Next()
	})
}

// Handler returns the Gin middleware function for security
func (sm *SecurityMiddleware) Handler() gin.HandlerFunc {
	return sm.Middleware()
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		if cfg.HSTSMaxAge > 0 {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", cfg.HSTSMaxAge))
		}

		// Content Security Policy
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'"
		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}

// RequestSizeLimit middleware limits request size
func RequestSizeLimit(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"message": "Request entity too large",
					"type":    "security_error",
					"code":    "request_too_large",
				},
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// InputSanitizer sanitizes and validates input
type InputSanitizer struct {
	logger *logrus.Logger
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{
		logger: logrus.New(),
	}
}

// SanitizeString removes potentially dangerous characters
func (is *InputSanitizer) SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except tab, newline, and carriage return
	var result strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// ValidateEmail validates email format
func (is *InputSanitizer) ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.ValidationError("Invalid email format", email)
	}
	return nil
}

// ValidateAPIKey validates API key format and strength
func (is *InputSanitizer) ValidateAPIKey(apiKey string) error {
	if len(apiKey) < 32 {
		return errors.ValidationError("API key too short", "minimum 32 characters required")
	}

	if len(apiKey) > 512 {
		return errors.ValidationError("API key too long", "maximum 512 characters allowed")
	}

	// Check for common patterns that might indicate weak keys
	if strings.Contains(strings.ToLower(apiKey), "password") ||
		strings.Contains(strings.ToLower(apiKey), "secret") ||
		regexp.MustCompile(`^[a-zA-Z0-9]{32,}$`).MatchString(apiKey) == false {
		return errors.ValidationError("API key format invalid", "key must contain alphanumeric characters")
	}

	return nil
}

// ValidateJSONStructure validates JSON structure for common injection patterns
func (is *InputSanitizer) ValidateJSONStructure(data interface{}) error {
	// Check for potential JSON injection patterns
	jsonStr := fmt.Sprintf("%v", data)

	dangerous := []string{
		"__proto__",
		"constructor",
		"prototype",
		"eval(",
		"function(",
		"javascript:",
		"<script",
		"</script>",
	}

	lowerData := strings.ToLower(jsonStr)
	for _, pattern := range dangerous {
		if strings.Contains(lowerData, pattern) {
			is.logger.WithField("pattern", pattern).Warn("Dangerous pattern detected in JSON")
			return errors.SecurityError("Potentially dangerous content detected", pattern)
		}
	}

	return nil
}

// PasswordHasher handles secure password operations
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost: 12, // bcrypt cost
	}
}

// HashPassword securely hashes a password
func (ph *PasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), ph.cost)
	if err != nil {
		return "", errors.Wrap(errors.ErrCodeSecurity, "Failed to hash password", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (ph *PasswordHasher) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SecureCompare performs constant-time string comparison
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.Wrap(errors.ErrCodeSecurity, "Failed to generate secure token", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SessionManager manages secure sessions
type SessionManager struct {
	sessions map[string]*Session
	logger   *logrus.Logger
}

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      map[string]interface{}
}

// SecureSession represents a secure user session with additional security features
type SecureSession struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
	Data      map[string]interface{}
}

// IsValid checks if the session is still valid
func (s *SecureSession) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// Refresh extends the session expiration time
func (s *SecureSession) Refresh(duration time.Duration) {
	s.ExpiresAt = time.Now().Add(duration)
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		logger:   logrus.New(),
	}
}

// CreateSession creates a new secure session
func (sm *SessionManager) CreateSession(userID string, duration time.Duration) (*Session, error) {
	sessionID, err := GenerateSecureToken(32)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeSecurity, "Failed to generate session ID", err)
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Data:      make(map[string]interface{}),
	}

	sm.sessions[sessionID] = session

	sm.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"user_id":    userID,
		"expires_at": session.ExpiresAt,
	}).Info("Session created")

	return session, nil
}

// ValidateSession validates and retrieves a session
func (sm *SessionManager) ValidateSession(sessionID string) (*Session, error) {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, errors.New(errors.ErrCodeAuthentication, "Invalid session")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sm.sessions, sessionID)
		return nil, errors.New(errors.ErrCodeAuthentication, "Session expired")
	}

	return session, nil
}

// DestroySession removes a session
func (sm *SessionManager) DestroySession(sessionID string) {
	delete(sm.sessions, sessionID)
	sm.logger.WithField("session_id", sessionID).Info("Session destroyed")
}

// CleanupExpiredSessions removes expired sessions
func (sm *SessionManager) CleanupExpiredSessions() {
	now := time.Now()
	for id, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, id)
		}
	}
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	UserID    string                 `json:"user_id"`
	IPAddress string                 `json:"ip_address"`
	Timestamp time.Time              `json:"timestamp"`
	RemoteIP  string                 `json:"remote_ip"`
	UserAgent string                 `json:"user_agent"`
	Details   map[string]interface{} `json:"details"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	logger *logrus.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		logger: logrus.New(),
	}
}

// Log logs an audit event
func (al *AuditLogger) Log(event *AuditEvent) {
	al.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"user_id":    event.UserID,
		"remote_ip":  event.RemoteIP,
		"user_agent": event.UserAgent,
		"details":    event.Details,
	}).Info("Security audit event")
}

// LogWithContext logs an audit event with context
func (al *AuditLogger) LogWithContext(ctx context.Context, event *AuditEvent) {
	al.logger.WithContext(ctx).WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"user_id":    event.UserID,
		"remote_ip":  event.RemoteIP,
		"user_agent": event.UserAgent,
		"details":    event.Details,
	}).Info("Security audit event")
}

// IsValidInput validates input against common security threats
func IsValidInput(input string) bool {
	if len(input) == 0 {
		return true
	}

	// Check for basic XSS patterns
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onload=",
		"onerror=",
		"onclick=",
		"onmouseover=",
		"eval(",
		"expression(",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(inputLower, pattern) {
			return false
		}
	}

	// Check for SQL injection patterns
	sqlPatterns := []string{
		"union select",
		"drop table",
		"delete from",
		"insert into",
		"update set",
		"' or '1'='1",
		"' or 1=1",
		"-- ",
		"/*",
		"*/",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(inputLower, pattern) {
			return false
		}
	}

	return true
}

// SanitizeInput sanitizes user input
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except for common whitespace
	var builder strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			continue
		}
		builder.WriteRune(r)
	}

	// Trim whitespace
	return strings.TrimSpace(builder.String())
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(hashedPassword, password string) bool {
	if len(hashedPassword) == 0 || len(password) == 0 {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// extractClientIP extracts the real client IP from the request
func extractClientIP(c *gin.Context) string {
	// Check various headers for real IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP",
	}

	for _, header := range headers {
		if ip := c.GetHeader(header); ip != "" {
			// X-Forwarded-For might contain multiple IPs
			if header == "X-Forwarded-For" {
				ips := strings.Split(ip, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}
			return ip
		}
	}

	// Fall back to remote address
	return c.ClientIP()
}

// extractClientIPFromRequest extracts the real client IP from an HTTP request (for testing)
func extractClientIPFromRequest(req *http.Request) string {
	// Check various headers for real IP
	headers := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Client-IP",
		"CF-Connecting-IP",
	}

	for _, header := range headers {
		if ip := req.Header.Get(header); ip != "" {
			// X-Forwarded-For might contain multiple IPs
			if header == "X-Forwarded-For" {
				ips := strings.Split(ip, ",")
				if len(ips) > 0 {
					return strings.TrimSpace(ips[0])
				}
			}
			return ip
		}
	}

	// Fall back to remote address
	if req.RemoteAddr != "" {
		// Remove port if present
		if colonIndex := strings.LastIndex(req.RemoteAddr, ":"); colonIndex > 0 {
			return req.RemoteAddr[:colonIndex]
		}
		return req.RemoteAddr
	}

	return ""
}
