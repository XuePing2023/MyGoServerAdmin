package api

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// JobAPI 定时任务接口。
type JobAPI struct{ svc *service.JobService }

// NewJobAPI 创建定时任务接口。
func NewJobAPI(svc *service.JobService) *JobAPI { return &JobAPI{svc: svc} }

// Register 注册路由。
func (a *JobAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/jobs")
	g.GET("", middleware.RequirePerm("sys:job:list"), a.list)
	g.GET("/handlers", middleware.RequirePerm("sys:job:list"), a.handlers)
	g.POST("", middleware.RequirePerm("sys:job:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:job:update"), a.update)
	g.DELETE("", middleware.RequirePerm("sys:job:delete"), a.delete)
	g.PUT("/:id/status", middleware.RequirePerm("sys:job:run"), a.setStatus)
	g.PUT("/:id/run", middleware.RequirePerm("sys:job:run"), a.runOnce)

	logs := authed.Group("/job-logs")
	logs.GET("", middleware.RequirePerm("sys:job:list"), a.logs)
}

// GET /api/v1/jobs 任务列表
func (a *JobAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.List(c.Request.Context(), c.Query("name"),
		strconv0(c.Query("status")), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// GET /api/v1/jobs/handlers 可用处理器清单
func (a *JobAPI) handlers(c *gin.Context) {
	response.OK(c, service.HandlerCatalog())
}

// POST /api/v1/jobs 新建任务
func (a *JobAPI) create(c *gin.Context) {
	var in service.JobInput
	if !bindJSON(c, &in) {
		return
	}
	j, err := a.svc.Create(c.Request.Context(), &in)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, j)
}

// PUT /api/v1/jobs/:id 更新任务
func (a *JobAPI) update(c *gin.Context) {
	var in service.JobInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/jobs 批量删除任务
func (a *JobAPI) delete(c *gin.Context) {
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

// PUT /api/v1/jobs/:id/status 启用/停用
func (a *JobAPI) setStatus(c *gin.Context) {
	var in struct {
		Status int `json:"status" binding:"required,oneof=1 2"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.SetStatus(c.Request.Context(), pathID(c), in.Status); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// PUT /api/v1/jobs/:id/run 立即执行一次
func (a *JobAPI) runOnce(c *gin.Context) {
	if err := a.svc.RunOnce(c.Request.Context(), pathID(c)); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, gin.H{"message": "已触发执行，结果请查看任务日志"})
}

// GET /api/v1/job-logs 任务执行日志
func (a *JobAPI) logs(c *gin.Context) {
	page, size := pageParams(c)
	list, total, err := a.svc.JobLogs(c.Request.Context(), c.Query("jobName"), page, size)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}
