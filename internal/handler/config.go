// Package handler HTTP 接口层：系统参数管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// DictAPI 字典管理接口。

type ConfigAPI struct{ svc *service.SysConfigService }

// NewConfigAPI 创建系统参数接口。
func NewConfigAPI(svc *service.SysConfigService) *ConfigAPI { return &ConfigAPI{svc: svc} }

// Register 注册路由。
func (a *ConfigAPI) Register(public, authed *gin.RouterGroup) {
	public.GET("/configs/public", a.publicInfo)

	g := authed.Group("/configs")
	g.GET("", middleware.RequirePerm("sys:config:list"), a.list)
	g.POST("", middleware.RequirePerm("sys:config:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:config:update"), a.update)
	g.DELETE("", middleware.RequirePerm("sys:config:delete"), a.delete)
}

// GET /api/v1/configs 参数列表
func (a *ConfigAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.List(c.Request.Context(), c.Query("keyword"), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// POST /api/v1/configs 新建参数
func (a *ConfigAPI) create(c *gin.Context) {
	var in service.ConfigInput
	if !bindJSON(c, &in) {
		return
	}
	cfg, err := a.svc.Create(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, cfg)
}

// PUT /api/v1/configs/:id 更新参数
func (a *ConfigAPI) update(c *gin.Context) {
	var in service.ConfigInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/configs 批量删除参数
func (a *ConfigAPI) delete(c *gin.Context) {
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

// GET /api/v1/configs/public 公开信息（登录页用）
func (a *ConfigAPI) publicInfo(c *gin.Context) {
	info, err := a.svc.Public(c.Request.Context())
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, info)
}

// NoticeAPI 通知公告接口。
