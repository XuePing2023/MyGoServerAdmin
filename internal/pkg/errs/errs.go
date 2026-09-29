// Package errs 定义统一的业务错误类型，service 层返回 *Error，
// 由 response.Handle 转换成统一的 JSON 响应。
package errs

import (
	"errors"
	"fmt"
)

// 常用业务码。
const (
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// Error 业务错误。
type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Error) Error() string { return fmt.Sprintf("[%d] %s", e.Code, e.Msg) }

// New 构造业务错误。
func New(code int, msg string) *Error { return &Error{Code: code, Msg: msg} }

func BadRequest(format string, args ...any) *Error {
	return New(CodeBadRequest, fmt.Sprintf(format, args...))
}

func Unauthorized(format string, args ...any) *Error {
	return New(CodeUnauthorized, fmt.Sprintf(format, args...))
}

func Forbidden(format string, args ...any) *Error {
	return New(CodeForbidden, fmt.Sprintf(format, args...))
}

func NotFound(format string, args ...any) *Error {
	return New(CodeNotFound, fmt.Sprintf(format, args...))
}

func Server(format string, args ...any) *Error {
	return New(CodeServerError, fmt.Sprintf(format, args...))
}

// From 将任意 error 尝试转换为 *Error，方便判断。
func From(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
