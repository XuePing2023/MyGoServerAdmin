// Package cache 提供 Redis 缓存基础设施：
// Cache-Aside 读取（singleflight 防击穿）、缓存 Key 规范与策略、HTTP 响应缓存中间件。
package cache

import (
	"context"
	"errors"
	"time"
)

// ErrMiss 缓存未命中。
var ErrMiss = errors.New("cache: miss")

// Cache 缓存抽象，便于替换实现与单测。
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	DelPrefix(ctx context.Context, prefix string) error
}
