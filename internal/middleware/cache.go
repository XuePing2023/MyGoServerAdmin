package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/cache"
)

// ResponseCache HTTP 响应缓存中间件（Cache-Aside）：
// HIT 直接返回 Redis 中的响应；MISS 放行到 Controller/MongoDB，
// 成功响应（HTTP 200 且 code=0）回填 Redis。
// routes 为显式配置表：路由模式（方法 + FullPath）-> 缓存规则，
// 未配置的路由一律不缓存（NONE）。
// 必须注册在 Auth 之后（USER/ROLE 策略需要登录身份）。
func ResponseCache(cc cache.Cache, ttl cache.TTLConfig, routes map[string]cache.Rule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cc == nil || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}
		rule, ok := routes[routeKey(c)]
		if !ok || rule.Policy == cache.PolicyNone {
			c.Next()
			return
		}

		key, valid := responseKey(c, rule)
		if !valid {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		if data, err := cc.Get(ctx, key); err == nil {
			c.Header("Cache-Control", "no-store")
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json; charset=utf-8", data)
			c.Abort()
			return
		}

		bw := &cacheBodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw
		c.Next()
		c.Header("Cache-Control", "no-store")
		c.Header("X-Cache", "MISS")

		if bw.Status() == http.StatusOK && bw.buf.Len() > 0 && isSuccessfulBody(bw.buf.Bytes()) {
			t := rule.TTL
			if t == 0 {
				t = ttl.TTLFor(rule.Policy)
			}
			_ = cc.Set(ctx, key, bw.buf.Bytes(), t)
		}
	}
}

// routeKey 策略表的 Key：方法 + 路由模式（c.FullPath()）。
func routeKey(c *gin.Context) string {
	return c.Request.Method + " " + c.FullPath()
}

// responseKey 按策略组装 Key：
//
//	PUBLIC -> serveradmin:v1:{scope}:{resource}[:{hash}]
//	USER   -> serveradmin:v1:user:{uid}:{resource}[:{hash}]
//	ROLE   -> serveradmin:v1:role:{role}:{resource}[:{hash}]
//
// 路径参数与规范化 QueryString 参与 SHA256，避免不同参数共用 Key。
// USER/ROLE 策略取不到登录身份时不缓存。
func responseKey(c *gin.Context, rule cache.Rule) (string, bool) {
	hash := requestParamHash(c)
	switch rule.Policy {
	case cache.PolicyPublic:
		return cache.PublicKey(rule.Scope, rule.Resource, hash), true
	case cache.PolicyUser:
		id := FromContext(c)
		if id == nil || id.UserID == "" {
			return "", false
		}
		return cache.UserKey(id.UserID, rule.Resource, hash), true
	case cache.PolicyRole:
		id := FromContext(c)
		if id == nil || len(id.Roles) == 0 {
			return "", false
		}
		return cache.RoleKey(cache.RoleSegment(id.Roles), rule.Resource, hash), true
	default:
		return "", false
	}
}

// requestParamHash 路径参数 + 规范化 QueryString 的 SHA256；无参数返回空。
func requestParamHash(c *gin.Context) string {
	vals := url.Values{}
	for _, p := range c.Params {
		vals.Set("param:"+p.Key, p.Value)
	}
	for k, vs := range c.Request.URL.Query() {
		vals[k] = vs
	}
	return cache.QueryHash(vals)
}

// isSuccessfulBody 仅缓存统一响应格式中 code=0 的成功响应。
func isSuccessfulBody(body []byte) bool {
	var probe struct {
		Code int `json:"code"`
	}
	return json.Unmarshal(body, &probe) == nil && probe.Code == 0
}

// cacheBodyWriter 捕获下游写入的响应体与状态码。
type cacheBodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *cacheBodyWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *cacheBodyWriter) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
