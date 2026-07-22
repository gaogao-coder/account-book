package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Error   int         `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Error struct {
	StatusCode int
	Code       int
	Message    string
	Internal   error
}

// Error 返回内部错误或对外错误信息。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Internal != nil {
		return e.Internal.Error()
	}
	return e.Message
}

// Success 写入统一成功响应。
func Success(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, Response{
		Error:   0,
		Message: "success",
		Data:    data,
	})
}

// Failure 将业务错误写入统一失败响应。
func Failure(ctx *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := http.StatusInternalServerError
	message := "服务内部错误"
	internalErr := err

	var apiErr *Error
	if errors.As(err, &apiErr) {
		status = apiErr.StatusCode
		code = apiErr.Code
		message = apiErr.Message
		internalErr = apiErr.Internal
	}
	if internalErr != nil && status >= http.StatusInternalServerError {
		log.Printf("request failed: %v", internalErr)
	}
	ctx.JSON(status, Response{
		Error:   code,
		Message: message,
		Data:    nil,
	})
}

// InvalidArgument 构造参数错误。
func InvalidArgument(message string) error {
	return &Error{
		StatusCode: http.StatusBadRequest,
		Code:       http.StatusBadRequest,
		Message:    message,
	}
}

// PermissionDenied 构造权限不足错误。
func PermissionDenied(message string) error {
	return &Error{
		StatusCode: http.StatusForbidden,
		Code:       http.StatusForbidden,
		Message:    message,
	}
}

// Conflict 构造资源冲突错误。
func Conflict(message string) error {
	return &Error{
		StatusCode: http.StatusConflict,
		Code:       http.StatusConflict,
		Message:    message,
	}
}

// Unauthorized 构造未授权错误。
func Unauthorized(message string) error {
	return &Error{
		StatusCode: http.StatusUnauthorized,
		Code:       http.StatusUnauthorized,
		Message:    message,
	}
}

// NotFound 构造资源不存在错误。
func NotFound(message string) error {
	return &Error{
		StatusCode: http.StatusNotFound,
		Code:       http.StatusNotFound,
		Message:    message,
	}
}

// DependencyUnavailable 构造依赖不可用错误。
func DependencyUnavailable(message string, err error) error {
	return &Error{
		StatusCode: http.StatusServiceUnavailable,
		Code:       http.StatusServiceUnavailable,
		Message:    message,
		Internal:   err,
	}
}

// UpstreamFailure 构造上游服务失败错误。
func UpstreamFailure(message string, err error) error {
	return &Error{
		StatusCode: http.StatusBadGateway,
		Code:       http.StatusBadGateway,
		Message:    message,
		Internal:   err,
	}
}

// InternalFailure 构造服务内部错误。
func InternalFailure(message string, err error) error {
	return &Error{
		StatusCode: http.StatusInternalServerError,
		Code:       http.StatusInternalServerError,
		Message:    message,
		Internal:   err,
	}
}
