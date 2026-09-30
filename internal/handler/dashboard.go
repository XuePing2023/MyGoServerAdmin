// Package handler HTTP 接口层：仪表盘。
package handler

import (
	"github.com/gin-gonic/gin"

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
