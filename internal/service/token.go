package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"serveradmin/internal/model"
)

// TokenService 已注销（登出/被踢下线）令牌黑名单。
// 内存为主，启动时从 MongoDB 加载未过期记录，写入时双写持久化。
type TokenService struct {
	db  *mongo.Database
	mu  sync.RWMutex
	mem map[string]time.Time // jti -> 过期时间
}

// NewTokenService 创建黑名单服务。
func NewTokenService(db *mongo.Database) *TokenService {
	return &TokenService{db: db, mem: make(map[string]time.Time)}
}

// Load 启动时清理过期记录并加载未过期黑名单到内存。
func (s *TokenService) Load(ctx context.Context) error {
	c := s.db.Collection(model.ColTokenBlacklist)
	if _, err := c.DeleteMany(ctx, bson.M{"expireAt": bson.M{"$lte": time.Now()}}); err != nil {
		return err
	}
	cursor, err := c.Find(ctx, bson.M{"expireAt": bson.M{"$gt": time.Now()}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	list := make([]*model.TokenBlacklist, 0)
	if err := cursor.All(ctx, &list); err != nil {
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
	_, err := s.db.Collection(model.ColTokenBlacklist).InsertOne(bg, doc)
	return err
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
