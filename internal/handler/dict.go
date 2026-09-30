// Package handler HTTP 接口层：字典管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// DictAPI 字典管理接口。
type DictAPI struct{ svc *service.DictService }

// NewDictAPI 创建字典接口。
func NewDictAPI(svc *service.DictService) *DictAPI { return &DictAPI{svc: svc} }

// Register 注册路由。
func (a *DictAPI) Register(authed *gin.RouterGroup) {
	types := authed.Group("/dict-types")
	{
		types.GET("", middleware.RequirePerm("sys:dict:list"), a.typeList)
		types.POST("", middleware.RequirePerm("sys:dict:create"), a.typeCreate)
		types.PUT("/:id", middleware.RequirePerm("sys:dict:update"), a.typeUpdate)
		types.DELETE("", middleware.RequirePerm("sys:dict:delete"), a.typeDelete)
	}
	items := authed.Group("/dict-items")
	{
		items.GET("", middleware.RequirePerm("sys:dict:list"), a.itemList)
		items.POST("", middleware.RequirePerm("sys:dict:create"), a.itemCreate)
		items.PUT("/:id", middleware.RequirePerm("sys:dict:update"), a.itemUpdate)
		items.DELETE("", middleware.RequirePerm("sys:dict:delete"), a.itemDelete)
	}
	// 登录即可读取字典项（供下拉框使用）
	authed.GET("/dicts/:code", a.byCode)
}

// GET /api/v1/dict-types 字典类型列表
func (a *DictAPI) typeList(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.TypeList(c.Request.Context(), c.Query("name"), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// POST /api/v1/dict-types 新建字典类型
func (a *DictAPI) typeCreate(c *gin.Context) {
	var in service.DictTypeInput
	if !bindJSON(c, &in) {
		return
	}
	t, err := a.svc.TypeCreate(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, t)
}

// PUT /api/v1/dict-types/:id 更新字典类型
func (a *DictAPI) typeUpdate(c *gin.Context) {
	var in service.DictTypeInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.TypeUpdate(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/dict-types 批量删除字典类型
func (a *DictAPI) typeDelete(c *gin.Context) {
	var in idsBody
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.TypeDelete(c.Request.Context(), in.IDs); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// GET /api/v1/dict-items 字典项列表
func (a *DictAPI) itemList(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.ItemList(c.Request.Context(), c.Query("typeCode"), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// POST /api/v1/dict-items 新建字典项
func (a *DictAPI) itemCreate(c *gin.Context) {
	var in service.DictItemInput
	if !bindJSON(c, &in) {
		return
	}
	i, err := a.svc.ItemCreate(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, i)
}

// PUT /api/v1/dict-items/:id 更新字典项
func (a *DictAPI) itemUpdate(c *gin.Context) {
	var in service.DictItemInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.ItemUpdate(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/dict-items 批量删除字典项
func (a *DictAPI) itemDelete(c *gin.Context) {
	var in idsBody
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.ItemDelete(c.Request.Context(), in.IDs); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// GET /api/v1/dicts/:code 按编码取启用的字典项
func (a *DictAPI) byCode(c *gin.Context) {
	list, err := a.svc.GetByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, list)
}

// ConfigAPI 系统参数接口。
