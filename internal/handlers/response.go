package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StandardResponse 标准API响应格式
type StandardResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// ErrorInfo 错误信息结构
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SuccessResponse 返回成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, StandardResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// ErrorResponse 返回错误响应
func ErrorResponse(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().Unix(),
	})
}

// ValidationErrorResponse 返回验证错误响应
func ValidationErrorResponse(c *gin.Context, message string, details interface{}) {
	ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", message, details)
}

// NotFoundErrorResponse 返回未找到错误响应
func NotFoundErrorResponse(c *gin.Context, resource string) {
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", resource+" not found", nil)
}

// InternalServerErrorResponse 返回内部服务器错误响应
func InternalServerErrorResponse(c *gin.Context, message string, details interface{}) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, details)
}

// UnauthorizedErrorResponse 返回未授权错误响应
func UnauthorizedErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
}

// ForbiddenErrorResponse 返回禁止访问错误响应
func ForbiddenErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message, nil)
}
