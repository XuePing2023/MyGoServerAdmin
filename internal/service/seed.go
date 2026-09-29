package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"serveradmin/internal/model"
	"serveradmin/internal/pkg/logger"
)

// Seed 首次启动时初始化演示数据（users 集合为空才执行）。
func Seed(ctx context.Context, db *mongo.Database) error {
	users := db.Collection(model.ColUser)
	n, err := count(ctx, users, bson.M{})
	if err != nil {
		return err
	}
	if n > 0 {
		logger.L.Info("数据库已有数据，跳过种子数据初始化")
		return nil
	}
	logger.L.Info("检测到首次启动，正在初始化种子数据 ...")

	ids := map[string]string{}

	// ---------- 部门 ----------
	var insertDept func(name, parentID string) (*model.Department, error)
	insertDept = func(name, parentID string) (*model.Department, error) {
		d := &model.Department{Name: name, ParentID: parentID, Status: model.StatusEnabled}
		d.PrepareCreate()
		if _, err := db.Collection(model.ColDepartment).InsertOne(ctx, d); err != nil {
			return nil, err
		}
		return d, nil
	}
	root, err := insertDept("ServerAdmin科技集团", "")
	if err != nil {
		return err
	}
	childDepts := map[string]string{}
	for _, name := range []string{"技术部", "市场部", "财务部", "人事部"} {
		d, err := insertDept(name, root.ID)
		if err != nil {
			return err
		}
		childDepts[name] = d.ID
	}

	// ---------- 菜单 ----------
	type menuSeed struct {
		key      string
		name     string
		path     string
		comp     string
		icon     string
		perm     string
		sort     int
		typ      int
		visible  bool
		children []menuSeed
	}
	dir, page, btn := model.MenuTypeDir, model.MenuTypePage, model.MenuTypeButton

	seeds := []menuSeed{
		{key: "m:dashboard", name: "首页", path: "/dashboard", comp: "dashboard", icon: "🏠", sort: 1, typ: page, visible: true},
		{key: "m:system", name: "系统管理", path: "/system", icon: "⚙️", sort: 2, typ: dir, visible: true, children: []menuSeed{
			{key: "m:users", name: "用户管理", path: "/system/users", comp: "users", icon: "👤", perm: "sys:user:list", sort: 1, typ: page, visible: true, children: []menuSeed{
				{key: "b:user:create", name: "新增用户", perm: "sys:user:create", typ: btn, sort: 1, visible: true},
				{key: "b:user:update", name: "修改用户", perm: "sys:user:update", typ: btn, sort: 2, visible: true},
				{key: "b:user:delete", name: "删除用户", perm: "sys:user:delete", typ: btn, sort: 3, visible: true},
				{key: "b:user:resetpwd", name: "重置密码", perm: "sys:user:resetpwd", typ: btn, sort: 4, visible: true},
				{key: "b:user:status", name: "启停用户", perm: "sys:user:status", typ: btn, sort: 5, visible: true},
			}},
			{key: "m:roles", name: "角色管理", path: "/system/roles", comp: "roles", icon: "🎭", perm: "sys:role:list", sort: 2, typ: page, visible: true, children: []menuSeed{
				{key: "b:role:create", name: "新增角色", perm: "sys:role:create", typ: btn, sort: 1, visible: true},
				{key: "b:role:update", name: "修改角色", perm: "sys:role:update", typ: btn, sort: 2, visible: true},
				{key: "b:role:delete", name: "删除角色", perm: "sys:role:delete", typ: btn, sort: 3, visible: true},
				{key: "b:role:assign", name: "分配权限", perm: "sys:role:assign", typ: btn, sort: 4, visible: true},
			}},
			{key: "m:menus", name: "菜单管理", path: "/system/menus", comp: "menus", icon: "🧭", perm: "sys:menu:list", sort: 3, typ: page, visible: true, children: []menuSeed{
				{key: "b:menu:create", name: "新增菜单", perm: "sys:menu:create", typ: btn, sort: 1, visible: true},
				{key: "b:menu:update", name: "修改菜单", perm: "sys:menu:update", typ: btn, sort: 2, visible: true},
				{key: "b:menu:delete", name: "删除菜单", perm: "sys:menu:delete", typ: btn, sort: 3, visible: true},
			}},
			{key: "m:depts", name: "部门管理", path: "/system/depts", comp: "depts", icon: "🏢", perm: "sys:dept:list", sort: 4, typ: page, visible: true, children: []menuSeed{
				{key: "b:dept:create", name: "新增部门", perm: "sys:dept:create", typ: btn, sort: 1, visible: true},
				{key: "b:dept:update", name: "修改部门", perm: "sys:dept:update", typ: btn, sort: 2, visible: true},
				{key: "b:dept:delete", name: "删除部门", perm: "sys:dept:delete", typ: btn, sort: 3, visible: true},
			}},
			{key: "m:dicts", name: "字典管理", path: "/system/dicts", comp: "dicts", icon: "📖", perm: "sys:dict:list", sort: 5, typ: page, visible: true, children: []menuSeed{
				{key: "b:dict:create", name: "新增字典", perm: "sys:dict:create", typ: btn, sort: 1, visible: true},
				{key: "b:dict:update", name: "修改字典", perm: "sys:dict:update", typ: btn, sort: 2, visible: true},
				{key: "b:dict:delete", name: "删除字典", perm: "sys:dict:delete", typ: btn, sort: 3, visible: true},
			}},
			{key: "m:configs", name: "参数配置", path: "/system/configs", comp: "configs", icon: "🔧", perm: "sys:config:list", sort: 6, typ: page, visible: true, children: []menuSeed{
				{key: "b:config:create", name: "新增参数", perm: "sys:config:create", typ: btn, sort: 1, visible: true},
				{key: "b:config:update", name: "修改参数", perm: "sys:config:update", typ: btn, sort: 2, visible: true},
				{key: "b:config:delete", name: "删除参数", perm: "sys:config:delete", typ: btn, sort: 3, visible: true},
			}},
			{key: "m:notices", name: "通知公告", path: "/system/notices", comp: "notices", icon: "📢", perm: "sys:notice:list", sort: 7, typ: page, visible: true, children: []menuSeed{
				{key: "b:notice:create", name: "新增公告", perm: "sys:notice:create", typ: btn, sort: 1, visible: true},
				{key: "b:notice:update", name: "修改公告", perm: "sys:notice:update", typ: btn, sort: 2, visible: true},
				{key: "b:notice:delete", name: "删除公告", perm: "sys:notice:delete", typ: btn, sort: 3, visible: true},
			}},
			{key: "m:files", name: "文件管理", path: "/system/files", comp: "files", icon: "📁", perm: "sys:file:list", sort: 8, typ: page, visible: true, children: []menuSeed{
				{key: "b:file:upload", name: "上传文件", perm: "sys:file:upload", typ: btn, sort: 1, visible: true},
				{key: "b:file:delete", name: "删除文件", perm: "sys:file:delete", typ: btn, sort: 2, visible: true},
			}},
			{key: "m:oplogs", name: "操作日志", path: "/system/oplogs", comp: "oplogs", icon: "📝", perm: "sys:oplog:list", sort: 9, typ: page, visible: true, children: []menuSeed{
				{key: "b:oplog:delete", name: "删除日志", perm: "sys:oplog:delete", typ: btn, sort: 1, visible: true},
			}},
			{key: "m:loginlogs", name: "登录日志", path: "/system/loginlogs", comp: "loginlogs", icon: "🔐", perm: "sys:loginlog:list", sort: 10, typ: page, visible: true, children: []menuSeed{
				{key: "b:loginlog:delete", name: "删除日志", perm: "sys:loginlog:delete", typ: btn, sort: 1, visible: true},
			}},
			{key: "m:online", name: "在线用户", path: "/system/online", comp: "online", icon: "🟢", perm: "sys:online:list", sort: 11, typ: page, visible: true, children: []menuSeed{
				{key: "b:online:kick", name: "强制下线", perm: "sys:online:kick", typ: btn, sort: 1, visible: true},
			}},
		}},
		{key: "m:tools", name: "系统工具", path: "/tools", icon: "🧰", sort: 3, typ: dir, visible: true, children: []menuSeed{
			{key: "m:jobs", name: "定时任务", path: "/tools/jobs", comp: "jobs", icon: "⏰", perm: "sys:job:list", sort: 1, typ: page, visible: true, children: []menuSeed{
				{key: "b:job:create", name: "新增任务", perm: "sys:job:create", typ: btn, sort: 1, visible: true},
				{key: "b:job:update", name: "修改任务", perm: "sys:job:update", typ: btn, sort: 2, visible: true},
				{key: "b:job:delete", name: "删除任务", perm: "sys:job:delete", typ: btn, sort: 3, visible: true},
				{key: "b:job:run", name: "执行/启停", perm: "sys:job:run", typ: btn, sort: 4, visible: true},
			}},
			{key: "m:monitor", name: "系统监控", path: "/tools/monitor", comp: "monitor", icon: "🖥️", perm: "sys:monitor", sort: 2, typ: page, visible: true},
		}},
	}

	var insertMenu func(items []menuSeed, parentID string) error
	insertMenu = func(items []menuSeed, parentID string) error {
		for _, ms := range items {
			m := &model.Menu{
				ParentID: parentID, Name: ms.name, Path: ms.path, Component: ms.comp,
				Icon: ms.icon, Perm: ms.perm, Sort: ms.sort, Type: ms.typ,
				Visible: ms.visible, Status: model.StatusEnabled,
			}
			m.PrepareCreate()
			if ms.key != "" {
				ids[ms.key] = m.ID
			}
			if _, err := db.Collection(model.ColMenu).InsertOne(ctx, m); err != nil {
				return err
			}
			if len(ms.children) > 0 {
				if err := insertMenu(ms.children, m.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := insertMenu(seeds, ""); err != nil {
		return err
	}

	// ---------- 角色 ----------
	// 只读角色：可见页面 + 查询按钮
	viewerMenus := []string{
		ids["m:dashboard"], ids["m:system"], ids["m:tools"],
		ids["m:users"], ids["m:roles"], ids["m:depts"], ids["m:dicts"],
		ids["m:configs"], ids["m:notices"], ids["m:files"],
		ids["m:oplogs"], ids["m:loginlogs"], ids["m:online"],
		ids["m:jobs"], ids["m:monitor"],
		ids["b:user:create"], ids["b:user:update"], ids["b:user:delete"], ids["b:user:resetpwd"], ids["b:user:status"],
		ids["b:role:create"], ids["b:role:update"], ids["b:role:delete"], ids["b:role:assign"],
		ids["b:menu:create"], ids["b:menu:update"], ids["b:menu:delete"],
		ids["b:dept:create"], ids["b:dept:update"], ids["b:dept:delete"],
		ids["b:dict:create"], ids["b:dict:update"], ids["b:dict:delete"],
		ids["b:config:create"], ids["b:config:update"], ids["b:config:delete"],
		ids["b:notice:create"], ids["b:notice:update"], ids["b:notice:delete"],
		ids["b:file:upload"], ids["b:file:delete"],
		ids["b:oplog:delete"], ids["b:loginlog:delete"], ids["b:online:kick"],
		ids["b:job:create"], ids["b:job:update"], ids["b:job:delete"], ids["b:job:run"],
	}
	roles := []*model.Role{
		{Name: "超级管理员", Code: model.SuperRoleCode, Sort: 1, Status: model.StatusEnabled,
			Remark: "内置角色，拥有全部权限", Menus: []string{}, BuiltIn: true},
		{Name: "示例：运维管理员", Code: "ops", Sort: 2, Status: model.StatusEnabled,
			Remark: "演示用：除权限管理外的系统管理权限", Menus: []string{
				ids["m:dashboard"], ids["m:system"], ids["m:tools"],
				ids["m:users"], ids["m:depts"], ids["m:dicts"], ids["m:notices"], ids["m:files"],
				ids["m:oplogs"], ids["m:loginlogs"], ids["m:online"], ids["m:jobs"], ids["m:monitor"],
				ids["b:user:create"], ids["b:user:update"], ids["b:user:delete"], ids["b:user:resetpwd"], ids["b:user:status"],
				ids["b:dict:create"], ids["b:dict:update"], ids["b:dict:delete"],
				ids["b:notice:create"], ids["b:notice:update"], ids["b:notice:delete"],
				ids["b:file:upload"], ids["b:file:delete"],
				ids["b:oplog:delete"], ids["b:loginlog:delete"], ids["b:online:kick"],
				ids["b:job:create"], ids["b:job:update"], ids["b:job:delete"], ids["b:job:run"],
			}},
		{Name: "示例：只读用户", Code: "viewer", Sort: 3, Status: model.StatusEnabled,
			Remark: "演示用：仅可查看数据，按钮权限为空（仅演示页面展示）", Menus: []string{
				ids["m:dashboard"], ids["m:system"], ids["m:tools"],
				ids["m:users"], ids["m:depts"], ids["m:dicts"], ids["m:notices"], ids["m:monitor"],
			}},
	}
	// viewer 去掉演示中的按钮 ID（上面误加，修正为纯只读）
	viewerMenus = viewerMenus[:14]
	roles[2].Menus = viewerMenus

	for _, r := range roles {
		r.PrepareCreate()
		if _, err := db.Collection(model.ColRole).InsertOne(ctx, r); err != nil {
			return err
		}
	}

	// ---------- 用户 ----------
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	usersList := []*model.User{
		{Username: "admin", Nickname: "超级管理员", Password: string(hash),
			Email: "admin@example.com", Roles: []string{model.SuperRoleCode},
			DeptID: childDepts["技术部"],
			Status: model.StatusEnabled, Remark: "内置管理员"},
		{Username: "zhangsan", Nickname: "张三", Password: string(hash),
			Email: "zhangsan@example.com", Gender: 1, Roles: []string{"ops"},
			DeptID: childDepts["技术部"],
			Status: model.StatusEnabled, Remark: "演示用户"},
		{Username: "lisi", Nickname: "李四", Password: string(hash),
			Gender: 2, Roles: []string{"viewer"},
			DeptID: childDepts["市场部"],
			Status: model.StatusEnabled, Remark: "演示用户"},
	}
	for _, u := range usersList {
		u.PrepareCreate()
		if _, err := users.InsertOne(ctx, u); err != nil {
			return err
		}
	}

	// ---------- 字典 ----------
	dictTypes := []*model.DictType{
		{Name: "系统状态", Code: "sys_status", Status: model.StatusEnabled, Remark: "通用启用/停用状态"},
		{Name: "性别", Code: "sys_gender", Status: model.StatusEnabled, Remark: ""},
		{Name: "公告类型", Code: "sys_notice_type", Status: model.StatusEnabled, Remark: ""},
		{Name: "登录状态", Code: "sys_login_status", Status: model.StatusEnabled, Remark: ""},
	}
	for _, t := range dictTypes {
		t.PrepareCreate()
		if _, err := db.Collection(model.ColDictType).InsertOne(ctx, t); err != nil {
			return err
		}
	}
	dictItems := []*model.DictItem{
		{TypeCode: "sys_status", Label: "启用", Value: "1", TagType: "success", Sort: 1, Status: model.StatusEnabled},
		{TypeCode: "sys_status", Label: "停用", Value: "2", TagType: "danger", Sort: 2, Status: model.StatusEnabled},
		{TypeCode: "sys_gender", Label: "未知", Value: "0", Sort: 0, Status: model.StatusEnabled},
		{TypeCode: "sys_gender", Label: "男", Value: "1", Sort: 1, Status: model.StatusEnabled},
		{TypeCode: "sys_gender", Label: "女", Value: "2", Sort: 2, Status: model.StatusEnabled},
		{TypeCode: "sys_notice_type", Label: "通知", Value: "1", TagType: "info", Sort: 1, Status: model.StatusEnabled},
		{TypeCode: "sys_notice_type", Label: "公告", Value: "2", TagType: "warning", Sort: 2, Status: model.StatusEnabled},
		{TypeCode: "sys_login_status", Label: "成功", Value: "1", TagType: "success", Sort: 1, Status: model.StatusEnabled},
		{TypeCode: "sys_login_status", Label: "失败", Value: "2", TagType: "danger", Sort: 2, Status: model.StatusEnabled},
	}
	for _, i := range dictItems {
		i.PrepareCreate()
		if _, err := db.Collection(model.ColDictItem).InsertOne(ctx, i); err != nil {
			return err
		}
	}

	// ---------- 系统参数 ----------
	configs := []*model.SysConfig{
		{Key: "sys.name", Name: "系统名称", Value: "ServerAdmin 后台管理系统", BuiltIn: true, Remark: "登录页与浏览器标题"},
		{Key: "sys.version", Name: "系统版本", Value: "1.0.0", BuiltIn: true, Remark: ""},
		{Key: "sys.copyright", Name: "版权信息", Value: "© 2026 ServerAdmin", BuiltIn: true, Remark: "展示在登录页底部"},
		{Key: "sys.user.initPassword", Name: "新用户初始密码", Value: "admin123", Remark: "创建用户未指定密码时使用"},
		{Key: "sys.upload.allowedExt", Name: "上传扩展名白名单", Value: "", Remark: "留空表示不限制"},
	}
	for _, c := range configs {
		c.PrepareCreate()
		if _, err := db.Collection(model.ColSysConfig).InsertOne(ctx, c); err != nil {
			return err
		}
	}

	// ---------- 通知公告 ----------
	notices := []*model.Notice{
		{Title: "欢迎使用 ServerAdmin 后台管理系统", Type: model.NoticeTypeAnnouce,
			Content: "系统已完成初始化。\n\n默认管理员账号：admin / admin123，请登录后及时修改密码。\n\n本次更新内容：\n1. 用户/角色/菜单/部门管理\n2. 操作日志与登录日志\n3. 定时任务与系统监控",
			Status:  1, Publisher: "admin"},
		{Title: "服务器例行维护通知", Type: model.NoticeTypeNotice,
			Content: "本周六 02:00-04:00 进行例行维护，期间系统可能短暂不可用，请提前保存工作内容。",
			Status:  1, Publisher: "admin"},
	}
	for _, notice := range notices {
		notice.PrepareCreate()
		if _, err := db.Collection(model.ColNotice).InsertOne(ctx, notice); err != nil {
			return err
		}
	}

	// ---------- 定时任务 ----------
	jobs := []*model.Job{
		{Name: "清理过期令牌", Spec: "30 * * * *", Handler: "cleanup_expired_tokens",
			Status: model.StatusEnabled, Remark: "每小时清理已过期的令牌黑名单"},
		{Name: "清理历史操作日志", Spec: "0 2 * * *", Handler: "cleanup_operation_logs",
			Params: `{"days":90}`, Status: model.StatusEnabled, Remark: "每天凌晨 2 点清理 90 天前的操作日志"},
		{Name: "心跳演示任务", Spec: "*/5 * * * *", Handler: "demo_heartbeat",
			Status: model.StatusDisabled, Remark: "演示任务，启用后每 5 分钟输出一次心跳"},
	}
	for _, j := range jobs {
		j.PrepareCreate()
		if _, err := db.Collection(model.ColJob).InsertOne(ctx, j); err != nil {
			return err
		}
	}

	logger.L.Info(fmt.Sprintf("种子数据初始化完成：用户 %d 个（admin/admin123），角色 %d 个，部门 5 个，字典 4 类，任务 %d 个",
		len(usersList), len(roles), len(jobs)))
	return nil
}
