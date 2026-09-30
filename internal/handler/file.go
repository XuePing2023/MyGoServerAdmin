package handler

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// FileAPI 文件管理接口。
type FileAPI struct{ svc *service.FileService }

// NewFileAPI 创建文件接口。
func NewFileAPI(svc *service.FileService) *FileAPI { return &FileAPI{svc: svc} }

// Register 注册路由。
func (a *FileAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/files")
	g.GET("", middleware.RequirePerm("sys:file:list"), a.list)
	g.POST("/upload", middleware.RequirePerm("sys:file:upload"), a.upload)
	g.GET("/:id/download", a.download)
	g.DELETE("", middleware.RequirePerm("sys:file:delete"), a.delete)
}

// GET /api/v1/files 文件列表
func (a *FileAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.List(c.Request.Context(), c.Query("name"), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// POST /api/v1/files/upload 上传文件（multipart 字段名 file）
func (a *FileAPI) upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.Handle(c, errs.BadRequest("请选择要上传的文件（字段名 file）"))
		return
	}
	id := middleware.FromContext(c)
	rec, err := a.svc.Upload(c.Request.Context(), fh, id.Username)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, rec)
}

// GET /api/v1/files/:id/download 下载文件
func (a *FileAPI) download(c *gin.Context) {
	rec, err := a.svc.Get(c.Request.Context(), pathID(c))
	if err != nil {
		response.Handle(c, err)
		return
	}
	c.FileAttachment(a.svc.AbsPath(rec), rec.OriginalName)
}

// DELETE /api/v1/files 批量删除文件
func (a *FileAPI) delete(c *gin.Context) {
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
