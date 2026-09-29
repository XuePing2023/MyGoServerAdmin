package api

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
	response.OK(c, a.svc.Public(c.Request.Context()))
}

// NoticeAPI 通知公告接口。
type NoticeAPI struct{ svc *service.NoticeService }

// NewNoticeAPI 创建公告接口。
func NewNoticeAPI(svc *service.NoticeService) *NoticeAPI { return &NoticeAPI{svc: svc} }

// Register 注册路由。
func (a *NoticeAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/notices")
	g.GET("", middleware.RequirePerm("sys:notice:list"), a.list)
	g.POST("", middleware.RequirePerm("sys:notice:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:notice:update"), a.update)
	g.DELETE("", middleware.RequirePerm("sys:notice:delete"), a.delete)
}

// GET /api/v1/notices 公告列表
func (a *NoticeAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.List(c.Request.Context(), c.Query("title"),
		strconv0(c.Query("type")), strconv0(c.Query("status")), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// POST /api/v1/notices 新建公告
func (a *NoticeAPI) create(c *gin.Context) {
	var in service.NoticeInput
	if !bindJSON(c, &in) {
		return
	}
	id := middleware.FromContext(c)
	n, err := a.svc.Create(c.Request.Context(), &in, id.Username)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, n)
}

// PUT /api/v1/notices/:id 更新公告
func (a *NoticeAPI) update(c *gin.Context) {
	var in service.NoticeInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/notices 批量删除公告
func (a *NoticeAPI) delete(c *gin.Context) {
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
