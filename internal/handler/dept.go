// Package handler HTTP 接口层：部门管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// RoleAPI 角色管理接口。

type DeptAPI struct{ svc *service.DepartmentService }

// NewDeptAPI 创建部门接口。
func NewDeptAPI(svc *service.DepartmentService) *DeptAPI { return &DeptAPI{svc: svc} }

// Register 注册路由。
func (a *DeptAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/departments")
	g.GET("/tree", middleware.RequirePerm("sys:dept:list"), a.tree)
	g.POST("", middleware.RequirePerm("sys:dept:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:dept:update"), a.update)
	g.DELETE("/:id", middleware.RequirePerm("sys:dept:delete"), a.delete)
}

// GET /api/v1/departments/tree 部门树
func (a *DeptAPI) tree(c *gin.Context) {
	tree, err := a.svc.Tree(c.Request.Context())
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, tree)
}

// POST /api/v1/departments 新建部门
func (a *DeptAPI) create(c *gin.Context) {
	var in service.DeptInput
	if !bindJSON(c, &in) {
		return
	}
	d, err := a.svc.Create(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, d)
}

// PUT /api/v1/departments/:id 更新部门
func (a *DeptAPI) update(c *gin.Context) {
	var in service.DeptInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/departments/:id 删除部门
func (a *DeptAPI) delete(c *gin.Context) {
	if err := a.svc.Delete(c.Request.Context(), pathID(c)); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}
