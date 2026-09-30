package api

import (
	"context"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/config"
	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/jwtx"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
	"serveradmin/internal/webui"
)

// NewRouter 构建全部路由。
func NewRouter(cfg *config.Config, reg *service.Registry, jwtMgr *jwtx.Manager, mongoClient *mongo.Client) *gin.Engine {
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery(), middleware.Cors(cfg), middleware.AccessLog())

	// 健康检查（公开，含数据库连通性）
	pingDB := func(ctx context.Context) error {
		pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return mongoClient.Ping(pctx, nil)
	}
	NewSystemAPI(pingDB).Register(r)

	// 上传文件静态服务
	r.Static("/uploads", cfg.App.UploadDir)

	// API v1
	apiGroup := r.Group("/api/v1")
	public := apiGroup.Group("")
	authed := apiGroup.Group("")
	authed.Use(middleware.Auth(jwtMgr, reg.Perm, reg.Online, reg.Token))
	authed.Use(middleware.OpLog(cfg, reg.Log))

	NewAuthAPI(reg.Auth, reg.Online).Register(public, authed)
	NewConfigAPI(reg.SysConfig).Register(public, authed)
	NewUserAPI(reg.User).Register(authed)
	NewRoleAPI(reg.Role).Register(authed)
	NewMenuAPI(reg.Menu).Register(authed)
	NewDeptAPI(reg.Dept).Register(authed)
	NewDictAPI(reg.Dict).Register(authed)
	NewNoticeAPI(reg.Notice).Register(authed)
	NewLogAPI(reg.Log).Register(authed)
	NewOnlineAPI(reg.Online, reg.Token, jwtMgr).Register(authed)
	NewFileAPI(reg.File).Register(authed)
	NewJobAPI(reg.Job).Register(authed)
	NewDashboardAPI(reg.Dashboard, reg.Online).Register(authed)
	NewMonitorAPI(reg.Monitor).Register(authed)

	// 前端静态资源（SPA 回退）
	setupWebUI(r)

	return r
}

func setupWebUI(r *gin.Engine) {
	fsys := webui.FS()
	fileServer := http.FileServer(http.FS(fsys))

	// 直接读取内容返回，避免 http.ServeFile 对 index.html 的 301 规范化重定向
	indexHTML, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		panic("读取内嵌前端 index.html 失败: " + err.Error())
	}
	htmlType := "text/html; charset=utf-8"

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, htmlType, indexHTML)
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/uploads/") {
			response.Fail(c, http.StatusNotFound, "接口不存在")
			return
		}
		// 静态文件存在则直接返回，否则回退到 index.html
		if f, err := fsys.Open(strings.TrimPrefix(p, "/")); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Data(http.StatusOK, htmlType, indexHTML)
	})
}
