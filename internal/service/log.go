package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"serveradmin/internal/model"
)

// LogService 操作日志与登录日志。
type LogService struct {
	db *mongo.Database
}

// NewLogService 创建日志服务。
func NewLogService(db *mongo.Database) *LogService {
	return &LogService{db: db}
}

// RecordOperation 保存操作日志。
func (s *LogService) RecordOperation(log *model.OperationLog) {
	log.PrepareCreate()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.db.Collection(model.ColOperationLog).InsertOne(ctx, log)
}

// OpQuery 操作日志查询条件。
type OpQuery struct {
	Username string
	Module   string
	Start    time.Time
	End      time.Time
	Page     int
	Size     int
}

// OpList 操作日志分页。
func (s *LogService) OpList(ctx context.Context, q *OpQuery) ([]*model.OperationLog, int64, error) {
	filter := bson.M{}
	if q.Username != "" {
		filter["username"] = likeFilter(q.Username)
	}
	if q.Module != "" {
		filter["module"] = likeFilter(q.Module)
	}
	if rangeFilter, ok := timeRange("createdAt", q.Start, q.End); ok {
		filter["createdAt"] = rangeFilter
	}
	page, size := normalizePage(q.Page, q.Size)
	return pageFind[model.OperationLog](ctx, s.db.Collection(model.ColOperationLog), filter, page, size, nil)
}

// OpDelete 批量删除；ids 为空时清空全部。
func (s *LogService) OpDelete(ctx context.Context, ids []string) (int64, error) {
	var filter bson.M
	if len(ids) > 0 {
		filter = bson.M{"_id": bson.M{"$in": ids}}
	}
	res, err := s.db.Collection(model.ColOperationLog).DeleteMany(ctx, filter)
	return res.DeletedCount, err
}

// LoginQuery 登录日志查询条件。
type LoginQuery struct {
	Username string
	Status   int
	Start    time.Time
	End      time.Time
	Page     int
	Size     int
}

// LoginList 登录日志分页。
func (s *LogService) LoginList(ctx context.Context, q *LoginQuery) ([]*model.LoginLog, int64, error) {
	filter := bson.M{}
	if q.Username != "" {
		filter["username"] = likeFilter(q.Username)
	}
	if q.Status == model.LoginStatusSuccess || q.Status == model.LoginStatusFailed {
		filter["status"] = q.Status
	}
	if rangeFilter, ok := timeRange("loginAt", q.Start, q.End); ok {
		filter["loginAt"] = rangeFilter
	}
	page, size := normalizePage(q.Page, q.Size)
	return pageFind[model.LoginLog](ctx, s.db.Collection(model.ColLoginLog), filter, page, size, nil)
}

// LoginDelete 批量删除；ids 为空时清空全部。
func (s *LogService) LoginDelete(ctx context.Context, ids []string) (int64, error) {
	var filter bson.M
	if len(ids) > 0 {
		filter = bson.M{"_id": bson.M{"$in": ids}}
	}
	res, err := s.db.Collection(model.ColLoginLog).DeleteMany(ctx, filter)
	return res.DeletedCount, err
}

// timeRange 构造时间范围过滤条件（end 为开区间）。
func timeRange(field string, start, end time.Time) (bson.M, bool) {
	m := bson.M{}
	if !start.IsZero() {
		m["$gte"] = start
	}
	if !end.IsZero() {
		m["$lt"] = end
	}
	if len(m) == 0 {
		return nil, false
	}
	return m, true
}
