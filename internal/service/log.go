package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/model"
	"serveradmin/internal/repository"
)

// LogService 操作日志与登录日志。
type LogService struct {
	repo *repository.LogRepository
}

// NewLogService 创建日志服务。
func NewLogService(repo *repository.LogRepository) *LogService {
	return &LogService{repo: repo}
}

// RecordOperation 保存操作日志。
func (s *LogService) RecordOperation(log *model.OperationLog) {
	log.PrepareCreate()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.repo.InsertOperation(ctx, log)
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
	return s.repo.PageOperation(ctx, filter, page, size)
}

// OpDelete 批量删除；ids 为空时清空全部。
func (s *LogService) OpDelete(ctx context.Context, ids []string) (int64, error) {
	return s.repo.DeleteOperation(ctx, ids)
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
	return s.repo.PageLogin(ctx, filter, page, size)
}

// LoginDelete 批量删除；ids 为空时清空全部。
func (s *LogService) LoginDelete(ctx context.Context, ids []string) (int64, error) {
	return s.repo.DeleteLogin(ctx, ids)
}
