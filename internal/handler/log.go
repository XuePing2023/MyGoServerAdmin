package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/jwtx"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// LogAPI 日志接口（操作日志 + 登录日志）。
type LogAPI struct{ svc *service.LogService }

// NewLogAPI 创建日志接口。
func NewLogAPI(svc *service.LogService) *LogAPI { return &LogAPI{svc: svc} }

// Register 注册路由。
func (a *LogAPI) Register(authed *gin.RouterGroup) {
	op := authed.Group("/oplogs")
	{
		op.GET("", middleware.RequirePerm("sys:oplog:list"), a.opList)
		op.DELETE("", middleware.RequirePerm("sys:oplog:delete"), a.opDelete)
	}
	login := authed.Group("/loginlogs")
	{
		login.GET("", middleware.RequirePerm("sys:loginlog:list"), a.loginList)
		login.DELETE("", middleware.RequirePerm("sys:loginlog:delete"), a.loginDelete)
	}
}

// GET /api/v1/oplogs 操作日志列表
func (a *LogAPI) opList(c *gin.Context) {
	page, size := pageParams(c)
	start, end := dateRange(c)
	list, total, err := a.svc.OpList(c.Request.Context(), &service.OpQuery{
		Username: c.Query("username"),
		Module:   c.Query("module"),
		Start:    start,
		End:      end,
		Page:     page,
		Size:     size,
	})
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// DELETE /api/v1/oplogs 删除操作日志（body.ids 可选；为空清空全部）
func (a *LogAPI) opDelete(c *gin.Context) {
	var in idsBody
	_ = c.ShouldBindJSON(&in) // ids 可选
	n, err := a.svc.OpDelete(c.Request.Context(), in.IDs)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": n})
}

// GET /api/v1/loginlogs 登录日志列表
func (a *LogAPI) loginList(c *gin.Context) {
	page, size := pageParams(c)
	start, end := dateRange(c)
	list, total, err := a.svc.LoginList(c.Request.Context(), &service.LoginQuery{
		Username: c.Query("username"),
		Status:   strconv0(c.Query("status")),
		Start:    start,
		End:      end,
		Page:     page,
		Size:     size,
	})
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// DELETE /api/v1/loginlogs 删除登录日志（body.ids 可选；为空清空全部）
func (a *LogAPI) loginDelete(c *gin.Context) {
	var in idsBody
	_ = c.ShouldBindJSON(&in)
	n, err := a.svc.LoginDelete(c.Request.Context(), in.IDs)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": n})
}

// OnlineAPI 在线用户接口。
type OnlineAPI struct {
	online *service.OnlineService
	token  *service.TokenService
	jwtMgr *jwtx.Manager
}

// NewOnlineAPI 创建在线用户接口。
func NewOnlineAPI(online *service.OnlineService, token *service.TokenService, jwtMgr *jwtx.Manager) *OnlineAPI {
	return &OnlineAPI{online: online, token: token, jwtMgr: jwtMgr}
}

// Register 注册路由。
func (a *OnlineAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/online")
	g.GET("", middleware.RequirePerm("sys:online:list"), a.list)
	g.DELETE("/:jti", middleware.RequirePerm("sys:online:kick"), a.kick)
}

// GET /api/v1/online 在线用户列表
func (a *OnlineAPI) list(c *gin.Context) {
	response.OK(c, a.online.List())
}

// DELETE /api/v1/online/:jti 强制下线
func (a *OnlineAPI) kick(c *gin.Context) {
	jti := c.Param("jti")
	if !a.online.Remove(jti) {
		response.Handle(c, errs.NotFound("会话不存在或已下线"))
		return
	}
	// 黑名单保持到该访问令牌自然过期
	exp := time.Now().Add(a.jwtMgr.AccessExpire())
	_ = a.token.Add(c.Request.Context(), jti, exp)
	response.OK(c, nil)
}
