package response

import (
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
)

// Errors 自定义错误类型
type Errors struct {
	code    int
	message string
}

// Code 获取错误码
func (e *Errors) Code() int {
	return e.code
}

// Error 实现error接口
func (e *Errors) Error() string {
	return e.message
}

// NewError 创建新的错误
func NewError(code int, message string) *Errors {
	return &Errors{
		code:    code,
		message: message,
	}
}

// 预定义错误码
const (
	// 通用错误码
	ErrCodeSuccess           = 200
	ErrCodeBadRequest        = 400
	ErrCodeUnauthorized      = 401
	ErrCodeForbidden         = 403
	ErrCodeNotFound          = 404
	ErrCodeInternalError     = 500

	// 业务错误码 (1000-1999)
	ErrCodeInvalidParams     = 1001
	ErrCodeUserNotFound      = 1002
	ErrCodeUserExists        = 1003
	ErrCodePasswordError     = 1004
	ErrCodeTokenExpired      = 1005
	ErrCodeTokenInvalid      = 1006
	ErrCodePermissionDenied  = 1007
	ErrCodeResourceNotFound  = 1008
	ErrCodeOperationFailed   = 1009
	ErrCodeRateLimit         = 1010
)

// 预定义错误
var (
	// 通用错误
	ErrSuccess          = NewError(ErrCodeSuccess, "success")
	ErrBadRequest       = NewError(ErrCodeBadRequest, "bad request")
	ErrUnauthorized     = NewError(ErrCodeUnauthorized, "unauthorized")
	ErrForbidden        = NewError(ErrCodeForbidden, "forbidden")
	ErrNotFound         = NewError(ErrCodeNotFound, "not found")
	ErrInternalError    = NewError(ErrCodeInternalError, "internal server error")

	// 业务错误
	ErrInvalidParams    = NewError(ErrCodeInvalidParams, "invalid parameters")
	ErrUserNotFound     = NewError(ErrCodeUserNotFound, "user not found")
	ErrUserExists       = NewError(ErrCodeUserExists, "user already exists")
	ErrPasswordError    = NewError(ErrCodePasswordError, "password error")
	ErrTokenExpired     = NewError(ErrCodeTokenExpired, "token expired")
	ErrTokenInvalid     = NewError(ErrCodeTokenInvalid, "token invalid")
	ErrPermissionDenied = NewError(ErrCodePermissionDenied, "permission denied")
	ErrResourceNotFound = NewError(ErrCodeResourceNotFound, "resource not found")
	ErrOperationFailed  = NewError(ErrCodeOperationFailed, "operation failed")
	ErrRateLimit        = NewError(ErrCodeRateLimit, "rate limit exceeded")
)

// GetErrorFromCode 根据错误码获取错误
func GetErrorFromCode(code int) error {
	switch code {
	case ErrCodeSuccess:
		return ErrSuccess
	case ErrCodeBadRequest:
		return ErrBadRequest
	case ErrCodeUnauthorized:
		return ErrUnauthorized
	case ErrCodeForbidden:
		return ErrForbidden
	case ErrCodeNotFound:
		return ErrNotFound
	case ErrCodeInternalError:
		return ErrInternalError
	case ErrCodeInvalidParams:
		return ErrInvalidParams
	case ErrCodeUserNotFound:
		return ErrUserNotFound
	case ErrCodeUserExists:
		return ErrUserExists
	case ErrCodePasswordError:
		return ErrPasswordError
	case ErrCodeTokenExpired:
		return ErrTokenExpired
	case ErrCodeTokenInvalid:
		return ErrTokenInvalid
	case ErrCodePermissionDenied:
		return ErrPermissionDenied
	case ErrCodeResourceNotFound:
		return ErrResourceNotFound
	case ErrCodeOperationFailed:
		return ErrOperationFailed
	case ErrCodeRateLimit:
		return ErrRateLimit
	default:
		return errors.New("unknown error")
	}
}

// IsErrorCode 检查错误是否为特定错误码
func IsErrorCode(err error, code int) bool {
	if customErr, ok := err.(*Errors); ok {
		return customErr.code == code
	}
	return false
}

// GetErrorCode 获取错误的错误码
func GetErrorCode(err error) int {
	if customErr, ok := err.(*Errors); ok {
		return customErr.code
	}
	return ErrCodeInternalError
}

// Error 错误响应
func Error(c *app.RequestContext, err error) {
	if customErr, ok := err.(*Errors); ok {
		FailWithError(c, customErr)
	} else {
		Fail(c, err)
	}
}

// WithDetails 为错误添加详细信息
func (e *Errors) WithDetails(details string) *Errors {
	return &Errors{
		code:    e.code,
		message: e.message + ": " + details,
	}
}