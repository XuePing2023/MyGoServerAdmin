// Package service 业务逻辑层：所有对 MongoDB 的读写都集中在这里。
package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"serveradmin/internal/cache"
	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/jwtx"
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
	helper := cache.NewHelper(cacheClient, cache.TTLConfig{
		Public: time.Duration(cfg.Cache.PublicTTLSeconds) * time.Second,
		User:   time.Duration(cfg.Cache.UserTTLSeconds) * time.Second,
		Role:   time.Duration(cfg.Cache.RoleTTLSeconds) * time.Second,
	})
	r := &Registry{
		Perm:        NewPermService(db),
		Token:       NewTokenService(db),
		Online:      NewOnlineService(time.Duration(cfg.JWT.AccessExpireMinutes) * time.Minute),
		Cache:       cacheClient,
		CacheHelper: helper,
	}
	r.Menu = NewMenuService(db, helper)
	r.Menu.SetPerm(r.Perm)
	r.Auth = NewAuthService(db, cfg, jwtMgr, r.Token, r.Menu, r.Perm, helper)
	r.User = NewUserService(db, cfg, helper)
	r.Role = NewRoleService(db, helper)
	r.Role.SetPerm(r.Perm)
	r.Dept = NewDepartmentService(db)
	r.Dict = NewDictService(db, helper)
	r.SysConfig = NewSysConfigService(db, helper)
	r.Log = NewLogService(db)
	r.File = NewFileService(db, cfg)
	r.Notice = NewNoticeService(db)
	r.Job = NewJobService(db)
	r.Dashboard = NewDashboardService(db)
	r.Monitor = NewMonitorService()
	return r
}

// ---------- 通用小工具（包内共享） ----------

func findOne(ctx context.Context, c *mongo.Collection, filter bson.M, doc any) error {
	return c.FindOne(ctx, filter).Decode(doc)
}

func count(ctx context.Context, c *mongo.Collection, filter bson.M) (int64, error) {
	return c.CountDocuments(ctx, filter)
}

// pageFind 通用分页查询：按 createdAt 倒序（sort 为 nil 时）。
func pageFind[T any](ctx context.Context, c *mongo.Collection, filter bson.M, page, size int, sort bson.D) ([]*T, int64, error) {
	total, err := c.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*T{}, 0, nil
	}
	if sort == nil {
		sort = bson.D{{Key: "createdAt", Value: -1}}
	}
	opts := options.Find().
		SetSort(sort).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cursor, err := c.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	list := make([]*T, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
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

// findOpts 便捷构造 FindOptions（按 createdAt 排序 + limit）。
// v2 中 Find 接受 options.Lister[FindOptions]，*FindOptionsBuilder 实现了该接口。
func findOpts(sortValue, limit int) *options.FindOptionsBuilder {
	return options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: sortValue}}).
		SetLimit(int64(limit))
}

// statusFilter 若 status 有效则返回状态过滤条件。
func statusFilter(status int) bson.M {
	if status == model.StatusEnabled || status == model.StatusDisabled {
		return bson.M{"status": status}
	}
	return bson.M{}
}
