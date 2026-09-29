// Package middleware Gin 中间件：CORS、JWT 认证、权限校验、访问日志、操作日志、异常恢复。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/config"
)

// Cors 跨域中间件。
func Cors(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		for _, o := range cfg.CORS.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization,X-Requested-With")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// headerPrefix Authorization 头前缀。
const headerPrefix = "Bearer "

// BearerToken 从请求头提取 Bearer 令牌。
func BearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		// 兼容 WebSocket / 下载链接无法携带请求头的场景
		if t := c.Query("access_token"); t != "" {
			return t
		}
		return ""
	}
	if len(h) > len(headerPrefix) && strings.EqualFold(h[:len(headerPrefix)], headerPrefix) {
		return h[len(headerPrefix):]
	}
	return ""
}
