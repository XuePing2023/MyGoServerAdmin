package middleware

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/logger"
	"serveradmin/internal/service"
)

// bodyWriter 包装 ResponseWriter，捕获响应体以提取业务码。
type bodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	if w.buf.Len() < 4096 {
		w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// moduleMapping 路径前缀 -> 模块名。
var moduleMapping = []struct {
	prefix string
	name   string
}{
	{"/api/v1/users", "用户管理"},
	{"/api/v1/roles", "角色管理"},
	{"/api/v1/menus", "菜单管理"},
	{"/api/v1/departments", "部门管理"},
	{"/api/v1/dict-types", "字典管理"},
	{"/api/v1/dict-items", "字典管理"},
	{"/api/v1/configs", "参数配置"},
	{"/api/v1/notices", "通知公告"},
	{"/api/v1/files", "文件管理"},
	{"/api/v1/oplogs", "操作日志"},
	{"/api/v1/loginlogs", "登录日志"},
	{"/api/v1/online", "在线用户"},
	{"/api/v1/jobs", "定时任务"},
	{"/api/v1/auth", "认证中心"},
}

// actionMapping HTTP 方法 -> 动作。
var actionMapping = map[string]string{
	http.MethodPost:   "新增",
	http.MethodPut:    "修改",
	http.MethodPatch:  "修改",
	http.MethodDelete: "删除",
}

var (
	sensitiveKeys = regexp.MustCompile(`(?i)"(password|oldPassword|newPassword|captchaCode|secret)"\s*:\s*"[^"]*"`)
	bizCodeRe     = regexp.MustCompile(`"code"\s*:\s*(-?\d+)`)
)

// maskSensitive 对请求体中的敏感字段脱敏。
func maskSensitive(body string) string {
	return sensitiveKeys.ReplaceAllString(body, `"$1":"***"`)
}

// OpLog 操作日志中间件：记录所有非 GET 请求（异步落库，不阻塞请求）。
func OpLog(cfg *config.Config, logSvc *service.LogService) gin.HandlerFunc {
	if !cfg.OpLog.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	ch := make(chan *model.OperationLog, 512)
	go func() {
		for log := range ch {
			logSvc.RecordOperation(log)
		}
	}()

	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if c.Request.Method == http.MethodGet || strings.HasPrefix(p, "/uploads") {
			c.Next()
			return
		}

		// 读取请求体后还原，交给后续 handler
		var bodyStr string
		if c.Request.Body != nil {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(cfg.OpLog.BodyMaxBytes)+1))
			if err == nil {
				if len(raw) > cfg.OpLog.BodyMaxBytes {
					bodyStr = string(raw[:cfg.OpLog.BodyMaxBytes]) + "...(截断)"
				} else {
					bodyStr = string(raw)
				}
				c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
			}
		}

		bw := &bodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw
		start := time.Now()
		c.Next()

		// 登录/刷新已有专门的登录日志，避免重复记录
		if p == "/api/v1/auth/login" || p == "/api/v1/auth/refresh" {
			return
		}

		module := "其他"
		for _, m := range moduleMapping {
			if strings.HasPrefix(p, m.prefix) {
				module = m.name
				break
			}
		}
		action := actionMapping[c.Request.Method]
		if action == "" {
			action = c.Request.Method
		}
		if p == "/api/v1/auth/logout" {
			action = "登出"
		}

		id := FromContext(c)
		entry := &model.OperationLog{
			Username:  usernameOf(id),
			UserID:    userIDOf(id),
			Module:    module,
			Action:    action,
			Method:    c.Request.Method,
			Path:      p,
			Query:     c.Request.URL.RawQuery,
			Body:      maskSensitive(bodyStr),
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			HTTPCode:  bw.Status(),
			CostMs:    time.Since(start).Milliseconds(),
		}
		if m := bizCodeRe.FindSubmatch(bw.buf.Bytes()); m != nil {
			if code, err := strconv.Atoi(string(m[1])); err == nil {
				entry.BizCode = code
			}
		}
		entry.PrepareCreate()

		select {
		case ch <- entry:
		default:
			logger.L.Warn("操作日志队列已满，丢弃一条日志")
		}
	}
}

func userIDOf(id *Identity) string {
	if id == nil {
		return ""
	}
	return id.UserID
}

func usernameOf(id *Identity) string {
	if id == nil {
		return ""
	}
	return id.Username
}

// AccessLog 访问日志中间件。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		p := c.Request.URL.Path
		c.Next()
		if p == "/healthz" || p == "/readyz" {
			return
		}
		logger.L.Infof("%s %s %d %s %s",
			c.Request.Method, p, c.Writer.Status(),
			time.Since(start).Round(time.Millisecond), c.ClientIP())
	}
}

// Recovery panic 恢复中间件（zap 记录）。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.L.Errorf("panic: %v, path=%s", r, c.Request.URL.Path)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
