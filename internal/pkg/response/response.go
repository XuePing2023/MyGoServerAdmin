// Package response 统一 API 响应格式：
//
//	{ "code": 0, "msg": "ok", "data": ... }
//
// code=0 表示成功；业务失败时 code 为 400/403/404/500 等，HTTP 状态码同步。
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/logger"
)

// Body 统一响应体。
type Body struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// PageResult 分页数据。
type PageResult struct {
	List  any   `json:"list"`
	Total int64 `json:"total"`
}

// OK 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "ok", Data: data})
}

// Page 分页成功响应。
func Page(c *gin.Context, list any, total int64) {
	OK(c, PageResult{List: list, Total: total})
}

// Fail 失败响应。code 若在 400-599 之间则作为 HTTP 状态码。
func Fail(c *gin.Context, code int, msg string) {
	status := code
	if status < 400 || status > 599 {
		status = http.StatusOK
	}
	c.JSON(status, Body{Code: code, Msg: msg})
}

// Handle 统一错误处理：业务错误原样返回，未知错误记日志并返回 500。
func Handle(c *gin.Context, err error) {
	if err == nil {
		OK(c, nil)
		return
	}
	var be *errs.Error
	if errors.As(err, &be) {
		Fail(c, be.Code, be.Msg)
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		Fail(c, http.StatusNotFound, "数据不存在")
		return
	}
	if logger.L != nil {
		logger.L.Errorf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	Fail(c, http.StatusInternalServerError, "系统内部错误")
}
