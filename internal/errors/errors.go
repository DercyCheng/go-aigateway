// Package errors provides enhanced error handling utilities
package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents error categories
type ErrorCode string

const (
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeAuthentication ErrorCode = "AUTH_ERROR"
	ErrCodeAuthorization  ErrorCode = "AUTHZ_ERROR"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeTimeout        ErrorCode = "TIMEOUT_ERROR"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT_ERROR"
	ErrCodeNetwork        ErrorCode = "NETWORK_ERROR"
	ErrCodeDatabase       ErrorCode = "DATABASE_ERROR"
	ErrCodeResource       ErrorCode = "RESOURCE_ERROR"
	ErrCodeSecurity       ErrorCode = "SECURITY_ERROR"
)

// AppError represents application-specific error with enhanced context
type AppError struct {
	Code      ErrorCode   `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	File      string      `json:"file,omitempty"`
	Line      int         `json:"line,omitempty"`
	Cause     error       `json:"-"`
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error type
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Code == t.Code
	}
	return false
}

// New creates a new AppError with caller information
func New(code ErrorCode, message string) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// NewWithDetails creates a new AppError with additional details
func NewWithDetails(code ErrorCode, message string, details interface{}) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// Wrap creates a new AppError wrapping an existing error
func Wrap(code ErrorCode, message string, err error) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Cause:     err,
	}
}

// WrapWithDetails creates a new AppError wrapping an existing error with details
func WrapWithDetails(code ErrorCode, message string, details interface{}, err error) *AppError {
	_, file, line, _ := runtime.Caller(1)
	return &AppError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Cause:     err,
	}
}

// WithDetails adds details to an existing AppError
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// SecurityError creates a security-related error
func SecurityError(message string, details interface{}) *AppError {
	return NewWithDetails(ErrCodeSecurity, message, details)
}

// ResourceError creates a resource-related error
func ResourceError(message string, details interface{}) *AppError {
	return NewWithDetails(ErrCodeResource, message, details)
}
