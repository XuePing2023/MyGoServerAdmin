package api

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/middleware"
	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// UserAPI 用户管理接口。
type UserAPI struct {
	svc *service.UserService
}

// NewUserAPI 创建用户接口。
func NewUserAPI(svc *service.UserService) *UserAPI { return &UserAPI{svc: svc} }

// Register 注册路由。
func (a *UserAPI) Register(authed *gin.RouterGroup) {
	g := authed.Group("/users")
	g.GET("", middleware.RequirePerm("sys:user:list"), a.list)
	g.GET("/:id", middleware.RequirePerm("sys:user:list"), a.get)
	g.POST("", middleware.RequirePerm("sys:user:create"), a.create)
	g.PUT("/:id", middleware.RequirePerm("sys:user:update"), a.update)
	g.DELETE("", middleware.RequirePerm("sys:user:delete"), a.delete)
	g.PUT("/:id/status", middleware.RequirePerm("sys:user:status"), a.setStatus)
	g.PUT("/:id/password", middleware.RequirePerm("sys:user:resetpwd"), a.resetPassword)
}

// GET /api/v1/users 用户列表
func (a *UserAPI) list(c *gin.Context) {
	page, size := pageParams(c)
	status := strconv0(c.Query("status"))
	list, total, err := a.svc.List(c.Request.Context(), &service.UserQuery{
		Username: c.Query("username"),
		Nickname: c.Query("nickname"),
		Phone:    c.Query("phone"),
		Status:   status,
		DeptID:   c.Query("deptId"),
		Page:     page,
		Size:     size,
	})
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.Page(c, list, total)
}

// GET /api/v1/users/:id 用户详情
func (a *UserAPI) get(c *gin.Context) {
	u, err := a.svc.Get(c.Request.Context(), pathID(c))
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, u)
}

// POST /api/v1/users 新建用户
func (a *UserAPI) create(c *gin.Context) {
	var in service.UserInput
	if !bindJSON(c, &in) {
		return
	}
	id := middleware.FromContext(c)
	u, err := a.svc.Create(c.Request.Context(), &in, id.Username)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, u)
}

// PUT /api/v1/users/:id 更新用户
func (a *UserAPI) update(c *gin.Context) {
	var in service.UserInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.Update(c.Request.Context(), pathID(c), &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/users 批量删除
func (a *UserAPI) delete(c *gin.Context) {
	var in idsBody
	if !bindJSON(c, &in) {
		return
	}
	id := middleware.FromContext(c)
	if err := a.svc.Delete(c.Request.Context(), in.IDs, id.UserID); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// PUT /api/v1/users/:id/status 启用/禁用
func (a *UserAPI) setStatus(c *gin.Context) {
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

// PUT /api/v1/users/:id/password 重置密码
func (a *UserAPI) resetPassword(c *gin.Context) {
	var in struct {
		Password string `json:"password" binding:"required,min=6,max=64"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.ResetPassword(c.Request.Context(), pathID(c), in.Password); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}
