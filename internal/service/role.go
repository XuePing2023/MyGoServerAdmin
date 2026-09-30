package service

import (
	"context"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"serveradmin/internal/cache"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
)

var roleCodeRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,31}$`)

// RoleService 角色管理。
type RoleService struct {
	db    *mongo.Database
	perm  *PermService
	cache *cache.Helper
}

// NewRoleService 创建角色服务。
func NewRoleService(db *mongo.Database, helper *cache.Helper) *RoleService {
	return &RoleService{db: db, cache: helper}
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
		opts := options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
		cursor, err := s.db.Collection(model.ColRole).Find(ctx, filter, opts)
		if err != nil {
			return nil, 0, err
		}
		defer cursor.Close(ctx)
		list := make([]*model.Role, 0)
		if err := cursor.All(ctx, &list); err != nil {
			return nil, 0, err
		}
		return list, int64(len(list)), nil
	}
	page, size := normalizePage(q.Page, q.Size)
	return pageFind[model.Role](ctx, s.db.Collection(model.ColRole), filter, page, size,
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
}

// Get 角色详情。
func (s *RoleService) Get(ctx context.Context, id string) (*model.Role, error) {
	var r model.Role
	if err := findOne(ctx, s.db.Collection(model.ColRole), bson.M{"_id": id}, &r); err != nil {
		return nil, errs.NotFound("角色不存在")
	}
	return &r, nil
}

// Create 创建角色。
func (s *RoleService) Create(ctx context.Context, in *RoleInput) (*model.Role, error) {
	if !roleCodeRe.MatchString(in.Code) {
		return nil, errs.BadRequest("角色编码需以字母开头，仅含字母数字_-，长度 2-32")
	}
	n, err := count(ctx, s.db.Collection(model.ColRole), bson.M{"code": in.Code})
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
	_, err = s.db.Collection(model.ColRole).InsertOne(ctx, r)
	return r, err
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
			n, err := count(ctx, s.db.Collection(model.ColRole), bson.M{"code": in.Code, "_id": bson.M{"$ne": id}})
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
	_, err = s.db.Collection(model.ColRole).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
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
	roles, err := s.db.Collection(model.ColRole).Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return err
	}
	list := make([]*model.Role, 0)
	if err := roles.All(ctx, &list); err != nil {
		roles.Close(ctx)
		return err
	}
	roles.Close(ctx)

	codes := make([]string, 0)
	for _, r := range list {
		if r.BuiltIn {
			return errs.Forbidden("内置角色 [%s] 不允许删除", r.Name)
		}
		codes = append(codes, r.Code)
	}
	n, err := count(ctx, s.db.Collection(model.ColUser), bson.M{"roles": bson.M{"$in": codes}})
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.BadRequest("有 %d 个用户正在使用待删除的角色，请先调整", n)
	}
	_, err = s.db.Collection(model.ColRole).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
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
	_, err := s.db.Collection(model.ColRole).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"menus": menuIDs, "updatedAt": time.Now()}})
	if err != nil {
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
