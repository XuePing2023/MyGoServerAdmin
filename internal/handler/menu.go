// Package handler HTTP 接口层：菜单管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// RoleAPI 角色管理接口。

type MenuAPI struct{ svc *service.MenuService }

// NewMenuAPI 创建菜单接口。
func NewMenuAPI(svc *service.MenuService) *MenuAPI { return &MenuAPI{svc: svc} }

// Register 注册路由。
func (a *MenuAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/menus")
	g.GET("/tree", middleware.RequirePerm("sys:menu:list"), a.tree)
	g.POST("", middleware.RequirePerm("sys:menu:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:menu:update"), a.update)
	g.DELETE("/:id", middleware.RequirePerm("sys:menu:delete"), a.delete)
}

// GET /api/v1/menus/tree 菜单树
func (a *MenuAPI) tree(c *gin.Context) {
	tree, err := a.svc.Tree(c.Request.Context())
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, tree)
}

// POST /api/v1/menus 新建菜单
func (a *MenuAPI) create(c *gin.Context) {
	var in service.MenuInput
	if !bindJSON(c, &in) {
		return
	}
	m, err := a.svc.Create(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, m)
}

// PUT /api/v1/menus/:id 更新菜单
func (a *MenuAPI) update(c *gin.Context) {
	var in service.MenuInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/menus/:id 删除菜单
func (a *MenuAPI) delete(c *gin.Context) {
	if err := a.svc.Delete(c.Request.Context(), pathID(c)); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DeptAPI 部门管理接口。
