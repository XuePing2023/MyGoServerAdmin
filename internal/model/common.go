// Package model 定义所有 MongoDB 文档结构。
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 集合名称。
const (
	ColUser           = "users"
	ColRole           = "roles"
	ColMenu           = "menus"
	ColDepartment     = "departments"
	ColDictType       = "dict_types"
	ColDictItem       = "dict_items"
	ColSysConfig      = "sys_configs"
	ColOperationLog   = "operation_logs"
	ColLoginLog       = "login_logs"
	ColNotice         = "notices"
	ColFile           = "files"
	ColJob            = "jobs"
	ColJobLog         = "job_logs"
	ColTokenBlacklist = "token_blacklist"
)

// 通用状态：1 启用 / 2 停用。
const (
	StatusEnabled  = 1
	StatusDisabled = 2
)

// 超级管理员角色编码，拥有全部权限。
const SuperRoleCode = "super"

// 通知类型。
const (
	NoticeTypeNotice  = 1 // 通知
	NoticeTypeAnnouce = 2 // 公告
)

// BaseModel 所有文档的公共字段。ID 统一使用 ObjectID 的 hex 字符串。
type BaseModel struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// NewID 生成新的 ObjectID hex 字符串。
func NewID() string { return primitive.NewObjectID().Hex() }

// PrepareCreate 新建时填充 ID 与时间。
func (b *BaseModel) PrepareCreate() {
	now := time.Now()
	if b.ID == "" {
		b.ID = NewID()
	}
	b.CreatedAt = now
	b.UpdatedAt = now
}

// TouchUpdate 更新时刷新时间字段。
func (b *BaseModel) TouchUpdate() { b.UpdatedAt = time.Now() }
