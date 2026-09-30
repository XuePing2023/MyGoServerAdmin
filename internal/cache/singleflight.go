package cache

import (
	"context"
	"encoding/json"
	"time"

	"golang.org/x/sync/singleflight"
)

// Helper Cache-Aside 通用泛型助手：Redis -> singleflight -> 数据库 -> 回填 Redis。
// 单个 Helper 可被多个 Service 共享（Key 按业务前缀隔离，不会冲突）。
type Helper struct {
	cache Cache // 可为 nil（未启用 Redis 时直接回源）
	group singleflight.Group
	TTL   TTLConfig
}

// NewHelper 创建缓存助手。c 为 nil 时退化为仅 singleflight 合并并发回源。
func NewHelper(c Cache, ttl TTLConfig) *Helper {
	return &Helper{cache: c, TTL: ttl}
}

// Get 按 Key 读缓存，未命中则并发合并后回源并回填（防缓存击穿）。
func (h *Helper) Get(
	ctx context.Context,
	key string,
	ttl time.Duration,
	loader func(context.Context) ([]byte, error),
) ([]byte, error) {
	// 1. Redis
	if h.cache != nil {
		if data, err := h.cache.Get(ctx, key); err == nil {
			return data, nil
		}
	}

	// 2. singleflight 防缓存击穿
	value, err, _ := h.group.Do(key, func() (any, error) {
		// double check：等锁期间可能已被其他协程回填
		if h.cache != nil {
			if data, err := h.cache.Get(ctx, key); err == nil {
				return data, nil
			}
		}
		// 3. 数据库（解除取消关联，避免首个请求取消殃及同等待者）
		data, err := loader(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		// 4. 回填 Redis（失败不影响本次返回）
		if h.cache != nil {
			_ = h.cache.Set(ctx, key, data, ttl)
		}
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return value.([]byte), nil
}

// GetJSON Get 的类型化封装：loader 返回业务对象，自动 JSON 编解码。
func GetJSON[T any](h *Helper, ctx context.Context, key string, ttl time.Duration, loader func(context.Context) (T, error)) (T, error) {
	var zero T
	raw, err := h.Get(ctx, key, ttl, func(c context.Context) ([]byte, error) {
		v, err := loader(c)
		if err != nil {
			return nil, err
		}
		return json.Marshal(v)
	})
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return zero, err
	}
	return out, nil
}

// Del 删除指定缓存 Key（写操作成功后调用）。
func (h *Helper) Del(ctx context.Context, keys ...string) {
	if h.cache == nil {
		return
	}
	_ = h.cache.Del(ctx, keys...)
}

// Invalidate 按前缀批量失效（写操作成功后调用，清除该业务的全部 Key 变体）。
func (h *Helper) Invalidate(ctx context.Context, prefixes ...string) {
	if h.cache == nil {
		return
	}
	for _, p := range prefixes {
		_ = h.cache.DelPrefix(ctx, p)
	}
}
