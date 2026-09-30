// Package handler HTTP 接口层：通知公告管理。
package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// DictAPI 字典管理接口。

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
