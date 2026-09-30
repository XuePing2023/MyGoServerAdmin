package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"serveradmin/internal/model"
)

// NoticeRepository 通知公告集合数据访问。
type NoticeRepository struct {
	collection[model.Notice]
}

// NewNoticeRepository 创建通知公告仓库。
func NewNoticeRepository(db *mongo.Database) *NoticeRepository {
	return &NoticeRepository{collectionOf[model.Notice](db, model.ColNotice)}
}

// FindRecent 取最近发布的启用公告（按创建时间倒序取 limit 条）。
func (r *NoticeRepository) FindRecent(ctx context.Context, limit int) ([]*model.Notice, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.coll.Find(ctx, bson.M{"status": 1}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*model.Notice, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}
