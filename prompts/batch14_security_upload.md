# 任务批次 14：登录安全加固 + 文件上传

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。项目功能完备（含用户/角色管理、审计日志）。本批次实现**登录安全加固**与**文件上传**。严格分层，统一响应，JWT + 权限点。

## 一、登录安全加固

### 1. 图形验证码（用 github.com/mojocn/base64Captcha）
- 安装：`cd backend && go get github.com/mojocn/base64Captcha`
- `GET /api/v1/auth/captcha`（公开，无 JWT）→ 返回 `{captcha_id, captcha_image}`（base64 PNG，4 位数字+字母）
- 验证码存储：内存 store（base64Captcha 自带 `base64Captcha.DefaultMemStore`）即可，单实例够用
- 登录接口改造：`POST /auth/login` 请求体增加 `captcha_id`、`captcha_code`（必填，校验失败返回 400「验证码错误或已过期」）
- 登录成功后验证码作废（base64Captcha store 的 Verify 默认消费一次）

### 2. 登录失败锁定（防暴力破解）
- `users` 表新增字段：`FailedLoginCount int`（默认 0）、`LockedUntil *time.Time`（可空）
- 规则：连续失败 5 次 → `LockedUntil = now + 15分钟`，同时清空计数；锁定期间（LockedUntil > now）直接拒绝登录（即使密码正确），返回 400「账号已锁定，请 15 分钟后再试」
- 登录成功 → 清空 FailedLoginCount 与 LockedUntil
- AutoMigrate 自动加列（注意：已有表加非空字段需给默认值，FailedLoginCount 用 `gorm:"default:0"` 即可）
- 审计日志：锁定触发时记录一条 `module=auth, action=lock`

### 3. Token 刷新机制
- 登录响应改为：`{token, refresh_token, expires_in}`（token 仍 24h；refresh_token 有效期 7 天，JWT claims 加 `"type":"refresh"`）
- `POST /api/v1/auth/refresh`（公开）→ body `{refresh_token}` → 校验 type=refresh 且未过期 → 返回新的 `{token, refresh_token, expires_in}`（双刷新，滚动续期）
- 中间件 JWTAuth 校验时，若 claims.type == "refresh" 则拒绝（refresh token 不能当 access 用）
- 修改密码后（可选）可通过校验旧密码逻辑自然失效，不做强制吊销

### 4. JWT Secret 强化
- `.env` 中生成一个随机 64 字符 JWT_SECRET（用 `openssl rand -hex 32` 或 `python -c "import secrets;print(secrets.token_hex(32))"` 生成后写入 backend/.env）
- config.Load 中：若 JWT_SECRET 仍是默认值 "dev-secret-change-me"，启动时 slog.Warn 警告（不强退，便于开发）

## 二、文件上传

### 1. 上传接口（permission: 登录即可，但仅限上传头像类图片）
- `POST /api/v1/upload`（JWT 保护，无额外权限点）→ multipart 字段名 `file`
- 校验：大小 ≤ 5MB；MIME 类型仅 image/jpeg、image/png、image/webp、image/gif（用扩展名+content-type 双重判断）
- 保存：`backend/uploads/` 下按 `YYYYMM/` 分子目录，文件名 `<uuid>.ext`（用 github.com/google/uuid 或 crypto/rand 生成）
- 返回：`{url: "/uploads/202608/xxxx.png"}`（相对路径，前端拼接 origin）
- 目录不存在自动创建；`.gitignore` 增加 `backend/uploads/`

### 2. 静态文件服务
- router 中挂载：`r.Static("/uploads", "./uploads")`（注意：应在 `/api` 路由外注册，Gin 原生支持）
- 验证 `curl -s -o /dev/null -w "%{http_code}" localhost:8080/uploads/<path>` 返回 200

### 3. 员工头像使用
- 前端（后续批次做，本批次只保证后端）：员工模型已有 Avatar 字段，上传后前端把 url 存到员工记录的 avatar 字段即可（PUT /employees/:id 已支持）

## 三、完成后必须执行
1. `cd backend && go mod tidy && go build ./...`、`go vet ./...`
2. 启动验证（admin 登录）：
   - GET /auth/captcha 返回 captcha_id + 图片 base64
   - 不带验证码登录 → 400；带错误验证码 → 400「验证码错误或已过期」
   - 用 Python（或 curl + 直接调用 base64Captcha 校验接口）模拟拿到验证码后正确登录 → 200 且响应含 refresh_token
   - 连续输错密码 5 次 → 第 6 次（即使密码正确）返回 400 锁定提示；审计日志出现 lock 记录
   - POST /auth/refresh 用 refresh_token 换新 token 成功；用 access token 调 refresh → 拒绝
   - 上传一张图片（用 Python 生成 1x1 PNG 或用已有图片）→ 返回 url → GET 该 url 200 → 上传 6MB 文件 → 400/413
3. 停掉进程，端口释放

## 汇报格式
新增/修改文件清单、各安全项验证结果（含实际响应）、上传验证结果、JWT_SECRET 是否已生成。
