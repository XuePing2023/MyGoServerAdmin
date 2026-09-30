package service

import (
	"context"
	"time"

	"serveradmin/internal/model"
	"serveradmin/internal/repository"
)

// DashboardService 仪表盘统计。
type DashboardService struct {
	repo *repository.DashboardRepository
}

// NewDashboardService 创建仪表盘服务。
func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
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

	if st.UserCount, err = s.repo.UserCount(ctx); err != nil {
		return nil, err
	}
	if st.RoleCount, err = s.repo.RoleCount(ctx); err != nil {
		return nil, err
	}
	if st.NoticeCount, err = s.repo.NoticeCount(ctx); err != nil {
		return nil, err
	}
	if st.FileCount, err = s.repo.FileCount(ctx); err != nil {
		return nil, err
	}
	if st.JobEnabledCount, err = s.repo.EnabledJobCount(ctx); err != nil {
		return nil, err
	}
	st.OnlineCount = len(online.List())

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	if st.TodayLogins, err = s.repo.TodayLoginCount(ctx, today); err != nil {
		return nil, err
	}
	if st.TodayOpLogs, err = s.repo.TodayOpLogCount(ctx, today); err != nil {
		return nil, err
	}

	// 近 7 天登录趋势
	weekStart := today.AddDate(0, 0, -6)
	counts, err := s.repo.LoginTrend(ctx, weekStart)
	if err != nil {
		return nil, err
	}
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		st.Trend = append(st.Trend, TrendItem{Date: key, Count: counts[key]})
	}

	// 最近操作日志
	if st.RecentOpLogs, err = s.repo.RecentOpLogs(ctx, 8); err != nil {
		return nil, err
	}

	// 最近公告
	if st.RecentNotices, err = s.repo.RecentNotices(ctx, 5); err != nil {
		return nil, err
	}

	return st, nil
}
