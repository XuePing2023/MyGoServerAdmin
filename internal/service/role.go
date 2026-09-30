package service

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/cache"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

var roleCodeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,32}$`)

// RoleService 角色管理。
type RoleService struct {
	repo  *repository.RoleRepository
	perm  *PermService
	cache *cache.Helper
}

// NewRoleService 创建角色服务。
func NewRoleService(repo *repository.RoleRepository, helper *cache.Helper) *RoleService {
	return &RoleService{repo: repo, cache: helper}
}

// SetPerm 注入权限缓存。
func (s *RoleService) SetPerm(p *PermService) { s.perm = p }

// RoleInput 创建/更新角色。
type RoleInput struct {
	Name   string `json:"name" binding:"required,max=32"`
	Code   string `json:"code" binding:"required,min=2,max=32"`
	Sort   int    `json:"sort"`
	Status int    `json:"status" binding:"required,oneof=1 2"`
	Remark string `json:"remark"`
}

// RoleQuery 查询条件。
type RoleQuery struct {
	Name   string
	Status int
	All    bool // true 时不分页返回全部（用于下拉选择）
	Page   int
	Size   int
}

// List 查询角色。
func (s *RoleService) List(ctx context.Context, q *RoleQuery) ([]*model.Role, int64, error) {
	filter := statusFilter(q.Status)
	if q.Name != "" {
		filter["name"] = likeFilter(q.Name)
	}
	if q.All {
		list, err := s.repo.FindAll(ctx, filter,
			bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
		if err != nil {
			return nil, 0, err
		}
		return list, int64(len(list)), nil
	}
	page, size := normalizePage(q.Page, q.Size)
	return s.repo.Page(ctx, filter, page, size,
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
}

// Get 角色详情。
func (s *RoleService) Get(ctx context.Context, id string) (*model.Role, error) {
	r, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, errs.NotFound("角色不存在")
	}
	return r, nil
}

// Create 创建角色。
func (s *RoleService) Create(ctx context.Context, in *RoleInput) (*model.Role, error) {
	if !roleCodeRe.MatchString(in.Code) {
		return nil, errs.BadRequest("角色编码需以字母开头，仅含字母数字_-，长度 2-32")
	}
	n, err := s.repo.Count(ctx, bson.M{"code": in.Code})
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, errs.BadRequest("角色编码已存在")
	}
	r := &model.Role{
		Name: in.Name, Code: in.Code, Sort: in.Sort,
		Status: in.Status, Remark: in.Remark, Menus: []string{},
	}
	r.PrepareCreate()
	if err := s.repo.Insert(ctx, r); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return r, nil
}

// Update 更新角色（内置角色仅允许改名称/备注/排序）。
func (s *RoleService) Update(ctx context.Context, id string, in *RoleInput) error {
	r, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if r.BuiltIn {
		if in.Code != r.Code || in.Status != r.Status {
			return errs.Forbidden("内置角色不允许修改编码或状态")
		}
	} else {
		if in.Code != r.Code {
			if !roleCodeRe.MatchString(in.Code) {
				return errs.BadRequest("角色编码格式无效")
			}
			n, err := s.repo.Count(ctx, bson.M{"code": in.Code, "_id": bson.M{"$ne": id}})
			if err != nil {
				return err
			}
			if n > 0 {
				return errs.BadRequest("角色编码已存在")
			}
		}
	}
	update := bson.M{
		"name": in.Name, "code": in.Code, "sort": in.Sort,
		"status": in.Status, "remark": in.Remark, "updatedAt": time.Now(),
	}
	if r.BuiltIn {
		update["code"] = r.Code
		update["status"] = r.Status
	}
	if err := s.repo.UpdateSet(ctx, bson.M{"_id": id}, update); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// Delete 批量删除角色。
func (s *RoleService) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的角色")
	}
	list, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}

	codes := make([]string, 0)
	for _, r := range list {
		if r.BuiltIn {
			return errs.Forbidden("内置角色 [%s] 不允许删除", r.Name)
		}
		codes = append(codes, r.Code)
	}
	n, err := s.repo.CountUsersByRoleCodes(ctx, codes)
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.BadRequest("有 %d 个用户正在使用待删除的角色，请先调整", n)
	}
	if _, err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// AssignMenus 为角色分配菜单/权限。
func (s *RoleService) AssignMenus(ctx context.Context, id string, menuIDs []string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if menuIDs == nil {
		menuIDs = []string{}
	}
	if err := s.repo.UpdateSet(ctx, bson.M{"_id": id},
		bson.M{"menus": menuIDs, "updatedAt": time.Now()}); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

func (s *RoleService) invalidate(ctx context.Context) {
	if s.perm != nil {
		s.perm.InvalidateAll()
	}
	// 角色变更影响按角色缓存的菜单树（serveradmin:v1:role:*）
	s.cache.Invalidate(ctx, cache.RolePrefix())
}
