package errors

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ImprovedErrorHandler provides enhanced error handling
type ImprovedErrorHandler struct {
	logger      *logrus.Logger
	environment string
	enableDebug bool
}

// NewImprovedErrorHandler creates an enhanced error handler
func NewImprovedErrorHandler(environment string, enableDebug bool) *ImprovedErrorHandler {
	return &ImprovedErrorHandler{
		logger:      logrus.New(),
		environment: environment,
		enableDebug: enableDebug,
	}
}

// ErrorContext provides additional context for errors
type ErrorContext struct {
	RequestID   string                 `json:"request_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string][]string    `json:"query_params,omitempty"`
	Stack       []string               `json:"stack,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EnhancedErrorResponse provides structured error responses
type EnhancedErrorResponse struct {
	Error ErrorDetail  `json:"error"`
	Meta  ErrorContext `json:"meta,omitempty"`
}

// ContextualErrorMiddleware adds request context to all errors
func (eh *ImprovedErrorHandler) ContextualErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := generateRequestID()
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		// Store request start time
		c.Set("start_time", time.Now())

		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			eh.handleRequestErrors(c, requestID)
		}
	}
}

// PanicRecoveryMiddleware with enhanced recovery and logging
func (eh *ImprovedErrorHandler) PanicRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := eh.getRequestID(c)

		// Get stack trace
		stack := make([]byte, 4096)
		length := runtime.Stack(stack, false)
		stackTrace := string(stack[:length])

		// Log panic with full context
		eh.logger.WithFields(logrus.Fields{
			"request_id":  requestID,
			"panic":       recovered,
			"stack_trace": stackTrace,
			"client_ip":   c.ClientIP(),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"user_agent":  c.GetHeader("User-Agent"),
		}).Error("Panic recovered")

		// Send structured error response
		errorCtx := eh.buildErrorContext(c, requestID)
		if eh.enableDebug {
			errorCtx.Stack = parseStackTrace(stackTrace)
		}

		response := EnhancedErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: "An unexpected error occurred",
				TraceID: requestID,
			},
			Meta: errorCtx,
		}

		c.JSON(http.StatusInternalServerError, response)
		c.Abort()
	})
}

// TimeoutMiddleware handles request timeouts gracefully
func (eh *ImprovedErrorHandler) TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Next()
		}()

		select {
		case <-done:
			// Request completed successfully
		case <-ctx.Done():
			// Request timed out
			requestID := eh.getRequestID(c)

			eh.logger.WithFields(logrus.Fields{
				"request_id": requestID,
				"timeout":    timeout,
				"client_ip":  c.ClientIP(),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
			}).Warn("Request timeout")

			c.JSON(http.StatusRequestTimeout, EnhancedErrorResponse{
				Error: ErrorDetail{
					Code:    "REQUEST_TIMEOUT",
					Message: fmt.Sprintf("Request timeout after %v", timeout),
					TraceID: requestID,
				},
				Meta: eh.buildErrorContext(c, requestID),
			})
			c.Abort()
		}
	}
}

// CircuitBreakerMiddleware implements circuit breaker pattern
func (eh *ImprovedErrorHandler) CircuitBreakerMiddleware(threshold int, timeout time.Duration) gin.HandlerFunc {
	failures := 0
	lastFailTime := time.Time{}
	isOpen := false

	return func(c *gin.Context) {
		if isOpen {
			if time.Since(lastFailTime) > timeout {
				// Half-open state: try one request
				isOpen = false
				failures = 0
			} else {
				// Circuit is open, reject request
				requestID := eh.getRequestID(c)
				c.JSON(http.StatusServiceUnavailable, EnhancedErrorResponse{
					Error: ErrorDetail{
						Code:    "SERVICE_UNAVAILABLE",
						Message: "Service temporarily unavailable",
						TraceID: requestID,
					},
					Meta: eh.buildErrorContext(c, requestID),
				})
				c.Abort()
				return
			}
		}

		c.Next()

		// Check if request failed
		if c.Writer.Status() >= 500 {
			failures++
			lastFailTime = time.Now()

			if failures >= threshold {
				isOpen = true
				eh.logger.WithFields(logrus.Fields{
					"failures":  failures,
					"threshold": threshold,
				}).Warn("Circuit breaker opened")
			}
		} else if c.Writer.Status() < 400 {
			// Reset on success
			failures = 0
		}
	}
}

// handleRequestErrors processes errors accumulated during request processing
func (eh *ImprovedErrorHandler) handleRequestErrors(c *gin.Context, requestID string) {
	for _, ginErr := range c.Errors {
		eh.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"error":      ginErr.Error(),
			"type":       ginErr.Type,
			"meta":       ginErr.Meta,
		}).Error("Request error")
	}
}

// buildErrorContext creates error context from request
func (eh *ImprovedErrorHandler) buildErrorContext(c *gin.Context, requestID string) ErrorContext {
	ctx := ErrorContext{
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	// Add user ID if available
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			ctx.UserID = uid
		}
	}

	// Add debug information in development
	if eh.enableDebug {
		ctx.Headers = make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				ctx.Headers[key] = values[0]
			}
		}
		ctx.QueryParams = c.Request.URL.Query()
	}

	return ctx
}

// getRequestID extracts request ID from context
func (eh *ImprovedErrorHandler) getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return generateRequestID()
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// parseStackTrace converts stack trace string to slice
func parseStackTrace(stackTrace string) []string {
	// Implementation to parse stack trace into readable format
	// This is a simplified version
	return []string{stackTrace}
}
