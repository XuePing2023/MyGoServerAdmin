package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/model"
	"serveradmin/internal/repository"
)

type permEntry struct {
	Status int
	Perms  map[string]struct{}
	Expire time.Time
}

// PermService 用户权限缓存：用户 -> 角色 -> 菜单 -> 权限标识。
// 短 TTL 缓存 + 变更时主动失效，避免每个请求都查库。
type PermService struct {
	user  *repository.UserRepository
	role  *repository.RoleRepository
	menu  *repository.MenuRepository
	mu    sync.RWMutex
	cache map[string]*permEntry
	ttl   time.Duration
}

// NewPermService 创建权限缓存服务。
func NewPermService(user *repository.UserRepository, role *repository.RoleRepository, menu *repository.MenuRepository) *PermService {
	return &PermService{
		user:  user,
		role:  role,
		menu:  menu,
		cache: make(map[string]*permEntry),
		ttl:   time.Minute,
	}
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
	u, err := s.user.FindOne(ctx, bson.M{"_id": userID})
	if err != nil {
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
		roles, err := s.role.FindAll(ctx,
			bson.M{"code": bson.M{"$in": u.Roles}, "status": model.StatusEnabled}, nil)
		if err == nil {
			menuIDs := make([]string, 0)
			for _, r := range roles {
				menuIDs = append(menuIDs, r.Menus...)
			}
			if len(menuIDs) > 0 {
				menus, err := s.menu.FindAll(ctx,
					bson.M{"_id": bson.M{"$in": menuIDs}, "perm": bson.M{"$exists": true, "$ne": ""}}, nil)
				if err == nil {
					for _, m := range menus {
						perms[m.Perm] = struct{}{}
					}
				}
			}
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
