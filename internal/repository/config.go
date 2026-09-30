package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// SysConfigRepository 系统参数集合数据访问。
type SysConfigRepository struct {
	collection[model.SysConfig]
}

// NewSysConfigRepository 创建系统参数仓库。
func NewSysConfigRepository(db *mongo.Database) *SysConfigRepository {
	return &SysConfigRepository{collectionOf[model.SysConfig](db, model.ColSysConfig)}
}

// FindByKeys 按参数键批量取配置。
func (r *SysConfigRepository) FindByKeys(ctx context.Context, keys []string) ([]*model.SysConfig, error) {
	return r.FindAll(ctx, bson.M{"key": bson.M{"$in": keys}}, nil)
}
