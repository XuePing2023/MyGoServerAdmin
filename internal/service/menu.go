package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
)

// MenuService 菜单/权限管理。
type MenuService struct {
	db   *mongo.Database
	perm *PermService // 变更后失效权限缓存
}

// NewMenuService 创建菜单服务。
func NewMenuService(db *mongo.Database) *MenuService {
	return &MenuService{db: db}
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
func (s *MenuService) GetForRoles(ctx context.Context, codes []string) ([]*model.Menu, error) {
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
		cursor, err := s.db.Collection(model.ColRole).Find(ctx, bson.M{"code": bson.M{"$in": codes}})
		if err != nil {
			return nil, err
		}
		var roles []*model.Role
		if err := cursor.All(ctx, &roles); err != nil {
			cursor.Close(ctx)
			return nil, err
		}
		cursor.Close(ctx)
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
	opts := options.Find().SetSort(bson.D{{Key: "sort", Value: 1}, {Key: "createdAt", Value: 1}})
	cursor, err := s.db.Collection(model.ColMenu).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]*model.Menu, 0)
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
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
	_, err := s.db.Collection(model.ColMenu).InsertOne(ctx, m)
	if err != nil {
		return nil, err
	}
	s.invalidate()
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
		descendants, err := s.descendantIDs(ctx, []string{m.ID})
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
	_, err = s.db.Collection(model.ColMenu).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	s.invalidate()
	return nil
}

// Delete 删除菜单（有子菜单时禁止）。
func (s *MenuService) Delete(ctx context.Context, id string) error {
	n, err := count(ctx, s.db.Collection(model.ColMenu), bson.M{"parentId": id})
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.BadRequest("存在子菜单，请先删除子菜单")
	}
	res, err := s.db.Collection(model.ColMenu).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errs.NotFound("菜单不存在")
	}
	s.invalidate()
	return nil
}

func (s *MenuService) get(ctx context.Context, id string) (*model.Menu, error) {
	var m model.Menu
	if err := findOne(ctx, s.db.Collection(model.ColMenu), bson.M{"_id": id}, &m); err != nil {
		return nil, errs.NotFound("菜单不存在")
	}
	return &m, nil
}

func (s *MenuService) ensureExists(ctx context.Context, id string) error {
	_, err := s.get(ctx, id)
	return err
}

func (s *MenuService) descendantIDs(ctx context.Context, roots []string) ([]string, error) {
	result := make([]string, 0)
	frontier := roots
	for len(frontier) > 0 {
		cursor, err := s.db.Collection(model.ColMenu).Find(ctx, bson.M{"parentId": bson.M{"$in": frontier}})
		if err != nil {
			return nil, err
		}
		var children []*model.Menu
		if err := cursor.All(ctx, &children); err != nil {
			cursor.Close(ctx)
			return nil, err
		}
		cursor.Close(ctx)
		frontier = frontier[:0]
		for _, c := range children {
			result = append(result, c.ID)
			frontier = append(frontier, c.ID)
		}
	}
	return result, nil
}

func (s *MenuService) invalidate() {
	if s.perm != nil {
		s.perm.InvalidateAll()
	}
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
