package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"serveradmin/internal/model"
)

// DashboardRepository 仪表盘跨集合统计查询。
type DashboardRepository struct {
	db *mongo.Database
}

// NewDashboardRepository 创建仪表盘仓库。
func NewDashboardRepository(db *mongo.Database) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) count(ctx context.Context, coll string, filter bson.M) (int64, error) {
	return r.db.Collection(coll).CountDocuments(ctx, filter)
}

// UserCount 用户总数。
func (r *DashboardRepository) UserCount(ctx context.Context) (int64, error) {
	return r.count(ctx, model.ColUser, bson.M{})
}

// RoleCount 角色总数。
func (r *DashboardRepository) RoleCount(ctx context.Context) (int64, error) {
	return r.count(ctx, model.ColRole, bson.M{})
}

// NoticeCount 公告总数。
func (r *DashboardRepository) NoticeCount(ctx context.Context) (int64, error) {
	return r.count(ctx, model.ColNotice, bson.M{})
}

// FileCount 文件总数。
func (r *DashboardRepository) FileCount(ctx context.Context) (int64, error) {
	return r.count(ctx, model.ColFile, bson.M{})
}

// EnabledJobCount 启用的定时任务数。
func (r *DashboardRepository) EnabledJobCount(ctx context.Context) (int64, error) {
	return r.count(ctx, model.ColJob, bson.M{"status": model.StatusEnabled})
}

// TodayLoginCount 当日成功登录数。
func (r *DashboardRepository) TodayLoginCount(ctx context.Context, today time.Time) (int64, error) {
	return r.count(ctx, model.ColLoginLog,
		bson.M{"status": model.LoginStatusSuccess, "loginAt": bson.M{"$gte": today}})
}

// TodayOpLogCount 当日操作日志数。
func (r *DashboardRepository) TodayOpLogCount(ctx context.Context, today time.Time) (int64, error) {
	return r.count(ctx, model.ColOperationLog,
		bson.M{"createdAt": bson.M{"$gte": today}})
}

// trendRow 登录趋势聚合行。
type trendRow struct {
	ID    string `bson:"_id"`
	Count int64  `bson:"count"`
}

// LoginTrend 近 7 天登录趋势，返回 日期 -> 登录数。
func (r *DashboardRepository) LoginTrend(ctx context.Context, weekStart time.Time) (map[string]int64, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"status": model.LoginStatusSuccess, "loginAt": bson.M{"$gte": weekStart}}},
		{"$group": bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$loginAt"}},
			"count": bson.M{"$sum": 1},
		}},
	}
	cursor, err := r.db.Collection(model.ColLoginLog).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var rows []trendRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.ID] = row.Count
	}
	return counts, nil
}

// RecentOpLogs 最近操作日志。
func (r *DashboardRepository) RecentOpLogs(ctx context.Context, limit int) ([]*model.OperationLog, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.db.Collection(model.ColOperationLog).Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*model.OperationLog, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// RecentNotices 最近启用公告。
func (r *DashboardRepository) RecentNotices(ctx context.Context, limit int) ([]*model.Notice, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))
	cursor, err := r.db.Collection(model.ColNotice).Find(ctx, bson.M{"status": 1}, opts)
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
