package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"serveradmin/internal/cache"
	"serveradmin/internal/config"
	"serveradmin/internal/model"
	"serveradmin/internal/pkg/errs"
)

// 内置超级管理员账号，受保护。
const BuiltInAdminUsername = "admin"

// UserService 用户管理。
type UserService struct {
	db    *mongo.Database
	cfg   *config.Config
	cache *cache.Helper
}

// NewUserService 创建用户服务。
func NewUserService(db *mongo.Database, cfg *config.Config, helper *cache.Helper) *UserService {
	return &UserService{db: db, cfg: cfg, cache: helper}
}

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_\-@.]{2,32}$`)

// UserItem 列表项（附加部门名/角色名）。
type UserItem struct {
	model.User
	DeptName  string   `json:"deptName"`
	RoleNames []string `json:"roleNames"`
}

// UserQuery 列表查询条件。
type UserQuery struct {
	Username string
	Nickname string
	Phone    string
	Status   int
	DeptID   string
	Page     int
	Size     int
}

// UserInput 创建/更新用户请求。
type UserInput struct {
	Username string   `json:"username" binding:"omitempty,min=2,max=32"`
	Nickname string   `json:"nickname" binding:"required,max=32"`
	Password string   `json:"password" binding:"omitempty,min=6,max=64"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Phone    string   `json:"phone"`
	Gender   int      `json:"gender"`
	DeptID   string   `json:"deptId"`
	Roles    []string `json:"roles"`
	Status   int      `json:"status" binding:"omitempty,oneof=1 2"`
	Remark   string   `json:"remark"`
}

// List 分页查询用户。
func (s *UserService) List(ctx context.Context, q *UserQuery) ([]*UserItem, int64, error) {
	page, size := normalizePage(q.Page, q.Size)
	filter := bson.M{}
	if q.Username != "" {
		filter["username"] = likeFilter(q.Username)
	}
	if q.Nickname != "" {
		filter["nickname"] = likeFilter(q.Nickname)
	}
	if q.Phone != "" {
		filter["phone"] = likeFilter(q.Phone)
	}
	if q.Status == model.StatusEnabled || q.Status == model.StatusDisabled {
		filter["status"] = q.Status
	}
	if q.DeptID != "" {
		// 包含子部门下的用户
		deptSvc := &DepartmentService{db: s.db}
		ids, err := deptSvc.selfAndDescendantIDs(ctx, q.DeptID)
		if err != nil {
			return nil, 0, err
		}
		filter["deptId"] = bson.M{"$in": ids}
	}

	users, total, err := pageFind[model.User](ctx, s.db.Collection(model.ColUser), filter, page, size, nil)
	if err != nil {
		return nil, 0, err
	}
	return s.decorate(ctx, users), total, nil
}

// decorate 附加部门名称与角色名称。
func (s *UserService) decorate(ctx context.Context, users []*model.User) []*UserItem {
	deptNames := map[string]string{}
	if cursor, err := s.db.Collection(model.ColDepartment).Find(ctx, bson.M{}); err == nil {
		var depts []*model.Department
		if cursor.All(ctx, &depts) == nil {
			for _, d := range depts {
				deptNames[d.ID] = d.Name
			}
		}
		cursor.Close(ctx)
	}
	roleNames := map[string]string{}
	if cursor, err := s.db.Collection(model.ColRole).Find(ctx, bson.M{}); err == nil {
		var roles []*model.Role
		if cursor.All(ctx, &roles) == nil {
			for _, r := range roles {
				roleNames[r.Code] = r.Name
			}
		}
		cursor.Close(ctx)
	}

	items := make([]*UserItem, 0, len(users))
	for _, u := range users {
		names := make([]string, 0, len(u.Roles))
		for _, c := range u.Roles {
			if n, ok := roleNames[c]; ok {
				names = append(names, n)
			}
		}
		items = append(items, &UserItem{
			User:      *u,
			DeptName:  deptNames[u.DeptID],
			RoleNames: names,
		})
	}
	return items
}

// Get 用户详情。
func (s *UserService) Get(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	if err := findOne(ctx, s.db.Collection(model.ColUser), bson.M{"_id": id}, &u); err != nil {
		return nil, errs.NotFound("用户不存在")
	}
	return &u, nil
}

