package model

import "time"

// OperationLog 操作日志。
type OperationLog struct {
	BaseModel `bson:",inline"`
	UserID    string `bson:"userId,omitempty" json:"userId,omitempty"`
	Username  string `bson:"username" json:"username"`
	Module    string `bson:"module" json:"module"` // 模块，如 用户管理
	Action    string `bson:"action" json:"action"` // 动作，如 新增用户 / 删除
	Method    string `bson:"method" json:"method"`
	Path      string `bson:"path" json:"path"`
	Query     string `bson:"query,omitempty" json:"query,omitempty"`
	Body      string `bson:"body,omitempty" json:"body,omitempty"` // 已脱敏
	IP        string `bson:"ip" json:"ip"`
	UserAgent string `bson:"userAgent,omitempty" json:"userAgent,omitempty"`
	HTTPCode  int    `bson:"httpCode" json:"httpCode"`
	BizCode   int    `bson:"bizCode" json:"bizCode"`
	CostMs    int64  `bson:"costMs" json:"costMs"`
}

// LoginLog 登录日志。
type LoginLog struct {
	BaseModel `bson:",inline"`
	UserID    string    `bson:"userId,omitempty" json:"userId,omitempty"`
	Username  string    `bson:"username" json:"username"`
	IP        string    `bson:"ip" json:"ip"`
	UserAgent string    `bson:"userAgent,omitempty" json:"userAgent,omitempty"`
	Status    int       `bson:"status" json:"status"` // 1 成功 2 失败
	Msg       string    `bson:"msg" json:"msg"`
	LoginAt   time.Time `bson:"loginAt" json:"loginAt"`
}

// 登录结果状态。
const (
	LoginStatusSuccess = 1
	LoginStatusFailed  = 2
)

// FileRecord 上传文件记录。
type FileRecord struct {
	BaseModel    `bson:",inline"`
	Name         string `bson:"name" json:"name"` // 存储文件名
	OriginalName string `bson:"originalName" json:"originalName"`
	Size         int64  `bson:"size" json:"size"`
	ContentType  string `bson:"contentType" json:"contentType"`
	Path         string `bson:"path" json:"path"` // 相对 uploadDir 的路径
	URL          string `bson:"url" json:"url"`   // 访问 URL
	Uploader     string `bson:"uploader" json:"uploader"`
}

// Job 定时任务。
type Job struct {
	BaseModel `bson:",inline"`
	Name      string     `bson:"name" json:"name"`
	Spec      string     `bson:"spec" json:"spec"`       // cron 表达式（5 段）
	Handler   string     `bson:"handler" json:"handler"` // 处理器名称
	Params    string     `bson:"params,omitempty" json:"params,omitempty"`
	Status    int        `bson:"status" json:"status"` // 1 启用 2 停用
	Remark    string     `bson:"remark,omitempty" json:"remark,omitempty"`
	LastRun   *time.Time `bson:"lastRun,omitempty" json:"lastRun,omitempty"`
	NextRun   *time.Time `bson:"nextRun,omitempty" json:"nextRun,omitempty"`
	RunCount  int64      `bson:"runCount" json:"runCount"`
	FailCount int64      `bson:"failCount" json:"failCount"`
}

// JobLog 任务执行日志。
type JobLog struct {
	BaseModel `bson:",inline"`
	JobName   string    `bson:"jobName" json:"jobName"`
	Handler   string    `bson:"handler" json:"handler"`
	Success   bool      `bson:"success" json:"success"`
	Output    string    `bson:"output,omitempty" json:"output,omitempty"`
	Error     string    `bson:"error,omitempty" json:"error,omitempty"`
	CostMs    int64     `bson:"costMs" json:"costMs"`
	RunAt     time.Time `bson:"runAt" json:"runAt"`
	Manual    bool      `bson:"manual" json:"manual"`
}

// TokenBlacklist 已注销令牌黑名单（带 TTL 索引自动清理）。
type TokenBlacklist struct {
	ID        string    `bson:"_id" json:"id"` // jti
	ExpireAt  time.Time `bson:"expireAt" json:"expireAt"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
