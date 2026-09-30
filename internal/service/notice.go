package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

// NoticeService 通知公告管理。
type NoticeService struct {
	repo *repository.NoticeRepository
}

// NewNoticeService 创建通知公告服务。
func NewNoticeService(repo *repository.NoticeRepository) *NoticeService {
	return &NoticeService{repo: repo}
}

// NoticeInput 创建/更新。
type NoticeInput struct {
	Title   string `json:"title" binding:"required,max=64"`
	Type    int    `json:"type" binding:"required,oneof=1 2"`
	Content string `json:"content" binding:"required"`
	Status  int    `json:"status" binding:"required,oneof=1 2"`
}

// List 分页。
func (s *NoticeService) List(ctx context.Context, title string, ntype, status, page, size int) ([]*model.Notice, int64, error) {
	filter := bson.M{}
	if title != "" {
		filter["title"] = likeFilter(title)
	}
	if ntype == model.NoticeTypeNotice || ntype == model.NoticeTypeAnnouce {
		filter["type"] = ntype
	}
	if status == 1 || status == 2 {
		filter["status"] = status
	}
	page, size = normalizePage(page, size)
	return s.repo.Page(ctx, filter, page, size, nil)
}

// Get 详情。
func (s *NoticeService) Get(ctx context.Context, id string) (*model.Notice, error) {
	n, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, errs.NotFound("通知不存在")
	}
	return n, nil
}

// Create 创建。
func (s *NoticeService) Create(ctx context.Context, in *NoticeInput, publisher string) (*model.Notice, error) {
	n := &model.Notice{
		Title: in.Title, Type: in.Type, Content: in.Content,
		Status: in.Status, Publisher: publisher,
	}
	n.PrepareCreate()
	if err := s.repo.Insert(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// Update 更新。
func (s *NoticeService) Update(ctx context.Context, id string, in *NoticeInput) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return s.repo.UpdateSet(ctx, bson.M{"_id": id},
		bson.M{"title": in.Title, "type": in.Type, "content": in.Content,
			"status": in.Status, "updatedAt": time.Now()})
}

// Delete 批量删除。
func (s *NoticeService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的公告")
	}
	_, err := s.repo.DeleteByIDs(ctx, ids)
	return err
}

// Recent 最近发布的公告。
func (s *NoticeService) Recent(ctx context.Context, limit int) ([]*model.Notice, error) {
	return s.repo.FindRecent(ctx, limit)
}