// Create 创建用户。
func (s *UserService) Create(ctx context.Context, in *UserInput, creator string) (*model.User, error) {
	if in.Username == "" {
		return nil, errs.BadRequest("用户名不能为空")
	}
	if !usernameRe.MatchString(in.Username) {
		return nil, errs.BadRequest("用户名只能包含字母、数字、_-@.，长度 2-32")
	}
	n, err := count(ctx, s.db.Collection(model.ColUser), bson.M{"username": in.Username})
	if err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, errs.BadRequest("用户名已存在")
	}
	if err := s.validateRoles(ctx, in.Roles); err != nil {
		return nil, err
	}

	pwd := in.Password
	if pwd == "" {
		pwd = "admin123" // 默认初始密码，可在系统参数中提示修改
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	status := in.Status
	if status == 0 {
		status = model.StatusEnabled
	}
	u := &model.User{
		Username: in.Username, Nickname: in.Nickname, Password: string(hash),
		Email: in.Email, Phone: in.Phone, Gender: in.Gender,
		DeptID: in.DeptID, Roles: in.Roles, Status: status,
		Remark: in.Remark,
	}
	u.PrepareCreate()
	_, err = s.db.Collection(model.ColUser).InsertOne(ctx, u)
	if err != nil {
		return nil, err
	}
	_ = creator
	return u, nil
}

// Update 更新用户。
func (s *UserService) Update(ctx context.Context, id string, in *UserInput) error {
	u, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.validateRoles(ctx, in.Roles); err != nil {
		return err
	}
	// 内置管理员必须保留 super 角色
	if u.Username == BuiltInAdminUsername && !contains(in.Roles, model.SuperRoleCode) {
		return errs.Forbidden("内置管理员必须保留超级管理员角色")
	}
	update := bson.M{
		"nickname": in.Nickname, "email": in.Email, "phone": in.Phone,
		"gender": in.Gender, "deptId": in.DeptID, "roles": in.Roles,
		"remark": in.Remark, "updatedAt": time.Now(),
	}
	if in.Status != 0 && !(u.Username == BuiltInAdminUsername && in.Status == model.StatusDisabled) {
		update["status"] = in.Status
	}
	_, err = s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err == nil {
		// 写操作成功后失效该用户的缓存（个人资料/工作台）
		s.cache.Invalidate(ctx, cache.UserPrefix(id))
	}
	return err
}

// Delete 批量删除用户。
func (s *UserService) Delete(ctx context.Context, ids []string, operatorID string) error {
	if len(ids) == 0 {
		return errs.BadRequest("请选择要删除的用户")
	}
	for _, id := range ids {
		if id == operatorID {
			return errs.BadRequest("不能删除当前登录账号")
		}
	}
	n, err := s.db.Collection(model.ColUser).CountDocuments(ctx,
		bson.M{"_id": bson.M{"$in": ids}, "username": BuiltInAdminUsername})
	if err != nil {
		return err
	}
	if n > 0 {
		return errs.Forbidden("内置管理员不允许删除")
	}
	_, err = s.db.Collection(model.ColUser).DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err == nil {
		prefixes := make([]string, 0, len(ids))
		for _, id := range ids {
			prefixes = append(prefixes, cache.UserPrefix(id))
		}
		s.cache.Invalidate(ctx, prefixes...)
	}
	return err
}

// SetStatus 启用/禁用。
func (s *UserService) SetStatus(ctx context.Context, id string, status int) error {
	u, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if u.Username == BuiltInAdminUsername && status == model.StatusDisabled {
		return errs.Forbidden("内置管理员不允许禁用")
	}
	if status != model.StatusEnabled && status != model.StatusDisabled {
		return errs.BadRequest("状态值无效")
	}
	_, err = s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": status, "updatedAt": time.Now()}})
	if err == nil {
		s.cache.Invalidate(ctx, cache.UserPrefix(id))
	}
	return err
}

// ResetPassword 管理员重置密码。
func (s *UserService) ResetPassword(ctx context.Context, id string, password string) error {
	if len(password) < 6 {
		return errs.BadRequest("密码长度至少 6 位")
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Collection(model.ColUser).UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"password": string(hash), "updatedAt": time.Now()}})
	if err == nil {
		s.cache.Invalidate(ctx, cache.UserPrefix(id))
	}
	return err
}

func (s *UserService) validateRoles(ctx context.Context, codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	n, err := count(ctx, s.db.Collection(model.ColRole), bson.M{"code": bson.M{"$in": codes}})
	if err != nil {
		return err
	}
	if int(n) != len(unique(codes)) {
		return errs.BadRequest("存在无效的角色")
	}
	return nil
}

// ---------- 通用小工具 ----------

func likeFilter(s string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(strings.TrimSpace(s)), "$options": "i"}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func unique(list []string) []string {
	seen := make(map[string]struct{}, len(list))
	out := make([]string, 0, len(list))
	for _, v := range list {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
