package model

import "time"

// User 用户。
type User struct {
	BaseModel `bson:",inline"`
	Username  string     `bson:"username" json:"username"`
	Nickname  string     `bson:"nickname" json:"nickname"`
	Password  string     `bson:"password" json:"-"`
	Avatar    string     `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Email     string     `bson:"email,omitempty" json:"email,omitempty"`
	Phone     string     `bson:"phone,omitempty" json:"phone,omitempty"`
	Gender    int        `bson:"gender,omitempty" json:"gender,omitempty"` // 1男 2女 0未知
	DeptID    string     `bson:"deptId,omitempty" json:"deptId,omitempty"`
	Roles     []string   `bson:"roles" json:"roles"` // 角色编码列表
	Status    int        `bson:"status" json:"status"`
	Remark    string     `bson:"remark,omitempty" json:"remark,omitempty"`
	LastLogin *time.Time `bson:"lastLogin,omitempty" json:"lastLogin,omitempty"`
	LastIP    string     `bson:"lastIp,omitempty" json:"lastIp,omitempty"`
}

// Role 角色。
type Role struct {
	BaseModel `bson:",inline"`
	Name      string   `bson:"name" json:"name"`
	Code      string   `bson:"code" json:"code"`
	Sort      int      `bson:"sort" json:"sort"`
	Status    int      `bson:"status" json:"status"`
	Remark    string   `bson:"remark,omitempty" json:"remark,omitempty"`
	Menus     []string `bson:"menus" json:"menus"`     // 关联的菜单 ID
	BuiltIn   bool     `bson:"builtIn" json:"builtIn"` // 内置角色不可删除
}

// 菜单类型。
const (
	MenuTypeDir    = 1 // 目录
	MenuTypePage   = 2 // 菜单（页面）
	MenuTypeButton = 3 // 按钮（权限点）
)

// Menu 菜单/权限。
type Menu struct {
	BaseModel `bson:",inline"`
	ParentID  string  `bson:"parentId" json:"parentId"`
	Name      string  `bson:"name" json:"name"`
	Path      string  `bson:"path,omitempty" json:"path,omitempty"`           // 前端路由
	Component string  `bson:"component,omitempty" json:"component,omitempty"` // 页面组件标识
	Perm      string  `bson:"perm,omitempty" json:"perm,omitempty"`           // 权限标识，如 sys:user:create
	Icon      string  `bson:"icon,omitempty" json:"icon,omitempty"`
	Sort      int     `bson:"sort" json:"sort"`
	Type      int     `bson:"type" json:"type"`
	Visible   bool    `bson:"visible" json:"visible"`
	Status    int     `bson:"status" json:"status"`
	Children  []*Menu `bson:"-" json:"children,omitempty"` // 非持久化，接口返回时组装
}

// Department 部门。
type Department struct {
	BaseModel `bson:",inline"`
	ParentID  string        `bson:"parentId" json:"parentId"`
	Name      string        `bson:"name" json:"name"`
	Leader    string        `bson:"leader,omitempty" json:"leader,omitempty"`
	Phone     string        `bson:"phone,omitempty" json:"phone,omitempty"`
	Sort      int           `bson:"sort" json:"sort"`
	Status    int           `bson:"status" json:"status"`
	Children  []*Department `bson:"-" json:"children,omitempty"`
}
