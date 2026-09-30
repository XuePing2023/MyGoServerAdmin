package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
	"serveradmin/internal/pkg/jwtx"

	"github.com/mojocn/base64Captcha"
)

// AuthService 认证服务：验证码、登录、刷新、登出、个人中心。
type AuthService struct {
	db      *mongo.Database
	cfg     *config.Config
	jwt     *jwtx.Manager
	token   *TokenService
	menu    *MenuService
	perm    *PermService
	captcha *base64Captcha.Captcha // 为 nil 表示未启用验证码
}

// NewAuthService 创建认证服务。
func NewAuthService(db *mongo.Database, cfg *config.Config, jwtMgr *jwtx.Manager, token *TokenService, menu *MenuService, perm *PermService) *AuthService {
	s := &AuthService{db: db, cfg: cfg, jwt: jwtMgr, token: token, menu: menu, perm: perm}
	if cfg.Captcha.Enabled {
		store := base64Captcha.NewMemoryStore(1024, time.Duration(cfg.Captcha.ExpireSeconds)*time.Second)
		driver := base64Captcha.NewDriverDigit(80, 240, 4, 0.7, 80)
		s.captcha = base64Captcha.NewCaptcha(driver, store)
	}
	return s
}

// LoginInput 登录请求。
type LoginInput struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

// UserVO 对外暴露的用户信息。
type UserVO struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar,omitempty"`
	Roles    []string `json:"roles"`
}

// LoginResult 登录/刷新结果。
type LoginResult struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	ExpiresAt    int64   `json:"expiresAt"`
	User         *UserVO `json:"user"`
}

// ProfileResult 个人信息。
type ProfileResult struct {
	User  *model.User   `json:"user"`
	Roles []string      `json:"roles"`
	Perms []string      `json:"perms"`
	Menus []*model.Menu `json:"menus"`
}

// GenerateCaptcha 生成登录验证码。
func (s *AuthService) GenerateCaptcha() (id, image string, err error) {
	if s.captcha == nil {
		return "", "", errs.BadRequest("验证码未启用")
	}
	id, b64, _, err := s.captcha.Generate()
	return id, b64, err
}

// Login 登录。
func (s *AuthService) Login(ctx context.Context, in *LoginInput, ip, ua string) (*LoginResult, error) {
	if s.captcha != nil {
		if in.CaptchaID == "" || in.CaptchaCode == "" || !s.captcha.Verify(in.CaptchaID, in.CaptchaCode, true) {
			return nil, errs.BadRequest("验证码错误或已过期")
		}
	}

	var u model.User
	err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"username": in.Username}, &u)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		s.recordLogin(ctx, "", in.Username, ip, ua, model.LoginStatusFailed, "用户名或密码错误")
		return nil, errs.BadRequest("用户名或密码错误")
	}
	if u.Status != model.StatusEnabled {
		s.recordLogin(ctx, u.ID, u.Username, ip, ua, model.LoginStatusFailed, "账号已被禁用")
		return nil, errs.Forbidden("账号已被禁用，请联系管理员")
	}

	access, refresh, exp, err := s.jwt.Generate(&jwtx.TokenUser{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname, Roles: u.Roles,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, _ = s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": u.ID},
		bson.M{"$set": bson.M{"lastLogin": now, "lastIp": ip}})
	s.recordLogin(ctx, u.ID, u.Username, ip, ua, model.LoginStatusSuccess, "登录成功")

	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp.Unix(),
		User:         toUserVO(&u),
	}, nil
}

// Refresh 使用刷新令牌换取新令牌对。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*LoginResult, error) {
	claims, err := s.jwt.Parse(refreshToken, jwtx.TypeRefresh)
	if err != nil {
		return nil, errs.Unauthorized("登录状态已失效，请重新登录")
	}
	if s.token.Banned(claims.ID) {
		return nil, errs.Unauthorized("登录状态已失效，请重新登录")
	}
	var u model.User
	if err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"_id": claims.UserID}, &u); err != nil {
		return nil, errs.Unauthorized("用户不存在或已被删除")
	}
	if u.Status != model.StatusEnabled {
		return nil, errs.Forbidden("账号已被禁用，请联系管理员")
	}

	access, refresh, exp, err := s.jwt.Generate(&jwtx.TokenUser{
		ID: u.ID, Username: u.Username, Nickname: u.Nickname, Roles: u.Roles,
	})
	if err != nil {
		return nil, err
	}
	// 旧刷新令牌作废
	_ = s.token.Add(ctx, claims.ID, claims.ExpiresAt.Time)

	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp.Unix(),
		User:         toUserVO(&u),
	}, nil
}

// Logout 登出：访问令牌与刷新令牌均加入黑名单。
func (s *AuthService) Logout(ctx context.Context, claims *jwtx.Claims, refreshToken string) error {
	if claims != nil && claims.ExpiresAt != nil {
		_ = s.token.Add(ctx, claims.ID, claims.ExpiresAt.Time)
	}
	if refreshToken != "" {
		if rc, err := s.jwt.Parse(refreshToken, jwtx.TypeRefresh); err == nil && rc.ExpiresAt != nil {
			_ = s.token.Add(ctx, rc.ID, rc.ExpiresAt.Time)
		}
	}
	return nil
}

// Profile 个人信息 + 权限 + 可见菜单。
func (s *AuthService) Profile(ctx context.Context, userID string) (*ProfileResult, error) {
	var u model.User
	if err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"_id": userID}, &u); err != nil {
		return nil, errs.NotFound("用户不存在")
	}
	permSet, _, err := s.perm.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	menus, err := s.menu.GetForRoles(ctx, u.Roles)
	if err != nil {
		return nil, err
	}
	return &ProfileResult{User: &u, Roles: u.Roles, Perms: perms, Menus: menus}, nil
}

// ProfileInput 修改个人资料。
type ProfileInput struct {
	Nickname string `json:"nickname" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone"`
	Gender   int    `json:"gender"`
	Avatar   string `json:"avatar"`
}

// UpdateProfile 修改个人资料。
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, in *ProfileInput) error {
	_, err := s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": userID},
		bson.M{"$set": bson.M{
			"nickname": in.Nickname, "email": in.Email,
			"phone": in.Phone, "gender": in.Gender, "avatar": in.Avatar,
			"updatedAt": time.Now(),
		}})
	return err
}

// PasswordInput 修改密码。
type PasswordInput struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=64"`
}

// ChangePassword 修改自己的密码。
func (s *AuthService) ChangePassword(ctx context.Context, userID string, in *PasswordInput) error {
	var u model.User
	if err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"_id": userID}, &u); err != nil {
		return errs.NotFound("用户不存在")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.OldPassword)) != nil {
		return errs.BadRequest("原密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": userID},
		bson.M{"$set": bson.M{"password": string(hash), "updatedAt": time.Now()}})
	return err
}

// recordLogin 写登录日志（使用独立 context，不受请求生命周期影响）。
func (s *AuthService) recordLogin(ctx context.Context, userID, username, ip, ua string, status int, msg string) {
	log := &model.LoginLog{
		UserID: userID, Username: username, IP: ip, UserAgent: ua,
		Status: status, Msg: msg, LoginAt: time.Now(),
	}
	log.PrepareCreate()
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.db.Collection(model.ColLoginLog).InsertOne(bg, log)
	_ = ctx
}

func toUserVO(u *model.User) *UserVO {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return &UserVO{ID: u.ID, Username: u.Username, Nickname: u.Nickname, Avatar: u.Avatar, Roles: roles}
}
