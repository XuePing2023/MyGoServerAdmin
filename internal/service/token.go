package service

import (
	"context"
	"sync"
	"time"

	"serveradmin/internal/model"
	"serveradmin/internal/repository"
)

// TokenService 已注销（登出/被踢下线）令牌黑名单。
// 内存为主，启动时从 MongoDB 加载未过期记录，写入时双写持久化。
type TokenService struct {
	repo *repository.TokenRepository
	mu   sync.RWMutex
	mem  map[string]time.Time // jti -> 过期时间
}

// NewTokenService 创建黑名单服务。
func NewTokenService(repo *repository.TokenRepository) *TokenService {
	return &TokenService{repo: repo, mem: make(map[string]time.Time)}
}

// Load 启动时清理过期记录并加载未过期黑名单到内存。
func (s *TokenService) Load(ctx context.Context) error {
	now := time.Now()
	if _, err := s.repo.DeleteExpired(ctx, now); err != nil {
		return err
	}
	list, err := s.repo.FindActive(ctx, now)
	if err != nil {
		return err
	}
	s.mu.Lock()
	for _, t := range list {
		s.mem[t.ID] = t.ExpireAt
	}
	s.mu.Unlock()
	return nil
}

// Add 加入黑名单（内存 + MongoDB，集合带 TTL 索引自动清理）。
func (s *TokenService) Add(ctx context.Context, jti string, expireAt time.Time) error {
	if jti == "" || !expireAt.After(time.Now()) {
		return nil
	}
	s.mu.Lock()
	s.mem[jti] = expireAt
	s.mu.Unlock()

	doc := &model.TokenBlacklist{ID: jti, ExpireAt: expireAt, CreatedAt: time.Now()}
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.repo.Insert(bg, doc)
}

// Banned 判断令牌是否已被注销。
func (s *TokenService) Banned(jti string) bool {
	if jti == "" {
		return false
	}
	s.mu.RLock()
	exp, ok := s.mem[jti]
	s.mu.RUnlock()
	return ok && exp.After(time.Now())
}
