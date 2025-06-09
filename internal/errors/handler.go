package errors

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler provides centralized error handling with recovery mechanisms
type ErrorHandler struct {
	logger         *logrus.Logger
	retryAttempts  int
	retryDelay     time.Duration
	circuitBreaker *CircuitBreaker
	healthChecker  *HealthChecker
}

// CircuitBreaker implements circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitState
}

type CircuitState int

const (
	Closed CircuitState = iota
	Open
	HalfOpen
)

// HealthChecker monitors service health
type HealthChecker struct {
	healthEndpoints map[string]string
	checkInterval   time.Duration
	timeout         time.Duration
	serviceStatus   map[string]bool
}

// NewErrorHandler creates a new enhanced error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		logger:        logrus.New(),
		retryAttempts: 3,
		retryDelay:    time.Second,
		circuitBreaker: &CircuitBreaker{
			failureThreshold: 5,
			resetTimeout:     30 * time.Second,
			state:            Closed,
		},
		healthChecker: &HealthChecker{
			healthEndpoints: make(map[string]string),
			checkInterval:   30 * time.Second,
			timeout:         5 * time.Second,
			serviceStatus:   make(map[string]bool),
		},
	}
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code        string      `json:"code"`
	Message     string      `json:"message"`
	Details     interface{} `json:"details,omitempty"`
	TraceID     string      `json:"trace_id,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	RetryAfter  *int        `json:"retry_after,omitempty"`
	Suggestions []string    `json:"suggestions,omitempty"`
}

// RecoveryInfo contains information about recovery attempts
type RecoveryInfo struct {
	Attempt     int           `json:"attempt"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	LastError   string        `json:"last_error"`
}

// HandleError processes different error types and returns appropriate HTTP responses
func (eh *ErrorHandler) HandleError(c *gin.Context, err error) {
	var errorResponse ErrorResponse
	var statusCode int

	switch e := err.(type) {
	case *AppError:
		statusCode = eh.getHTTPStatusCode(e.Code)
		errorResponse = ErrorResponse{
			Error: ErrorDetail{
				Code:        string(e.Code),
				Message:     e.Message,
				Details:     e.Details,
				TraceID:     c.GetString("trace_id"),
				Timestamp:   time.Now(),
				Suggestions: eh.getSuggestions(e.Code),
			},
		}
	default:
		statusCode = http.StatusInternalServerError
		errorResponse = ErrorResponse{
			Error: ErrorDetail{
				Code:        "INTERNAL_ERROR",
				Message:     "An internal error occurred",
				TraceID:     c.GetString("trace_id"),
				Timestamp:   time.Now(),
				Suggestions: []string{"Contact support if the problem persists"},
			},
		}
	}

	// Log the error with context
	eh.logError(c, err, statusCode)

	// Record error in circuit breaker
	eh.circuitBreaker.recordFailure()

	c.JSON(statusCode, errorResponse)
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func (eh *ErrorHandler) RetryWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < eh.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := eh.retryDelay * time.Duration(1<<uint(attempt-1))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// Check circuit breaker
		if !eh.circuitBreaker.allowRequest() {
			return fmt.Errorf("circuit breaker is open, service unavailable")
		}

		err := operation()
		if err == nil {
			eh.circuitBreaker.recordSuccess()
			return nil
		}

		lastErr = err

		// Don't retry for certain error types
		if !eh.shouldRetry(err) {
			break
		}

		eh.logger.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"error":   err.Error(),
		}).Warn("Operation failed, retrying...")
	}

	eh.circuitBreaker.recordFailure()
	return fmt.Errorf("operation failed after %d attempts: %w", eh.retryAttempts, lastErr)
}

// shouldRetry determines if an error is retryable
func (eh *ErrorHandler) shouldRetry(err error) bool {
	switch e := err.(type) {
	case *AppError:
		return e.Code == ErrCodeTimeout || e.Code == ErrCodeNetwork
	default:
		return false
	}
}

