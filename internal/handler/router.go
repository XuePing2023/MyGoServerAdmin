package handler

import (
	"context"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"serveradmin/internal/cache"
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
	authed.Use(responseCache(reg))

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

// responseCache 显式 API 缓存配置：路由模式 -> 缓存策略。
// 只缓存无权限门槛的 GET 接口（组级中间件先于 RequirePerm 执行，
// PUBLIC 缓存会把数据泄露给无权限用户，故带权限校验的接口一律不缓存）；
// 未列出的路由视为 NONE，完全不缓存（用户列表/日志/监控等实时数据）。
func responseCache(reg *service.Registry) gin.HandlerFunc {
	routes := map[string]cache.Rule{
		// USER：每个用户不同，serveradmin:v1:user:{uid}:profile
		"GET /api/v1/auth/profile": {Policy: cache.PolicyUser, Resource: "profile"},
		// USER：工作台统计，按用户区分，随 userTTL 自动过期
		"GET /api/v1/dashboard/stats": {Policy: cache.PolicyUser, Resource: "dashboard"},
	}
	return middleware.ResponseCache(reg.Cache, reg.CacheHelper.TTL, routes)
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
