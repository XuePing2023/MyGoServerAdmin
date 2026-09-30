// Package handler HTTP 接口层：健康检查。
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DashboardAPI 仪表盘接口。

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