// CircuitBreaker methods
func (cb *CircuitBreaker) allowRequest() bool {
	now := time.Now()

	switch cb.state {
	case Closed:
		return true
	case Open:
		if now.Sub(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = HalfOpen
			cb.failureCount = 0
			return true
		}
		return false
	case HalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0
	cb.state = Closed
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = Open
	}
}

// HealthChecker methods
func (hc *HealthChecker) StartHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAllServices()
		}
	}
}

func (hc *HealthChecker) checkAllServices() {
	for serviceName, endpoint := range hc.healthEndpoints {
		healthy := hc.checkServiceHealth(endpoint)
		hc.serviceStatus[serviceName] = healthy

		if !healthy {
			logrus.WithField("service", serviceName).Warn("Service health check failed")
		}
	}
}

func (hc *HealthChecker) checkServiceHealth(endpoint string) bool {
	client := &http.Client{Timeout: hc.timeout}

	resp, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (hc *HealthChecker) AddService(name, healthEndpoint string) {
	hc.healthEndpoints[name] = healthEndpoint
	hc.serviceStatus[name] = true // Assume healthy initially
}

func (hc *HealthChecker) IsServiceHealthy(name string) bool {
	status, exists := hc.serviceStatus[name]
	return exists && status
}

// Recovery middleware for panic recovery
func (eh *ErrorHandler) RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				eh.logger.WithFields(logrus.Fields{
					"panic":      r,
					"stack":      string(debug.Stack()),
					"request_id": c.GetString("request_id"),
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
				}).Error("Panic recovered")

				// Return error response
				errorResponse := ErrorResponse{
					Error: ErrorDetail{
						Code:        "INTERNAL_ERROR",
						Message:     "An unexpected error occurred",
						TraceID:     c.GetString("trace_id"),
						Timestamp:   time.Now(),
						Suggestions: []string{"Contact support with the trace ID"},
					},
				}

				c.JSON(http.StatusInternalServerError, errorResponse)
				c.Abort()
			}
		}()

		c.Next()
	}
}

// getSuggestions provides helpful suggestions based on error codes
func (eh *ErrorHandler) getSuggestions(code ErrorCode) []string {
	switch code {
	case ErrCodeAuthentication:
		return []string{
			"Check if your API key is correct",
			"Ensure the API key is not expired",
			"Verify the API key has proper permissions",
		}
	case ErrCodeRateLimit:
		return []string{
			"Reduce request frequency",
			"Implement request queuing",
			"Contact support to increase rate limits",
		}
	case ErrCodeNetwork:
		return []string{
			"Check service status page",
			"Retry after a few minutes",
			"Implement exponential backoff",
		}
	case ErrCodeTimeout:
		return []string{
			"Increase timeout settings",
			"Check network connectivity",
			"Reduce request payload size",
		}
	default:
		return []string{"Check API documentation", "Contact support if the issue persists"}
	}
}

// logError logs errors with appropriate context
func (eh *ErrorHandler) logError(c *gin.Context, err error, statusCode int) {
	logLevel := logrus.InfoLevel
	if statusCode >= 500 {
		logLevel = logrus.ErrorLevel
	} else if statusCode >= 400 {
		logLevel = logrus.WarnLevel
	}

	eh.logger.WithFields(logrus.Fields{
		"error":      err.Error(),
		"status":     statusCode,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
		"trace_id":   c.GetString("trace_id"),
		"request_id": c.GetString("request_id"),
	}).Log(logLevel, "Request error")
}

// getHTTPStatusCode maps error codes to HTTP status codes
func (eh *ErrorHandler) getHTTPStatusCode(code ErrorCode) int {
	switch code {
	case ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeAuthentication:
		return http.StatusUnauthorized
	case ErrCodeAuthorization:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrCodeNetwork:
		return http.StatusBadGateway
	case ErrCodeDatabase:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
