# ServerAdmin 后台管理系统

基于 **Go (Gin) + MongoDB** 的开箱即用后台管理系统。单一二进制内嵌前端管理界面，首次启动自动完成建库、建索引与演示数据初始化，登录即用。

## 功能模块

| 模块 | 说明 |
| --- | --- |
| 🔐 认证与安全 | 图形验证码登录、JWT 双令牌（Access + Refresh）、令牌黑名单、修改密码 |
| 👤 用户管理 | 用户 CRUD、启用/禁用、重置密码、按部门/状态筛选、批量删除 |
| 🎭 角色管理 | RBAC 角色 CRUD、菜单/按钮权限分配、内置角色保护 |
| 🧭 菜单管理 | 目录/菜单/按钮三级树，前端路由与权限标识（如 `sys:user:create`） |
| 🏢 部门管理 | 树形组织架构，用户归属部门，按部门级联筛选用户 |
| 📖 字典管理 | 字典类型 + 字典项，前端下拉/标签色统一取值 |
| 🔧 参数配置 | 系统参数 Key-Value，内置参数保护 |
| 📢 通知公告 | 通知/公告发布、草稿/发布状态 |
| 📁 文件管理 | 本地磁盘存储、上传/下载/删除、大小限制、文件登记 |
| 📝 操作日志 | 自动记录非 GET 请求（模块/动作/IP/耗时/业务码），请求体脱敏，可清空 |
| 🔐 登录日志 | 登录成功/失败记录，按时间/状态筛选 |
| 🟢 在线用户 | 内存会话登记、强制下线（拉黑令牌） |
| ⏰ 定时任务 | robfig/cron 调度，任务 CRUD、启停、立即执行、执行日志，内置 4 个示例处理器 |
| 🖥️ 系统监控 | CPU/内存/磁盘、主机信息、Go 运行时指标 |
| 📊 仪表盘 | 统计卡片、近 7 天登录趋势、最近操作、最新公告 |
| ❤️ 健康检查 | `/healthz` 存活、`/readyz` 就绪（含数据库连通性） |
| 🧱 基础设施 | 配置中心（文件+环境变量）、zap 日志轮转、统一响应、优雅停机、CORS、自动索引与种子数据 |

## 技术栈

