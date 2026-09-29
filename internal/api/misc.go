package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// DashboardAPI 仪表盘接口。
type DashboardAPI struct {
	svc    *service.DashboardService
	online *service.OnlineService
}

// NewDashboardAPI 创建仪表盘接口。
func NewDashboardAPI(svc *service.DashboardService, online *service.OnlineService) *DashboardAPI {
	return &DashboardAPI{svc: svc, online: online}
}

// Register 注册路由（登录即可查看）。
func (a *DashboardAPI) Register(authed *gin.RouterGroup) {
	authed.GET("/dashboard/stats", a.stats)
}

// GET /api/v1/dashboard/stats 仪表盘统计
func (a *DashboardAPI) stats(c *gin.Context) {
	st, err := a.svc.Stats(c.Request.Context(), a.online)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, st)
}

// MonitorAPI 系统监控接口。
type MonitorAPI struct {
	svc *service.MonitorService
}

// NewMonitorAPI 创建监控接口。
func NewMonitorAPI(svc *service.MonitorService) *MonitorAPI { return &MonitorAPI{svc: svc} }

// Register 注册路由。
func (a *MonitorAPI) Register(authed *gin.RouterGroup) {
	authed.GET("/monitor/server", middleware.RequirePerm("sys:monitor"), a.server)
}

// GET /api/v1/monitor/server 服务器监控信息
func (a *MonitorAPI) server(c *gin.Context) {
	// CPU 占用需要两次采样，这里用 200ms 窗口采样
	pct, err := cpu.Percent(200*time.Millisecond, false)
	var cpuPct float64
	if err == nil && len(pct) > 0 {
		cpuPct = pct[0]
	}
	info, err := a.svc.Info(c.Request.Context(), cpuPct)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, info)
}

// SystemAPI 健康检查（公开，不走认证）。
type SystemAPI struct {
	pingDB func(ctx context.Context) error
}

// NewSystemAPI 创建系统接口，pingDB 为数据库连通性探测。
func NewSystemAPI(pingDB func(ctx context.Context) error) *SystemAPI {
	return &SystemAPI{pingDB: pingDB}
}

// Register 注册路由。
func (a *SystemAPI) Register(r *gin.Engine) {
	r.GET("/healthz", a.health)
	r.GET("/readyz", a.ready)
}

// GET /healthz 存活检查
func (a *SystemAPI) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

// GET /readyz 就绪检查（含数据库连通性）
func (a *SystemAPI) ready(c *gin.Context) {
	if a.pingDB != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := a.pingDB(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "mongo": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "mongo": "up"})
}
