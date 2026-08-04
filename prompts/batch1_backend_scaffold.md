# 任务批次 1：Go 后端脚手架

你是资深 Go 工程师。请在本项目根目录下创建后端脚手架，**严格遵守**以下要求。项目根目录：`C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。

## 目标
搭建一个可编译、可运行的 Gin + GORM + PostgreSQL 后端骨架，包含配置加载、数据库连接、自动建表与种子数据、健康检查接口、统一响应封装。

## 必须创建的目录结构与文件

```
backend/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go        # 从 .env 加载配置（端口、数据库 DSN、JWT 密钥、Token 过期时间）
│   ├── database/database.go    # GORM 连接（pgx 驱动）+ AutoMigrate + Seed
│   ├── models/                 # 本批次先建 users/roles/permissions 三个模型
│   ├── response/response.go    # 统一响应 + 分页
│   ├── router/router.go        # 路由注册
│   └── handler/health.go       # 健康检查
├── .env                        # 实际配置（数据库密码 3143）
├── .env.example                # 示例配置
├── .gitignore                  # 忽略 .env、二进制、日志
└── go.mod
```

## 具体要求

1. **go.mod**：module 名 `hrms/backend`，Go 版本 1.25。依赖：gin、gorm、gorm.io/driver/postgres、joho/godotenv、golang-jwt/jwt/v5（后面批次会用，先加上）。

2. **config.go**：结构体 Config{Port, DBHost, DBPort, DBUser, DBPassword, DBName, JWTSecret, TokenExpireHours}，`Load()` 用 godotenv 读 .env，缺失项给默认值（Port=8080, JWTSecret 默认 "dev-secret-change-me"）。

3. **database.go**：
   - `Connect(cfg)` 返回 *gorm.DB，DSN 用 `host=... user=... password=... dbname=... port=... sslmode=disable TimeZone=Asia/Shanghai`
   - `AutoMigrate(db)` 迁移 models 里所有模型
   - `Seed(db)` 写入种子数据：内置角色 admin、hr、manager、employee；内置管理员账号 admin/admin123（bcrypt 哈希），role=admin。幂等：已存在则跳过。

4. **models**（GORM 标签齐全，用 `gorm:"..."` 指定约束）：
   - `User`: ID, Username(unique, not null), PasswordHash, Email, FullName, RoleID, Status(1启用/0禁用), Timestamps
   - `Role`: ID, Name, Code(unique), Description
   - `Permission`: ID, Name, Code(unique), Module
   - `RolePermission`: RoleID, PermissionID（复合唯一）
   - 用 `DeletedAt` 软删除（gorm.DeletedAt）

5. **response.go**：
   - `OK(c, data)` → 200 + {"code":0,"message":"ok","data":...}
   - `Error(c, httpStatus, code, message)` → 统一错误
   - `PageOK(c, list, total, page, pageSize)` → {code:0, data:{list,total,page,page_size}}
   - 提供 `Pagination(c)` 从 query 读 page/page_size（默认 1/20，上限 100）

6. **router.go**：注册 `/api/v1/health` GET → handler.Health（返回 {"code":0,"data":{"status":"ok","db":"up"}}，并执行 `db.Exec("SELECT 1")` 探测数据库，失败返回 503）。

7. **main.go**：加载配置 → 连接数据库 → AutoMigrate → Seed → 设置 Gin release 模式 → 注册路由 → `r.Run(":" + port)`。启动时用 slog 打印监听地址。

8. **代码质量**：handler/service/repository 分层先搭好雏形（health 直接调 db 即可，但后续模块必须分层）。所有注释用中文。代码风格 gofmt。

## 完成后必须执行
- `cd backend && go mod tidy && go build ./...` 编译通过（缺依赖就 go get）
- 启动 `go run ./cmd/server` 确认 `curl http://localhost:8080/api/v1/health` 返回 `"db":"up"`
- 用 `curl -s "http://localhost:8080/api/v1/health"` 验证后，把进程停掉

## 汇报格式
完成后汇报：创建了哪些文件、编译是否通过、health 接口返回内容、seed 了哪些角色和账号。
