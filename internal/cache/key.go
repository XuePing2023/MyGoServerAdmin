package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// KeyPrefix 全局缓存 Key 前缀：serveradmin:v1:{业务}:{资源}:{参数}。
const KeyPrefix = "serveradmin:v1"

// PublicKey PUBLIC 策略 Key：serveradmin:v1:{biz}:{resource}[:{hash}]。
// 如 PublicKey("system", "config") -> serveradmin:v1:system:config
func PublicKey(biz, resource string, paramHash ...string) string {
	return joinKey(KeyPrefix, biz, resource, strings.Join(paramHash, ""))
}

// UserKey USER 策略 Key：serveradmin:v1:user:{uid}:{resource}[:{hash}]。
// 如 UserKey(uid, "profile") -> serveradmin:v1:user:{uid}:profile
func UserKey(uid, resource string, paramHash ...string) string {
	return joinKey(KeyPrefix, "user", uid, resource, strings.Join(paramHash, ""))
}

// RoleKey ROLE 策略 Key：serveradmin:v1:role:{role}:{resource}[:{hash}]。
// 如 RoleKey("super", "menu") -> serveradmin:v1:role:super:menu
func RoleKey(role, resource string, paramHash ...string) string {
	return joinKey(KeyPrefix, "role", role, resource, strings.Join(paramHash, ""))
}

// UserPrefix 用户维度失效前缀：serveradmin:v1:user:{uid}:。
func UserPrefix(uid string) string { return KeyPrefix + ":user:" + uid + ":" }

// BizPrefix 业务维度失效前缀：serveradmin:v1:{biz}:。
func BizPrefix(biz string) string { return KeyPrefix + ":" + biz + ":" }

// RolePrefix 角色维度失效前缀：serveradmin:v1:role:。
func RolePrefix() string { return KeyPrefix + ":role:" }

func joinKey(parts ...string) string {
	// 跳过空段：无参数 hash 时不产生尾随冒号，保持 serveradmin:v1:system:config 形态
	nonEmpty := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, ":")
}

// QueryHash 规范化 QueryString 后做 SHA256，而不是把原始 QueryString 拼进 Key：
// 参数按名称排序、剔除空值与前端时间戳参数（_ 开头），保证同一组参数得到同一 Key。
// 无参数时返回空串，Key 保持简洁（如 serveradmin:v1:system:config）。
func QueryHash(vals url.Values) string {
	if len(vals) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		vs := vals[k]
		if k == "" || strings.HasPrefix(k, "_") {
			continue
		}
		clean := make([]string, 0, len(vs))
		for _, v := range vs {
			if v != "" {
				clean = append(clean, v)
			}
		}
		if len(clean) == 0 {
			continue
		}
		sort.Strings(clean)
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(strings.Join(clean, ","))
		b.WriteByte('&')
	}
	if b.Len() == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// RoleSegment 计算角色 Key 段：单角色直接用编码（可读性好），
// 多角色排序拼接，超长则取 SHA256 前 16 位。
func RoleSegment(roles []string) string {
	if len(roles) == 0 {
		return "-"
	}
	sorted := make([]string, len(roles))
	copy(sorted, roles)
	sort.Strings(sorted)
	joined := strings.Join(sorted, ",")
	if len(joined) <= 48 {
		return joined
	}
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:16])
}
