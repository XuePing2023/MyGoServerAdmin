package cache

import "time"

// Policy 缓存策略：决定 Key 中身份段的组成方式。
type Policy string

const (
	PolicyNone   Policy = "NONE"   // 完全不缓存（列表/日志/实时数据）
	PolicyPublic Policy = "PUBLIC" // 所有用户数据相同
	PolicyUser   Policy = "USER"   // 每个用户不同
	PolicyRole   Policy = "ROLE"   // 按角色/权限区分
)

// TTLConfig 各缓存策略的默认 TTL。
type TTLConfig struct {
	Public time.Duration // PUBLIC：全用户共享
	User   time.Duration // USER：按用户区分
	Role   time.Duration // ROLE：按角色区分
}

// Rule 单个 API 的显式缓存规则（只缓存 GET，成功响应才写入）。
type Rule struct {
	Policy   Policy        // PUBLIC / USER / ROLE
	Scope    string        // PUBLIC 时的业务段（如 system、menu）
	Resource string        // 资源段（如 config、menu、profile）
	TTL      time.Duration // 覆盖策略默认 TTL；0 表示使用策略默认值
}

// TTLFor 取策略默认 TTL。
func (t TTLConfig) TTLFor(p Policy) time.Duration {
	switch p {
	case PolicyUser:
		return t.User
	case PolicyRole:
		return t.Role
	default:
		return t.Public
	}
}
