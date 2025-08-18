package onedockclient

import (
	"fmt"
	"net/http"
)

// APIError API 错误类型
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error 实现 error 接口
func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// IsNotFound 检查是否为 404 错误
func (e *APIError) IsNotFound() bool {
	return e.Code == http.StatusNotFound
}

// IsBadRequest 检查是否为 400 错误
func (e *APIError) IsBadRequest() bool {
	return e.Code == http.StatusBadRequest
}

// IsUnauthorized 检查是否为 401 错误
func (e *APIError) IsUnauthorized() bool {
	return e.Code == http.StatusUnauthorized
}

// IsForbidden 检查是否为 403 错误
func (e *APIError) IsForbidden() bool {
	return e.Code == http.StatusForbidden
}

// IsServerError 检查是否为服务器错误 (5xx)
func (e *APIError) IsServerError() bool {
	return e.Code >= 500 && e.Code < 600
}

// IsClientError 检查是否为客户端错误 (4xx)
func (e *APIError) IsClientError() bool {
	return e.Code >= 400 && e.Code < 500
}

// NetworkError 网络错误
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// TimeoutError 超时错误
type TimeoutError struct {
	Operation string
	Timeout   string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout error: %s operation timed out after %s", e.Operation, e.Timeout)
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ConfigError 配置错误
type ConfigError struct {
	Parameter string
	Message   string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error for parameter '%s': %s", e.Parameter, e.Message)
}

// 常见错误构造函数

// NewAPIError 创建 API 错误
func NewAPIError(code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewNetworkError 创建网络错误
func NewNetworkError(err error) *NetworkError {
	return &NetworkError{
		Err: err,
	}
}

// NewValidationError 创建验证错误
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
