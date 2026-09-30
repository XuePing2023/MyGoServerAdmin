// Package handler HTTP 接口层：角色管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// RoleAPI 角色管理接口。

type RoleAPI struct{ svc *service.RoleService }

// NewRoleAPI 创建角色接口。
func NewRoleAPI(svc *service.RoleService) *RoleAPI { return &RoleAPI{svc: svc} }

// Register 注册路由。
func (a *RoleAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/roles")
	g.GET("", middleware.RequirePerm("sys:role:list"), a.list)
	g.GET("/:id", middleware.RequirePerm("sys:role:list"), a.get)
	g.POST("", middleware.RequirePerm("sys:role:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:role:update"), a.update)
	g.DELETE("", middleware.RequirePerm("sys:role:delete"), a.delete)
	g.PUT("/:id/menus", middleware.RequirePerm("sys:role:assign"), a.assignMenus)
}

// GET /api/v1/roles 角色列表（all=1 时不分页）
func (a *RoleAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.List(c.Request.Context(), &service.RoleQuery{
		Name:   c.Query("name"),
		Status: strconv0(c.Query("status")),
		All:    c.Query("all") == "1",
		Page:   page,
		Size:   size,
	})
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// GET /api/v1/roles/:id 角色详情
func (a *RoleAPI) get(c *gin.Context) {
	r, err := a.svc.Get(c.Request.Context(), pathID(c))
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, r)
}

// POST /api/v1/roles 新建角色
func (a *RoleAPI) create(c *gin.Context) {
	var in service.RoleInput
	if !bindJSON(c, &in) {
		return
	}
	r, err := a.svc.Create(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, r)
}

// PUT /api/v1/roles/:id 更新角色
func (a *RoleAPI) update(c *gin.Context) {
	var in service.RoleInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/roles 批量删除
func (a *RoleAPI) delete(c *gin.Context) {
	var in idsBody
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Delete(c.Request.Context(), in.IDs); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// PUT /api/v1/roles/:id/menus 分配菜单权限
func (a *RoleAPI) assignMenus(c *gin.Context) {
	var in struct {
		MenuIDs []string `json:"menuIds"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.AssignMenus(c.Request.Context(), pathID(c), in.MenuIDs); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// MenuAPI 菜单管理接口。
