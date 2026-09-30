package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

type permEntry struct {
	Status int
	Perms  map[string]struct{}
	Expire time.Time
}

// PermService 用户权限缓存：用户 -> 角色 -> 菜单 -> 权限标识。
// 短 TTL 缓存 + 变更时主动失效，避免每个请求都查库。
type PermService struct {
	db    *mongo.Database
	mu    sync.RWMutex
	cache map[string]*permEntry
	ttl   time.Duration
}

// NewPermService 创建权限缓存服务。
func NewPermService(db *mongo.Database) *PermService {
	return &PermService{db: db, cache: make(map[string]*permEntry), ttl: time.Minute}
}

// Get 返回用户权限集合与账号状态。
func (s *PermService) Get(ctx context.Context, userID string) (map[string]struct{}, int, error) {
	s.mu.RLock()
	e, ok := s.cache[userID]
	s.mu.RUnlock()
	if ok && time.Now().Before(e.Expire) {
		return e.Perms, e.Status, nil
	}

	perms, status, err := s.load(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	s.mu.Lock()
	s.cache[userID] = &permEntry{Perms: perms, Status: status, Expire: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return perms, status, nil
}

func (s *PermService) load(ctx context.Context, userID string) (map[string]struct{}, int, error) {
	var u model.User
	if err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"_id": userID}, &u); err != nil {
		return nil, 0, err
	}
	perms := make(map[string]struct{})
	for _, code := range u.Roles {
		if code == model.SuperRoleCode {
			perms["*"] = struct{}{}
			return perms, u.Status, nil
		}
	}

	if len(u.Roles) > 0 {
		cursor, err := s.db.Collection(model.ColRole).Find(ctx,
			bson.M{"code": bson.M{"$in": u.Roles}, "status": model.StatusEnabled})
		if err == nil {
			var roles []*model.Role
			if err := cursor.All(ctx, &roles); err == nil {
				menuIDs := make([]string, 0)
				for _, r := range roles {
					menuIDs = append(menuIDs, r.Menus...)
				}
				if len(menuIDs) > 0 {
					mcursor, err := s.db.Collection(model.ColMenu).Find(ctx,
						bson.M{"_id": bson.M{"$in": menuIDs}, "perm": bson.M{"$exists": true, "$ne": ""}})
					if err == nil {
						var menus []*model.Menu
						if err := mcursor.All(ctx, &menus); err == nil {
							for _, m := range menus {
								perms[m.Perm] = struct{}{}
							}
						}
						mcursor.Close(ctx)
					}
				}
			}
			cursor.Close(ctx)
		}
	}
	return perms, u.Status, nil
}

// InvalidateUser 使用户权限缓存失效。
func (s *PermService) InvalidateUser(userID string) {
	s.mu.Lock()
	delete(s.cache, userID)
	s.mu.Unlock()
}

// InvalidateAll 清空全部权限缓存（角色/菜单变更时调用）。
func (s *PermService) InvalidateAll() {
	s.mu.Lock()
	s.cache = make(map[string]*permEntry)
	s.mu.Unlock()
}
