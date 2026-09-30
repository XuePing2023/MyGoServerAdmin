package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// TokenRepository 令牌黑名单集合数据访问。
type TokenRepository struct {
	collection[model.TokenBlacklist]
}

// NewTokenRepository 创建令牌黑名单仓库。
func NewTokenRepository(db *mongo.Database) *TokenRepository {
	return &TokenRepository{collectionOf[model.TokenBlacklist](db, model.ColTokenBlacklist)}
}

// DeleteExpired 清理已过期的黑名单记录。
func (r *TokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return r.Delete(ctx, bson.M{"expireAt": bson.M{"$lte": now}})
}

// FindActive 取尚未过期的黑名单记录（启动预热用）。
func (r *TokenRepository) FindActive(ctx context.Context, now time.Time) ([]*model.TokenBlacklist, error) {
	return r.FindAll(ctx, bson.M{"expireAt": bson.M{"$gt": now}}, nil)
}