- **Web 框架**：[Gin](https://github.com/gin-gonic/gin)
- **数据库**：MongoDB（官方 [mongo-go-driver](https://github.com/mongodb/mongo-go-driver)）
- **认证**：[golang-jwt/v5](https://github.com/golang-jwt/jwt)（HS256 双令牌）+ [base64Captcha](https://github.com/mojocn/base64Captcha)
- **配置**：[Viper](https://github.com/spf13/viper)（config.yaml + `SA_` 前缀环境变量覆盖）
- **日志**：[zap](https://go.uber.org/zap) + [lumberjack](https://github.com/natefinch/lumberjack) 轮转
- **定时任务**：[robfig/cron/v3](https://github.com/robfig/cron)
- **监控**：[gopsutil/v3](https://github.com/shirou/gopsutil)
- **前端**：原生 JS 单页应用（零依赖，`go:embed` 打进二进制）

## 快速开始

### 1. 启动 MongoDB

```bash
# 方式一：Docker（推荐）
docker run -d --name serveradmin-mongo -p 27017:27017 -v mongo_data:/data/db mongo:7

# 方式二：本地已安装 MongoDB，确保 27017 端口可连即可
```

### 2. 启动服务

```bash
go mod tidy        # 首次拉取依赖
go run ./cmd/server
```

### 3. 登录

浏览器打开 **http://localhost:8080**，使用默认账号登录：

| 账号 | 密码 | 角色 |
| --- | --- | --- |
| admin | admin123 | 超级管理员（全部权限） |
| zhangsan | admin123 | 示例：运维管理员 |
| lisi | admin123 | 示例：只读用户 |

> 登录页会显示图形验证码，自动化测试可用 `SA_CAPTCHA_ENABLED=false` 关闭。

#### Windows 一键脚本

```bat
scripts\start-all.bat   # 启动 MongoDB + ServerAdmin（最小化窗口，路径按需修改）
scripts\stop-all.bat    # 停止两者
```

## 配置说明

所有配置项都可通过环境变量覆盖（前缀 `SA_`，`.` 换成 `_`），例如：

```bash
SA_APP_PORT=9090 SA_MONGO_URI="mongodb://user:pass@host:27017" SA_JWT_SECRET=xxx go run ./cmd/server
```

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| `app.port` | 8080 | HTTP 端口 |
| `app.env` | dev | dev/prod，prod 关闭调试模式 |
| `app.uploadDir` | ./uploads | 上传目录 |
| `app.maxUploadMB` | 20 | 单文件上限 |
| `mongo.uri` | mongodb://localhost:27017 | 连接串 |
| `mongo.database` | server_admin | 库名 |
| `jwt.secret` | （请修改） | 签名密钥，生产必改 |
| `jwt.accessExpireMinutes` | 120 | 访问令牌有效期 |
| `jwt.refreshExpireHours` | 168 | 刷新令牌有效期（7 天） |
| `captcha.enabled` | true | 登录验证码开关 |
| `oplog.enabled` | true | 操作日志开关 |
| `seed.enabled` | true | 首次启动初始化演示数据 |

## API 概览

统一前缀 `/api/v1`，响应格式 `{code, msg, data}`（code=0 成功）。除公开接口外均需 `Authorization: Bearer <token>`。

```
# 认证（公开）
GET    /auth/captcha                 获取验证码
POST   /auth/login                   登录 {username,password,captchaId,captchaCode}
POST   /auth/refresh                 刷新令牌 {refreshToken}
# 认证（需登录）
POST   /auth/logout                  登出
GET    /auth/profile                 个人信息+权限+菜单
PUT    /auth/profile                 修改个人资料
PUT    /auth/profile/password        修改密码

# 用户  GET/POST /users, GET/PUT /users/:id, DELETE /users {ids},
#       PUT /users/:id/status, PUT /users/:id/password
# 角色  GET/POST /roles, GET/PUT /roles/:id, DELETE /roles {ids}, PUT /roles/:id/menus
# 菜单  GET /menus/tree, POST /menus, PUT/DELETE /menus/:id
# 部门  GET /departments/tree, POST /departments, PUT/DELETE /departments/:id
# 字典  GET/POST /dict-types, PUT /dict-types/:id, DELETE /dict-types {ids}
#       GET/POST /dict-items, PUT /dict-items/:id, DELETE /dict-items {ids}
#       GET /dicts/:code（登录即可）
# 参数  GET/POST /configs, PUT /configs/:id, DELETE /configs {ids}, GET /configs/public
# 公告  GET/POST /notices, PUT /notices/:id, DELETE /notices {ids}
# 文件  GET /files, POST /files/upload(multipart: file), GET /files/:id/download, DELETE /files {ids}
# 日志  GET/DELETE /oplogs, GET/DELETE /loginlogs（DELETE 不带 ids 即清空）
# 在线  GET /online, DELETE /online/:jti
# 任务  GET/POST /jobs, PUT /jobs/:id, DELETE /jobs {ids},
#       PUT /jobs/:id/status, PUT /jobs/:id/run, GET /jobs/handlers, GET /job-logs
# 其他  GET /dashboard/stats, GET /monitor/server
# 健康检查（公开）  GET /healthz, GET /readyz
```

## 目录结构

```
ServerAdmin/
├── cmd/server/            # 启动入口（配置、连接、优雅停机）
├── config/config.yaml     # 配置文件（加载顺序：SA_CONFIG > ./config > .）
├── internal/
│   ├── handler/           # HTTP 接口层（参数绑定/响应）
│   ├── service/           # 业务层（业务规则、缓存编排、权限）
│   ├── repository/        # 数据访问层（全部 MongoDB 读写）
│   ├── cache/             # Redis 缓存（Cache-Aside/singleflight/Key 规范）
│   ├── config/            # 配置加载（Viper）
│   ├── database/          # MongoDB 连接与索引
│   ├── middleware/         # CORS/JWT 认证/权限/操作日志/响应缓存/访问日志
│   ├── model/             # 文档模型
│   ├── pkg/               # errs/response/jwtx/logger 通用包
│   ├── webui/             # 内嵌前端（go:embed）
│   └── static assets      # index.html + app.css + core.js + pages*.js
├── logs/                  # 运行日志（自动创建）
└── uploads/               # 上传文件（自动创建）
```

## 二次开发：新增一个业务模块

1. `internal/model/xxx.go` 定义文档结构，并在 `internal/database/mongo.go` 注册索引；
2. `internal/repository/xxx.go` 定义数据访问方法（复用 `collection[T]` 泛型基类）；
3. `internal/service/xxx.go` 实现业务方法（缓存读写与失效在此编排）；
4. `internal/handler/xxx.go` 绑定参数、调用 service，用 `response.OK/Page/Handle` 返回；
5. `internal/handler/router.go` 注册路由，并按需在 `middleware.RequirePerm("xxx:yyy")` 上挂权限；
5. 若需要在菜单中可见：登录后到【菜单管理】新增菜单/按钮（权限标识与路由一致）。

前端新增页面：在 `internal/webui/static/assets/pages*.js` 中添加 `window.Pages.xxx = function(container){...}`，菜单的 `component` 填 `xxx` 即可自动路由。

## 生产部署建议

- 修改 `jwt.secret`（或用 `SA_JWT_SECRET` 注入），启用 MongoDB 认证并修改连接串；
- `app.env=prod` 关闭调试模式；日志/上传目录挂载到持久卷；
- 前置 Nginx 做 TLS 终结；`/healthz`、`/readyz` 接入负载均衡探活；
- 在线用户与令牌黑名单当前为单实例内存实现，多副本部署时需替换为 Redis/MongoDB 查询（`internal/service/online.go`、`token.go` 已预留清晰的替换边界）。

## 常见问题

- **启动报 MongoDB Ping 失败**：确认 mongod 已启动、URI 正确；Docker 一条命令见“快速开始”。
- **无法登录验证码报错**：验证码 5 分钟有效，点击图片可刷新；自动化场景用 `SA_CAPTCHA_ENABLED=false`。
- **首次启动没有菜单/用户**：`seed.enabled=true` 且 `users` 集合为空时才会初始化；如需重建，删除数据库后重启。
