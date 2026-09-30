// Package handler HTTP 接口层：系统监控。
package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// DashboardAPI 仪表盘接口。

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
