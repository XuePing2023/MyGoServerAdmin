package api

import (
	"github.com/gin-gonic/gin"

	"serveradmin/internal/pkg/response"
	"serveradmin/internal/service"
)

// AuthAPI 认证接口。
type AuthAPI struct {
	svc    *service.AuthService
	online *service.OnlineService
}

// NewAuthAPI 创建认证接口。
func NewAuthAPI(svc *service.AuthService, online *service.OnlineService) *AuthAPI {
	return &AuthAPI{svc: svc, online: online}
}

// Register 注册路由（public 为无需登录的分组）。
func (a *AuthAPI) Register(public, authed *gin.RouterGroup) {
	public.GET("/auth/captcha", a.captcha)
	public.POST("/auth/login", a.login)
	public.POST("/auth/refresh", a.refresh)

	authed.POST("/auth/logout", a.logout)
	authed.GET("/auth/profile", a.profile)
	authed.PUT("/auth/profile", a.updateProfile)
	authed.PUT("/auth/profile/password", a.changePassword)
}

// captcha godoc
// GET /api/v1/auth/captcha 获取登录验证码
func (a *AuthAPI) captcha(c *gin.Context) {
	id, image, err := a.svc.GenerateCaptcha()
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, gin.H{"captchaId": id, "image": image})
}

// login godoc
// POST /api/v1/auth/login 登录
func (a *AuthAPI) login(c *gin.Context) {
	var in service.LoginInput
	if !bindJSON(c, &in) {
		return
	}
	res, err := a.svc.Login(c.Request.Context(), &in, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, res)
}

// refresh godoc
// POST /api/v1/auth/refresh 刷新令牌
func (a *AuthAPI) refresh(c *gin.Context) {
	var in struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if !bindJSON(c, &in) {
		return
	}
	res, err := a.svc.Refresh(c.Request.Context(), in.RefreshToken)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, res)
}

// logout godoc
// POST /api/v1/auth/logout 登出
func (a *AuthAPI) logout(c *gin.Context) {
	id, ok := requireIdentity(c)
	if !ok {
		return
	}
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = c.ShouldBindJSON(&in) // body 可选
	if err := a.svc.Logout(c.Request.Context(), id.Claims, in.RefreshToken); err != nil {
		response.Handle(c, err)
		return
	}
	a.online.Remove(id.JTI)
	response.OK(c, nil)
}

// profile godoc
// GET /api/v1/auth/profile 当前用户信息与权限
func (a *AuthAPI) profile(c *gin.Context) {
	id, ok := requireIdentity(c)
	if !ok {
		return
	}
	res, err := a.svc.Profile(c.Request.Context(), id.UserID)
	if err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, res)
}

// updateProfile godoc
// PUT /api/v1/auth/profile 修改个人资料
func (a *AuthAPI) updateProfile(c *gin.Context) {
	id, ok := requireIdentity(c)
	if !ok {
		return
	}
	var in service.ProfileInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.UpdateProfile(c.Request.Context(), id.UserID, &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}

// changePassword godoc
// PUT /api/v1/auth/profile/password 修改密码
func (a *AuthAPI) changePassword(c *gin.Context) {
	id, ok := requireIdentity(c)
	if !ok {
		return
	}
	var in service.PasswordInput
	if !bindJSON(c, &in) {
		return
	}
	if err := a.svc.ChangePassword(c.Request.Context(), id.UserID, &in); err != nil {
		response.Handle(c, err)
		return
	}
	response.OK(c, nil)
}
