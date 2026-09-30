package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/model"
)

// DashboardService 仪表盘统计。
type DashboardService struct {
	db *mongo.Database
}

// NewDashboardService 创建仪表盘服务。db 由包内工具函数共享，
// 这里通过构造器注入 *mongo.Database。
func NewDashboardService(db *mongo.Database) *DashboardService {
	return &DashboardService{db: db}
}

// TrendItem 登录趋势。
type TrendItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// DashboardStats 仪表盘数据。
type DashboardStats struct {
	UserCount       int64                 `json:"userCount"`
	RoleCount       int64                 `json:"roleCount"`
	NoticeCount     int64                 `json:"noticeCount"`
	FileCount       int64                 `json:"fileCount"`
	JobEnabledCount int64                 `json:"jobEnabledCount"`
	OnlineCount     int                   `json:"onlineCount"`
	TodayLogins     int64                 `json:"todayLogins"`
	TodayOpLogs     int64                 `json:"todayOpLogs"`
	Trend           []TrendItem           `json:"trend"`
	RecentOpLogs    []*model.OperationLog `json:"recentOpLogs"`
	RecentNotices   []*model.Notice       `json:"recentNotices"`
}

// Stats 汇总仪表盘数据。
func (s *DashboardService) Stats(ctx context.Context, online *OnlineService) (*DashboardStats, error) {
	st := &DashboardStats{Trend: []TrendItem{}}
	var err error

	if st.UserCount, err = count(ctx, s.db.Collection(model.ColUser), bson.M{}); err != nil {
		return nil, err
	}
	if st.RoleCount, err = count(ctx, s.db.Collection(model.ColRole), bson.M{}); err != nil {
		return nil, err
	}
	if st.NoticeCount, err = count(ctx, s.db.Collection(model.ColNotice), bson.M{}); err != nil {
		return nil, err
	}
	if st.FileCount, err = count(ctx, s.db.Collection(model.ColFile), bson.M{}); err != nil {
		return nil, err
	}
	if st.JobEnabledCount, err = count(ctx, s.db.Collection(model.ColJob), bson.M{"status": model.StatusEnabled}); err != nil {
		return nil, err
	}
	st.OnlineCount = len(online.List())

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	if st.TodayLogins, err = count(ctx, s.db.Collection(model.ColLoginLog),
		bson.M{"status": model.LoginStatusSuccess, "loginAt": bson.M{"$gte": today}}); err != nil {
		return nil, err
	}
	if st.TodayOpLogs, err = count(ctx, s.db.Collection(model.ColOperationLog),
		bson.M{"createdAt": bson.M{"$gte": today}}); err != nil {
		return nil, err
	}

	// 近 7 天登录趋势
	weekStart := today.AddDate(0, 0, -6)
	pipeline := []bson.M{
		{"$match": bson.M{"status": model.LoginStatusSuccess, "loginAt": bson.M{"$gte": weekStart}}},
		{"$group": bson.M{
			"_id":   bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$loginAt"}},
			"count": bson.M{"$sum": 1},
		}},
	}
	cursor, err := s.db.Collection(model.ColLoginLog).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		cursor.Close(ctx)
		return nil, err
	}
	cursor.Close(ctx)
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		counts[r.ID] = r.Count
	}
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		st.Trend = append(st.Trend, TrendItem{Date: key, Count: counts[key]})
	}

	// 最近操作日志
	opCursor, err := s.db.Collection(model.ColOperationLog).Find(ctx, bson.M{},
		findOpts(-1, 8))
	if err != nil {
		return nil, err
	}
	st.RecentOpLogs = make([]*model.OperationLog, 0)
	if err := opCursor.All(ctx, &st.RecentOpLogs); err != nil {
		opCursor.Close(ctx)
		return nil, err
	}
	opCursor.Close(ctx)

	// 最近公告
	nc, err := s.db.Collection(model.ColNotice).Find(ctx, bson.M{"status": 1}, findOpts(-1, 5))
	if err != nil {
		return nil, err
	}
	st.RecentNotices = make([]*model.Notice, 0)
	if err := nc.All(ctx, &st.RecentNotices); err != nil {
		nc.Close(ctx)
		return nil, err
	}
	nc.Close(ctx)

	return st, nil
}
