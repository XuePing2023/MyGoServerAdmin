package cache

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// IdentityFunc 从请求上下文提取登录身份（uid 与角色编码）。
// 由调用方注入（如基于 middleware.FromContext 实现），避免包间循环依赖。
type IdentityFunc func(c *gin.Context) (uid string, roles []string, ok bool)

// Rule 单个 API 的显式缓存规则（只缓存 GET，成功响应才写入）。
type Rule struct {
	Policy   Policy        // PUBLIC / USER / ROLE
	Scope    string        // PUBLIC 时的业务段（如 system、menu）
	Resource string        // 资源段（如 config、menu、profile）
	TTL      time.Duration // 覆盖策略默认 TTL；0 表示使用策略默认值
}

// routeKey 策略表的 Key：方法 + 路由模式（c.FullPath()）。
func routeKey(c *gin.Context) string {
	return c.Request.Method + " " + c.FullPath()
}

// ResponseCache HTTP 响应缓存中间件（Cache-Aside）：
// HIT 直接返回 Redis 中的响应；MISS 放行到 Controller/MongoDB，
// 成功响应（HTTP 200 且 code=0）回填 Redis。
// routes 为显式配置表：路由模式 -> 缓存规则，未配置的路由不缓存（NONE）。
// identity 用于 USER/ROLE 策略组装 Key。
func ResponseCache(cc Cache, ttl TTLConfig, routes map[string]Rule, identity IdentityFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cc == nil || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}
		rule, ok := routes[routeKey(c)]
		if !ok || rule.Policy == PolicyNone {
			c.Next()
			return
		}

		key, valid := responseKey(c, rule, identity)
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

		bw := &bodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw
		c.Next()
		c.Header("Cache-Control", "no-store")
		c.Header("X-Cache", "MISS")

		if bw.Status() == http.StatusOK && bw.buf.Len() > 0 && isSuccessfulBody(bw.buf.Bytes()) {
			t := rule.TTL
			if t == 0 {
				t = ttlFor(ttl, rule.Policy)
			}
			_ = cc.Set(ctx, key, bw.buf.Bytes(), t)
		}
	}
}

// ttlFor 取策略默认 TTL。
func ttlFor(ttl TTLConfig, p Policy) time.Duration {
	switch p {
	case PolicyUser:
		return ttl.User
	case PolicyRole:
		return ttl.Role
	default:
		return ttl.Public
	}
}

// responseKey 按策略组装 Key：
//
//	PUBLIC -> serveradmin:v1:{scope}:{resource}[:{hash}]
//	USER   -> serveradmin:v1:user:{uid}:{resource}[:{hash}]
//	ROLE   -> serveradmin:v1:role:{role}:{resource}[:{hash}]
//
// 路径参数与规范化 QueryString 参与 SHA256，避免不同参数共用 Key。
// USER 策略取不到登录身份时不缓存。
func responseKey(c *gin.Context, rule Rule, identity IdentityFunc) (string, bool) {
	hash := requestParamHash(c)
	switch rule.Policy {
	case PolicyPublic:
		return PublicKey(rule.Scope, rule.Resource, hash), true
	case PolicyUser:
		uid, _, ok := identity(c)
		if !ok || uid == "" {
			return "", false
		}
		return UserKey(uid, rule.Resource, hash), true
	case PolicyRole:
		_, roles, ok := identity(c)
		if !ok || len(roles) == 0 {
			return "", false
		}
		return RoleKey(RoleSegment(roles), rule.Resource, hash), true
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
	return QueryHash(vals)
}

// isSuccessfulBody 仅缓存统一响应格式中 code=0 的成功响应。
func isSuccessfulBody(body []byte) bool {
	var probe struct {
		Code int `json:"code"`
	}
	return json.Unmarshal(body, &probe) == nil && probe.Code == 0
}

// bodyWriter 捕获下游写入的响应体与状态码。
type bodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
