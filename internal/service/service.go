// Package service 业务逻辑层：业务规则、缓存编排与权限控制。
// 数据访问全部委托 internal/repository，本层不直接接触 mongo.Collection。
package service

import (
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/cache"
	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/jwtx"
	"serveradmin/internal/repository"
)

// Registry 聚合所有业务服务，统一构建与传递。
type Registry struct {
	Perm      *PermService
	Token     *TokenService
	Online    *OnlineService
	Auth      *AuthService
	User      *UserService
	Role      *RoleService
	Menu      *MenuService
	Dept      *DepartmentService
	Dict      *DictService
	SysConfig *SysConfigService
	Log       *LogService
	File      *FileService
	Notice    *NoticeService
	Job       *JobService
	Dashboard *DashboardService
	Monitor   *MonitorService

	// Cache 原始缓存实现（供 HTTP 响应缓存中间件使用），未启用 Redis 时为 nil。
	Cache cache.Cache
	// CacheHelper 服务层 Cache-Aside 助手（内置 singleflight），始终可用。
	CacheHelper *cache.Helper
}

// New 构建所有服务。jwtMgr 由 main 创建后传入；cacheClient 为 nil 表示未启用缓存。
func New(db *mongo.Database, cfg *config.Config, jwtMgr *jwtx.Manager, cacheClient cache.Cache) *Registry {
	repos := repository.NewAll(db)
	helper := cache.NewHelper(cacheClient, cache.TTLConfig{
		Public: time.Duration(cfg.Cache.PublicTTLSeconds) * time.Second,
		User:   time.Duration(cfg.Cache.UserTTLSeconds) * time.Second,
		Role:   time.Duration(cfg.Cache.RoleTTLSeconds) * time.Second,
	})
	r := &Registry{
		Perm:        NewPermService(repos.User, repos.Role, repos.Menu),
		Token:       NewTokenService(repos.Token),
		Online:      NewOnlineService(time.Duration(cfg.JWT.AccessExpireMinutes) * time.Minute),
		Cache:       cacheClient,
		CacheHelper: helper,
	}
	r.Menu = NewMenuService(repos.Menu, repos.Role, helper)
	r.Menu.SetPerm(r.Perm)
	r.Auth = NewAuthService(cfg, jwtMgr, r.Token, r.Menu, r.Perm, repos.User, repos.Log, helper)
	r.User = NewUserService(repos.User, repos.Role, repos.Department, cfg, helper)
	r.Role = NewRoleService(repos.Role, helper)
	r.Role.SetPerm(r.Perm)
	r.Dept = NewDepartmentService(repos.Department)
	r.Dict = NewDictService(repos.Dict, helper)
	r.SysConfig = NewSysConfigService(repos.SysConfig, helper)
	r.Log = NewLogService(repos.Log)
	r.File = NewFileService(repos.File, cfg)
	r.Notice = NewNoticeService(repos.Notice)
	r.Job = NewJobService(repos.Job, repos.Token, repos.Log)
	r.Dashboard = NewDashboardService(repos.Dashboard)
	r.Monitor = NewMonitorService()
	return r
}

// ---------- 通用小工具（包内共享，用于组装查询条件） ----------

// likeFilter 大小写不敏感的模糊匹配条件。
func likeFilter(s string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(strings.TrimSpace(s)), "$options": "i"}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func unique(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))
	for _, v := range list {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// normalizePage 规整分页参数。
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

// statusFilter 若 status 有效则返回状态过滤条件。
func statusFilter(status int) bson.M {
	if status == model.StatusEnabled || status == model.StatusDisabled {
		return bson.M{"status": status}
	}
	return bson.M{}
}

// timeRange 构造时间范围过滤条件（end 为开区间）。
func timeRange(field string, start, end time.Time) (bson.M, bool) {
	m := bson.M{}
	if !start.IsZero() {
		m["$gte"] = start
	}
	if !end.IsZero() {
		m["$lt"] = end
	}
	if len(m) == 0 {
		return nil, false
	}
	return m, true
}
