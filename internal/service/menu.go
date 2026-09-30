package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"serveradmin/internal/cache"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/repository"
)

// MenuService 菜单/权限管理。
type MenuService struct {
	repo  *repository.MenuRepository
	role  *repository.RoleRepository
	perm  *PermService // 变更后失效权限缓存
	cache *cache.Helper
}

// NewMenuService 创建菜单服务。
func NewMenuService(menu *repository.MenuRepository, role *repository.RoleRepository, helper *cache.Helper) *MenuService {
	return &MenuService{repo: menu, role: role, cache: helper}
}

// SetPerm 注入权限缓存服务（避免构建期循环）。
func (s *MenuService) SetPerm(p *PermService) { s.perm = p }

// MenuInput 菜单创建/更新请求。
type MenuInput struct {
	ParentID  string `json:"parentId"`
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Perm      string `json:"perm"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Type      int    `json:"type" binding:"required,oneof=1 2 3"`
	Visible   bool   `json:"visible"`
	Status    int    `json:"status" binding:"required,oneof=1 2"`
}

// Tree 全量菜单树（含按钮），管理端使用。
func (s *MenuService) Tree(ctx context.Context) ([]*model.Menu, error) {
	list, err := s.allMenus(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	return buildMenuTree(list, ""), nil
}

// GetForRoles 根据角色编码返回可见菜单树（目录/页面，不含按钮），
// 超级管理员返回全部。
// ROLE 策略：按角色区分，Key: serveradmin:v1:role:{roles}:menu，
// 菜单/角色变更时整体失效。
func (s *MenuService) GetForRoles(ctx context.Context, codes []string) ([]*model.Menu, error) {
	return cache.GetJSON(s.cache, ctx, cache.RoleKey(cache.RoleSegment(codes), "menu"), s.cache.TTL.Role,
		func(ctx context.Context) ([]*model.Menu, error) {
			return s.loadForRoles(ctx, codes)
		})
}

func (s *MenuService) loadForRoles(ctx context.Context, codes []string) ([]*model.Menu, error) {
	for _, c := range codes {
		if c == model.SuperRoleCode {
			list, err := s.allMenus(ctx, bson.M{"status": model.StatusEnabled})
			if err != nil {
				return nil, err
			}
			return buildMenuTree(list, ""), nil
		}
	}

	ids := make([]string, 0)
	if len(codes) > 0 {
		roles, err := s.role.FindByCodes(ctx, codes)
		if err != nil {
			return nil, err
		}
		for _, r := range roles {
			ids = append(ids, r.Menus...)
		}
	}
	if len(ids) == 0 {
		return []*model.Menu{}, nil
	}

	list, err := s.allMenus(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*model.Menu, len(list))
	for _, m := range list {
		byID[m.ID] = m
	}
	// 收集选中菜单及其全部祖先（保证目录链完整）
	keep := make(map[string]bool)
	var collect func(id string)
	collect = func(id string) {
		if keep[id] {
			return
		}
		m, ok := byID[id]
		if !ok {
			return
		}
		keep[id] = true
		if m.ParentID != "" {
			collect(m.ParentID)
		}
	}
	for _, id := range ids {
		collect(id)
	}

	filtered := make([]*model.Menu, 0)
	for _, m := range list {
		if keep[m.ID] && m.Status == model.StatusEnabled && m.Type != model.MenuTypeButton {
			filtered = append(filtered, m)
		}
	}
	return buildMenuTree(filtered, ""), nil
}

func (s *MenuService) allMenus(ctx context.Context, filter bson.M) ([]*model.Menu, error) {
	return s.repo.FindAll(ctx, filter,
		bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
}

// Create 新建菜单。
func (s *MenuService) Create(ctx context.Context, in *MenuInput) (*model.Menu, error) {
	if in.ParentID != "" {
		if err := s.ensureExists(ctx, in.ParentID); err != nil {
			return nil, err
		}
	}
	m := &model.Menu{
		ParentID: in.ParentID, Name: in.Name, Path: in.Path, Component: in.Component,
		Perm: in.Perm, Icon: in.Icon, Sort: in.Sort, Type: in.Type,
		Visible: in.Visible, Status: in.Status,
	}
	m.PrepareCreate()
	if err := s.repo.Insert(ctx, m); err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return m, nil
}

// Update 更新菜单。
func (s *MenuService) Update(ctx context.Context, id string, in *MenuInput) error {
	m, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if in.ParentID == id {
		return errs.BadRequest("上级菜单不能是自身")
	}
	if in.ParentID != "" {
		if err := s.ensureExists(ctx, in.ParentID); err != nil {
			return err
		}
		// 防止把自己的子孙设为上级
		descendants, err := s.repo.FindDescendantIDs(ctx, []string{m.ID})
		if err != nil {
			return err
		}
		for _, d := range descendants {
			if d == in.ParentID {
				return errs.BadRequest("不能把自身的子菜单设为上级菜单")
			}
		}
	}
	update := bson.M{
		"parentId": in.ParentID, "name": in.Name, "path": in.Path,
		"component": in.Component, "perm": in.Perm, "icon": in.Icon,
		"sort": in.Sort, "type": in.Type, "visible": in.Visible,
		"status": in.Status, "updatedAt": time.Now(),
	}
	if err := s.repo.UpdateSet(ctx, bson.M{"_id": id}, update); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// Delete 删除菜单（有子菜单时禁止）。
func (s *MenuService) Delete(ctx context.Context, id string) error {
	n, err := s.repo.Count(ctx, bson.M{"parentId": id})
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.BadRequest("存在子菜单，请先删除子菜单")
	}
	deleted, err := s.repo.Delete(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errs.NotFound("菜单不存在")
	}
	s.invalidate(ctx)
	return nil
}

func (s *MenuService) get(ctx context.Context, id string) (*model.Menu, error) {
	m, err := s.repo.FindOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, errs.NotFound("菜单不存在")
	}
	return m, nil
}

func (s *MenuService) ensureExists(ctx context.Context, id string) error {
	_, err := s.get(ctx, id)
	return err
}

func (s *MenuService) invalidate(ctx context.Context) {
	if s.perm != nil {
		s.perm.InvalidateAll()
	}
	// 菜单变更影响角色菜单树（ROLE）与菜单管理树（PUBLIC）缓存
	s.cache.Invalidate(ctx, cache.RolePrefix(), cache.BizPrefix("menu"))
}

// buildMenuTree 组装树（parentID 为根）。
func buildMenuTree(list []*model.Menu, parentID string) []*model.Menu {
	tree := make([]*model.Menu, 0)
	for _, m := range list {
		if m.ParentID == parentID {
			m.Children = buildMenuTree(list, m.ID)
			tree = append(tree, m)
		}
	}
	return tree
}
