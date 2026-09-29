package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/jwtx"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// Identity 当前请求的登录身份。
type Identity struct {
	UserID   string
	Username string
	Nickname string
	JTI      string
	Roles    []string
	Perms    map[string]struct{}
	Claims   *jwtx.Claims
}

const identityKey = "identity"

// HasPerm 判断身份是否拥有某权限（超级管理员通配）。
func (id *Identity) HasPerm(perm string) bool {
	if id == nil {
		return false
	}
	if _, ok := id.Perms["*"]; ok {
		return true
	}
	_, ok := id.Perms[perm]
	return ok
}

// FromContext 取当前身份。
func FromContext(c *gin.Context) *Identity {
	if v, ok := c.Get(identityKey); ok {
		if id, ok := v.(*Identity); ok {
			return id
		}
	}
	return nil
}

// Auth JWT 认证中间件：校验令牌、黑名单、账号状态，登记在线状态。
func Auth(jwtMgr *jwtx.Manager, perm *service.PermService, online *service.OnlineService, token *service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := BearerToken(c)
		if raw == "" {
			response.Fail(c, http.StatusUnauthorized, "未登录或令牌缺失")
			c.Abort()
			return
		}
		claims, err := jwtMgr.Parse(raw, jwtx.TypeAccess)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "登录状态已过期，请重新登录")
			c.Abort()
			return
		}
		if token.Banned(claims.ID) {
			response.Fail(c, http.StatusUnauthorized, "令牌已失效，请重新登录")
			c.Abort()
			return
		}

		perms, status, err := perm.Get(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Handle(c, err)
			c.Abort()
			return
		}
		if status != model.StatusEnabled {
			response.Fail(c, http.StatusForbidden, "账号已被禁用，请联系管理员")
			c.Abort()
			return
		}

		id := &Identity{
			UserID:   claims.UserID,
			Username: claims.Username,
			Nickname: claims.Nickname,
			JTI:      claims.ID,
			Roles:    claims.Roles,
			Perms:    perms,
			Claims:   claims,
		}
		c.Set(identityKey, id)
		online.Touch(claims.ID, claims.UserID, claims.Username, claims.Nickname, c.ClientIP())
		c.Next()
	}
}

// RequirePerm 权限校验中间件。
func RequirePerm(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := FromContext(c)
		if id == nil {
			response.Fail(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}
		if !id.HasPerm(perm) {
			response.Handle(c, errs.Forbidden("无操作权限（%s）", perm))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RolesContains 判断角色列表是否包含指定角色。
func RolesContains(roles []string, code string) bool {
	for _, r := range roles {
		if strings.EqualFold(r, code) {
			return true
		}
	}
	return false
}
