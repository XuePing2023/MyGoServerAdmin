package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// LogRepository 操作日志与登录日志数据访问。
type LogRepository struct {
	op    collection[model.OperationLog]
	login collection[model.LoginLog]
}

// NewLogRepository 创建日志仓库。
func NewLogRepository(db *mongo.Database) *LogRepository {
	return &LogRepository{
		op:    collectionOf[model.OperationLog](db, model.ColOperationLog),
		login: collectionOf[model.LoginLog](db, model.ColLoginLog),
	}
}

// ---------- 操作日志 ----------

// InsertOperation 保存操作日志。
func (r *LogRepository) InsertOperation(ctx context.Context, log *model.OperationLog) error {
	return r.op.Insert(ctx, log)
}

// PageOperation 操作日志分页。
func (r *LogRepository) PageOperation(ctx context.Context, filter bson.M, page, size int) ([]*model.OperationLog, int64, error) {
	return r.op.Page(ctx, filter, page, size, nil)
}

// DeleteOperation 删除操作日志；ids 为空时清空全部。
func (r *LogRepository) DeleteOperation(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return r.op.Delete(ctx, bson.M{})
	}
	return r.op.DeleteByIDs(ctx, ids)
}

// DeleteOperationBefore 删除某时间点之前的操作日志（定时清理用）。
func (r *LogRepository) DeleteOperationBefore(ctx context.Context, cut time.Time) (int64, error) {
	return r.op.Delete(ctx, bson.M{"createdAt": bson.M{"$lt": cut}})
}

// ---------- 登录日志 ----------

// InsertLogin 保存登录日志。
func (r *LogRepository) InsertLogin(ctx context.Context, log *model.LoginLog) error {
	return r.login.Insert(ctx, log)
}

// PageLogin 登录日志分页。
func (r *LogRepository) PageLogin(ctx context.Context, filter bson.M, page, size int) ([]*model.LoginLog, int64, error) {
	return r.login.Page(ctx, filter, page, size, nil)
}

// DeleteLogin 删除登录日志；ids 为空时清空全部。
func (r *LogRepository) DeleteLogin(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return r.login.Delete(ctx, bson.M{})
	}
	return r.login.DeleteByIDs(ctx, ids)
}

// DeleteLoginBefore 删除某时间点之前的登录日志（定时清理用）。
func (r *LogRepository) DeleteLoginBefore(ctx context.Context, cut time.Time) (int64, error) {
	return r.login.Delete(ctx, bson.M{"loginAt": bson.M{"$lt": cut}})
}
