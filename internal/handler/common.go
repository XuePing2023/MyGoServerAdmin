// Package api HTTP 接口层：参数绑定与响应，业务逻辑在 service。
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/response"
)

// pageParams 从 query 解析分页参数。
func pageParams(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	return page, size
}

// dateRange 解析 start/end 查询参数（格式 2006-01-02 或 RFC3339）。
// end 取当天 23:59:59.999 的下一刻（开区间）。
func dateRange(c *gin.Context) (time.Time, time.Time) {
	var start, end time.Time
	if s := c.Query("start"); s != "" {
		start = parseDate(s)
	}
	if e := c.Query("end"); e != "" {
		end = parseDate(e)
		if isDateOnly(e) {
			end = end.Add(24 * time.Hour)
		}
	}
	return start, end
}

func parseDate(s string) time.Time {
	layouts := []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

func isDateOnly(s string) bool {
	_, err := time.ParseInLocation("2006-01-02", s, time.Local)
	return err == nil
}

// requireIdentity 取登录身份，不存在时返回 false（已写响应）。
func requireIdentity(c *gin.Context) (*middleware.Identity, bool) {
	id := middleware.FromContext(c)
	if id == nil {
		response.Fail(c, http.StatusUnauthorized, "未登录")
		return nil, false
	}
	return id, true
}

// bindJSON 绑定请求体，失败时已写响应。
func bindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Handle(c, errs.BadRequest("参数错误: %v", err))
		return false
	}
	return true
}

// idsBody 批量 ID 请求体。
type idsBody struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

func pathID(c *gin.Context) string { return c.Param("id") }

// strconv0 解析可选整型参数，失败返回 0。
func strconv0(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
