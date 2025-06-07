package errors

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger *logrus.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		logger: logrus.New(),
	}
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
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
				Code:    string(e.Code),
				Message: e.Message,
				Details: e.Details,
				TraceID: c.GetString("trace_id"),
			},
		}
		
		// Log the error with context
		eh.logger.WithFields(logrus.Fields{
			"error_code": e.Code,
			"message":    e.Message,
			"file":       e.File,
			"line":       e.Line,
			"trace_id":   c.GetString("trace_id"),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
		}).Error("Application error occurred")

	default:
		statusCode = http.StatusInternalServerError
		errorResponse = ErrorResponse{
			Error: ErrorDetail{
				Code:    string(ErrCodeInternal),
				Message: "Internal server error",
				TraceID: c.GetString("trace_id"),
			},
		}
		
		// Log unknown errors with stack trace
		eh.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"trace_id": c.GetString("trace_id"),
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"stack":    string(debug.Stack()),
		}).Error("Unexpected error occurred")
	}

	c.JSON(statusCode, errorResponse)
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

// RecoveryMiddleware provides panic recovery with proper error handling
func (eh *ErrorHandler) RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		err := Wrap(ErrCodeInternal, "Panic recovered", recovered.(error))
		eh.HandleError(c, err)
		c.Abort()
	})
}

// ValidationError creates a validation error
func ValidationError(message string, details interface{}) *AppError {
	return NewWithDetails(ErrCodeValidation, message, details)
}

// AuthenticationError creates an authentication error
func AuthenticationError(message string) *AppError {
	return New(ErrCodeAuthentication, message)
}

// AuthorizationError creates an authorization error
func AuthorizationError(message string) *AppError {
	return New(ErrCodeAuthorization, message)
}

// NotFoundError creates a not found error
func NotFoundError(resource string) *AppError {
	return New(ErrCodeNotFound, "Resource not found: "+resource)
}

// TimeoutError creates a timeout error
func TimeoutError(operation string) *AppError {
	return New(ErrCodeTimeout, "Operation timeout: "+operation)
}

// RateLimitError creates a rate limit error
func RateLimitError(message string) *AppError {
	return New(ErrCodeRateLimit, message)
}
