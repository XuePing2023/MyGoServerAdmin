package model

// DictType 字典类型。
type DictType struct {
	BaseModel `bson:",inline"`
	Name      string `bson:"name" json:"name"`
	Code      string `bson:"code" json:"code"`
	Status    int    `bson:"status" json:"status"`
	Remark    string `bson:"remark,omitempty" json:"remark,omitempty"`
}

// DictItem 字典项。
type DictItem struct {
	BaseModel `bson:",inline"`
	TypeCode  string `bson:"typeCode" json:"typeCode"`
	Label     string `bson:"label" json:"label"`
	Value     string `bson:"value" json:"value"`
	TagType   string `bson:"tagType,omitempty" json:"tagType,omitempty"` // 前端标签色：success/info/warning/danger
	Sort      int    `bson:"sort" json:"sort"`
	Status    int    `bson:"status" json:"status"`
	Remark    string `bson:"remark,omitempty" json:"remark,omitempty"`
}

// SysConfig 系统参数（KV）。
type SysConfig struct {
	BaseModel `bson:",inline"`
	Key       string `bson:"key" json:"key"`
	Name      string `bson:"name" json:"name"`
	Value     string `bson:"value" json:"value"`
	Remark    string `bson:"remark,omitempty" json:"remark,omitempty"`
	BuiltIn   bool   `bson:"builtIn" json:"builtIn"`
}

// Notice 通知公告。
type Notice struct {
	BaseModel `bson:",inline"`
	Title     string `bson:"title" json:"title"`
	Type      int    `bson:"type" json:"type"` // 1 通知 2 公告
	Content   string `bson:"content" json:"content"`
	Status    int    `bson:"status" json:"status"` // 1 已发布 2 草稿
	Publisher string `bson:"publisher" json:"publisher"`
}
