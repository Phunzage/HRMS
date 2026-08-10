# HRMS 系统设计架构文档（完整版）

> 依据实际代码（`backend/internal/models`、`router/router.go`、`database/seed.go`、`desktop/`、`frontend/android/`）整理，与实现保持一致。
> 更新日期：2026-08-10

---

## 1. 系统架构

前后端分离的**单体架构**（Monolith-first），部署简单；模块边界清晰，未来可按模块拆分微服务。系统支持**三端形态**：浏览器网页版、Windows 桌面端（Electron loadURL 线上前端）、Android 端（Capacitor 远程加载模式），三者共用同一后端与数据库。

```
┌───────────────────────────────────────────────────────────────────────┐
│ 浏览器 (React SPA)         Windows 桌面端 (Electron)   Android (Capacitor) │
│  React 18 + TypeScript    loadURL 线上前端              WebView 远程加载     │
│  TailwindCSS + daisyUI    系统通知 (notify IPC)           LocalNotifications │
│  zustand / React Router   自动更新 (updater.js)           WorkManager 轮询   │
│  axios（拦截器注入 JWT）                                          TokenStore 桥接 │
└────────────────────────────────────┬──────────────────────────────────┘
                                     │ REST /api/v1 (JSON, Bearer Token)
┌────────────────────────────────────▼──────────────────────────────────┐
│                        Gin HTTP 服务 (Go 1.25)                         │
│  ┌─────────────┬──────────────┬──────────────┬───────────┐            │
│  │ CORS/日志   │ JWT 认证      │ RBAC 权限     │ 统一响应   │            │
│  │ 中间件      │ 中间件        │ 中间件        │ response  │            │
│  └─────────────┴──────────────┴──────────────┴───────────┘            │
│  路由层 router → handler（参数校验）→ service（业务规则）                 │
│               → repository（数据访问）→ GORM                            │
└────────────────────────────────────┬──────────────────────────────────┘
                                     │ PostgreSQL 18 (hrms 库, 24 张表)
                                     ▼
```

### 分层职责（禁止跨层调用）

| 层 | 职责 | 位置 |
|----|------|------|
| `router` | 路由注册、权限中间件挂载 | `internal/router/` |
| `handler` | HTTP 入参绑定/校验、调用 service、组装响应 | `internal/handler/` |
| `service` | 业务规则（审批流、余额计算、状态机、权限范围过滤） | `internal/service/` |
| `repository` | GORM 查询封装 | `internal/repository/` |
| `models` | GORM 模型（表结构映射） | `internal/models/` |
| `middleware` | JWTAuth / RequirePermission | `internal/middleware/` |
| `database` | 连接、AutoMigrate 建表、Seed 种子数据 | `internal/database/` |

### 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 + Gin v1.12 + GORM v2（pgx 驱动） |
| 数据库 | PostgreSQL 18（hrms 库） |
| 认证 | JWT (golang-jwt/v5, HS256) + bcrypt |
| 配置 | godotenv + .env |
| 日志 | log/slog |
| 前端 | React 18 + TypeScript + Vite 8 |
| UI | TailwindCSS v4 + daisyUI 5 |
| 图标 | lucide-react |
| 前端状态 | zustand（persist localStorage） |
| HTTP | axios（拦截器） |
| Windows 桌面端 | Electron + electron-builder（main/preload/updater/notify） |
| Android 端 | Capacitor + WorkManager + @capacitor/local-notifications + TokenStore 桥接 |

---

## 2. 数据库设计（24 张表）

> 表名与 `backend/internal/database/database.go` 的 AutoMigrate 清单一致（实测 `\dt` 24 张）。

### 2.1 表清单

| 模块 | 表 | 说明 |
|------|----|------|
| 认证权限 | `users` | 系统账号（登录名、bcrypt 密码、角色、关联员工、失败锁定、数据范围） |
| 认证权限 | `roles` | 角色（admin/hr/manager/employee，可自定义） |
| 认证权限 | `permissions` | 权限点（30 个，格式 `模块:动作`） |
| 认证权限 | `role_permissions` | 角色-权限关联（复合主键） |
| 认证权限 | `audit_logs` | 操作审计日志（永不删除） |
| 组织架构 | `departments` | 部门（树形，ParentID 自关联，ManagerID 负责人） |
| 组织架构 | `positions` | 岗位（Code 唯一，隶属部门） |
| 人事 | `employees` | 员工档案（工号唯一，归属部门/岗位，Avatar） |
| 考勤 | `attendances` | 考勤记录（员工+日期联合唯一；状态含 leave） |
| 考勤 | `attendance_corrections` | 补卡申请（check_in/check_out 修正审批流） |
| 请假 | `leave_requests` | 请假申请（业务数据，审批走通用引擎） |
| 请假 | `leave_balances` | 假期余额（员工+类型+年份联合唯一） |
| 薪酬 | `salary_records` | 工资单（员工+月份，净额自动计算，支持批量生成） |
| 招聘 | `job_openings` | 招聘职位 |
| 招聘 | `candidates` | 候选人（状态流转） |
| 通知 | `notifications` | 站内通知（审批待办/结果/补卡/系统） |
| 公告 | `announcements` | 公告（置顶/草稿/发布） |
| 公告 | `announcement_reads` | 公告已读记录（用户+公告联合唯一） |
| 绩效 | `performance_reviews` | 绩效考核（员工+周期唯一，S/A/B/C/D 评级） |
| 审批流 | `approval_flows` | 审批流程定义（请假/补卡各一条，active） |
| 审批流 | `approval_steps` | 流程步骤（StepOrder 1~5，审批人类型 manager/role/position/permission） |
| 审批流 | `approval_instances` | 审批实例（一次申请一条，当前步骤、状态） |
| 审批流 | `approval_tasks` | 审批任务（每级一个，审批人/状态/意见） |
| 更新 | `app_versions` | 客户端版本记录（win/android，版本号+包路径+强制标记） |

### 2.2 表关系 ER 图

```
roles ──1:N── users ──N:1── employees(可选, 通过 user.employee_id)
roles ──N:M── permissions（通过 role_permissions 关联表）
departments ──self── parent_id（树形）
departments ──1:N── positions（岗位所属部门）
departments ──1:N── employees（员工所属部门）
departments ──1:N── job_openings（招聘需求部门）
positions ──1:N── employees（员工所属岗位）
positions ──1:N── job_openings（招聘对应岗位）
employees ──1:N── attendances（每人每天一条）
employees ──1:N── leave_requests（请假人）
employees ──1:N── leave_balances（按年度分类型余额）
employees ──1:N── salary_records（每人每月一条）
employees ──1:N── performance_reviews（每人每周期一条）
leave_requests ──1:1── approval_instances（biz_type=biz_id 关联业务单）
attendance_corrections ──1:1── approval_instances
approval_flows ──1:N── approval_steps（流程 → 步骤）
approval_instances ──1:N── approval_tasks（实例 → 任务，按步骤顺序生成）
notifications ──N:1── users（收件人 receiver_id）
announcements ──N:M── users（已读记录 announcement_reads 关联表）
```

### 2.3 关键约束与索引

| 表 | 约束/索引 | 作用 |
|----|----------|------|
| users | username 唯一 | 登录名唯一 |
| employees | emp_no 唯一 | 工号唯一（EMP0001 起，advisory lock 防并发重复） |
| attendances | (employee_id, work_date) 联合唯一 | 每人每天只能一条考勤 |
| leave_balances | (employee_id, leave_type, year) 联合唯一 | 每年每类型一条余额 |
| salary_records | (employee_id, month) 业务唯一 | 每人每月一张工资单（service 层校验） |
| performance_reviews | (employee_id, period) 业务唯一 | 每人每周期一张（service 层校验） |
| announcement_reads | (announcement_id, user_id) 联合唯一 | 每条公告每用户一条已读 |
| app_versions | (platform, version) 联合唯一 | 每平台版本号唯一 |
| role_permissions | 复合主键 | 角色权限不重复 |
| approval_tasks | (instance_id, step_order) 唯一 | 每实例每级一个任务 |
| 全部业务表 | deleted_at 软删除 | 数据可追溯，GORM 自动过滤 |

### 2.4 核心状态机

```
通用审批流引擎（请假 / 补卡，1~5 级按 admin 配置推进）:

  提交 → 创建 ApprovalInstance（pending）+ 按流程配置生成 ApprovalTask 链
  每一级任务:   pending ──approve──▶ approved（自动推进下一级 / 终审触发业务回调）
                pending ──reject───▶ rejected（填原因，终止流程）
                pending ──cancel──▶ cancelled（申请人取消）
  终审 approve 同事务：业务回调 + 通知申请人
      - 请假：扣减余额 + 考勤区间标记 leave + 通知申请人
      - 补卡：更新考勤打卡时间并重算状态 + 通知申请人
  审批人解析（按步骤类型）：
      - manager   → 部门经理（递归向上查父部门经理）
      - role      → 指定角色池（该角色全部启用用户，未指派则广播）
      - position  → 指定岗位池（岗位内全部在职员工）
      - permission→ 按审批权限点匹配（如 leave:approve 拥有者），未指派则广播

工资单:     draft ──confirm──▶ confirmed ──pay──▶ paid
            （仅 draft 可编辑，confirm 后不可改；支持按月批量生成）

绩效:       draft ──submit──▶ submitted ──approve──▶ approved（hr）
            （draft 可编辑/删除，submitted 后可批准）

考勤状态:   check-in>9:00 → late；check-out<18:00 → early_leave
            两者皆正常 → normal；无打卡 → absent；请假审批通过 → leave

候选人:     applied → interview → offer → hired / rejected

员工状态:   1 在职 / 2 离职 / 3 试用
```

---

## 3. 功能结构

### 3.1 前端页面结构

```
/login            登录（验证码 + GuestOnly 守卫）
/                 仪表盘（统计卡片 + 部门分布 + 工资/招聘趋势 + 最新公告）
/approvals        统一待办中心（待我审批 / 我处理的 + 审批进度时间线）
/employees        员工管理（列表/搜索/分页/新增/编辑/删除，联动建账号）
/departments      部门管理（树形展示、增删改）
/positions        岗位管理（列表 CRUD）
/attendance       考勤（打卡、补录、补卡申请 Tab、考勤月报 Tab）
/leaves           请假审批（我的申请/待我审批/全部，余额展示）
/salaries         薪酬（列表、批量生成、确认/发放，员工仅本人）
/recruitment      招聘（职位 Tab + 候选人 Tab）
/announcements    公告管理（置顶/搜索，admin/hr 可编辑）
/performance      绩效管理（周期/状态筛选，评级 badge，状态流操作）
/profile          个人中心（档案 + 修改密码 + 头像上传）
/users            用户管理（admin，含负责部门范围设置）
/roles            角色管理（admin，权限分组勾选）
/approval-flows   审批流程配置（admin：请假/补卡级别与审批人类型）
/audit-logs       审计日志（admin 只读）

移动端适配：MainLayout 抽屉侧边栏（lg 常驻 / 移动 lg:hidden 抽屉 + 汉堡 + 遮罩）、
表格 overflow-x-auto 横滑、Modal max-w-[min(95vw,42rem)]、工具栏 flex-wrap。
导航菜单按权限点驱动（usePerm：has/hasAny），无权限菜单隐藏。
```

### 3.2 API 一览（前缀 /api/v1，JWT 保护）

| 模块 | 端点 |
|------|------|
| 健康 | GET /health（公开） |
| 认证 | GET /auth/captcha（公开）、POST /auth/login（验证码）、POST /auth/refresh（公开）、GET /auth/me、PUT /auth/password |
| 部门 | GET /departments(树)、POST、PUT /:id、DELETE /:id |
| 岗位 | GET(分页)、POST、PUT /:id、DELETE /:id |
| 员工 | GET(分页)、GET /me、PUT /me/avatar（自助）、GET /export、POST、GET /:id、PUT /:id、DELETE /:id |
| 考勤 | GET(分页)、GET /report（月报）、POST /check-in、/check-out、/manual、PUT /:id、DELETE /:id、/corrections 补卡（POST/GET/GET /mine/PUT /:id/cancel） |
| 请假 | POST、GET(分页)、GET /mine、GET /balance、PUT /:id/cancel（审批走 /approvals） |
| 通用审批 | GET /approvals/mine、GET /approvals/biz、PUT /tasks/:taskId/approve、/reject、GET /approvals/:instanceId/tasks |
| 审批流程配置 | GET /approval-flows、PUT /approval-flows/:code（admin） |
| 薪酬 | GET(分页)、GET /mine、GET /export、POST、POST /batch-generate、PUT /:id、/confirm、/pay |
| 招聘 | GET/POST/PUT/DELETE /recruitment/jobs、/recruitment/candidates |
| 通知 | GET /notifications、GET /notifications/recent（增量检测）、GET /unread-count、PUT /:id/read、PUT /read-all |
| 角标 | GET /badges（pending_approvals/unread_announcements/unread_notifications） |
| 公告 | GET/POST/PUT/DELETE /announcements、PUT /:id/read、PUT /read-all |
| 绩效 | GET(分页)、GET /mine、POST、PUT /:id、PUT /:id/submit、/approve、DELETE /:id |
| 用户管理 | GET /users、POST、PUT /:id、PUT /:id/password、DELETE /:id、GET /users/roles |
| 角色管理 | GET /roles、GET /roles/permissions、POST、PUT /:id、DELETE /:id |
| 审计日志 | GET /audit-logs |
| 更新 | GET /update/check（公开）、GET /update/latest（公开）、POST/GET /update/versions（admin） |
| 上传 | POST /upload、GET /uploads/*（静态） |
| 仪表盘 | GET /dashboard/stats |

---

## 4. 权限设计（RBAC）

### 4.1 权限点（30 个，定稿）

| 权限点 | 含义 |
|--------|------|
| employee:read / write | 员工查看/编辑 |
| department:read / write | 部门查看/编辑 |
| position:read / write | 岗位查看/编辑 |
| attendance:read / write | 考勤查看/打卡与补录/补卡 |
| leave:read / write | 请假查看/提交 |
| leave:approve | 请假审批 |
| attendance_correction:approve | 补卡审批 |
| salary:read / write | 薪酬查看/编辑 |
| recruitment:read / write | 招聘查看/编辑 |
| user:read / write | 用户管理查看/编辑 |
| role:read / write | 角色管理查看/编辑 |
| audit:read | 审计日志查看 |
| dashboard:read | 仪表盘查看 |
| announcement:read / write | 公告查看/编辑 |
| performance:read / write | 绩效查看/编辑 |
| approval_flow:read / write | 审批流程查看/配置 |
| update:read / write | 更新版本查看/登记 |

### 4.2 角色-权限矩阵（定稿，admin 无业务权限）

> admin 不再「直接放行」，统一查 role_permissions。admin 仅系统管理（用户/角色/审计/公告/审批流程配置/仪表盘监督）+ position:read（审批流程「指定岗位」步骤需要岗位下拉）。

| 权限 | admin | hr | manager | employee |
|------|:---:|:---:|:---:|:---:|
| employee:read | — | ✅ | ✅ | —（仅 /me 本人 + /me/avatar 自助） |
| employee:write | — | ✅ | — | — |
| department:read | — | ✅ | ✅ | — |
| department:write | — | ✅ | — | — |
| position:read | ✅ | ✅ | ✅ | — |
| position:write | — | ✅ | — | — |
| attendance:read | — | ✅ | ✅ | ✅ |
| attendance:write | — | ✅ | ✅（部门视图/补卡） | ✅（自助打卡/补卡） |
| leave:read | — | ✅ | ✅ | ✅ |
| leave:write | — | ✅ | ✅ | ✅（提交/取消） |
| leave:approve | — | ✅ | ✅ | — |
| attendance_correction:approve | — | ✅ | —（可配置） | — |
| salary:read | — | ✅ | — | ✅（仅本人） |
| salary:write | — | ✅ | — | — |
| recruitment:read/write | — | ✅ | — | — |
| performance:read | — | ✅ | ✅ | ✅（仅自己） |
| performance:write | — | ✅ | ✅（本部门） | — |
| dashboard:read | ✅ | ✅ | — | — |
| announcement:read | ✅ | ✅ | ✅ | ✅ |
| announcement:write | ✅ | ✅ | — | — |
| user:read / user:write | ✅ | — | — | — |
| role:read / role:write | ✅ | — | — | — |
| audit:read | ✅ | — | — | — |
| approval_flow:read / write | ✅ | — | — | — |
| update:read / write | ✅ | — | — | — |

> 说明：审批权限点（leave:approve / attendance_correction:approve）为**可配置**，admin 可在角色管理勾选任意角色获得；补卡一级审批默认按部门经理指派（manager 步骤），若需「按权限审批」类型则由 admin 勾给对应角色。角色权限**仅在首次配置时写入默认值，重启不还原 admin 的增删**。

### 4.3 双重权限控制（接口级 + 数据级）

1. **接口级（中间件）**：`RequirePermission(db, "employee:write")` 校验角色-权限关联，admin 也走权限点校验（无业务权限），无权限返回 403。
2. **数据级（service 层行级过滤）**：
   - `employee` 角色：`GET /employees` 无权限；`GET /employees/me` 仅本人档案；考勤/补卡列表**强制按本人过滤**（传 employee_id 参数也被忽略）
   - `manager` 角色：`GET /leaves` 仅本部门员工 + 待自己审批；`GET /salaries` 无权限
   - `employee` 角色：`GET /salaries` 仅本人记录（service 强制过滤 employee_id）
   - 审批操作：非任务审批人 / 非权限池成员调用 approve 返回 403「无权限审批该请假申请」
   - 手工补录考勤：service 层校验角色必须 hr/admin（防员工借打卡权限越权补录）
   - 数据范围（ResolveScope）：hr 绑定负责部门后列表仅见该部门及子部门数据；**当前所有 hr 账号 dept_scope_id=NULL（全公司口径）**，机制保留待复用

### 4.4 认证流程

```
GET /auth/captcha → 图形验证码（base64Captcha，登录必填）
登录 POST /auth/login（username + password + captcha_id + captcha_code）
    → 验证码校验（一次即消费）→ bcrypt 校验 → 签发 access+refresh 双 token
    → claims: user_id / username / role_code；/auth/me 返回 permissions 列表
请求 → 中间件 JWTAuth 解析 Bearer token → 上下文写入身份
     → 路由级 RequirePermission 校验角色权限点
前端 → axios 拦截器自动注入 Authorization 头；access 过期用 refresh 滚动续期
     → 401 时清除 token 跳转 /login（仅 401 触发，避免误跳登录）
```

---

## 5. 业务规则要点

| 模块 | 规则 |
|------|------|
| 员工 | 工号自动生成 EMP+4位序号（pg_advisory_xact_lock 防并发）；可联动创建登录账号（默认密码 123456，拼音用户名） |
| 考勤 | 9:00 后打卡=迟到；18:00 前下班=早退；打卡后状态自动计算；重复打卡 400 |
| 请假 | 审批人按配置的审批流解析（manager 递归向上查父部门经理 / role 角色池 / position 岗位池 / permission 权限池）；无审批人自动通过；余额不足 400；终审 approve 事务内原子扣减余额；年假 10/病假 5/事假 3/婚假 3/产假 98/其他 5 |
| 审批流 | 通用引擎：approval_flows/steps/instances/tasks 四表；admin 可配 1~5 级；配置变更仅影响新申请，进行中不受影响；默认请假=补卡=两级（部门经理→HR 角色池） |
| 审批权限 | 审批权由权限点 leave:approve / attendance_correction:approve 控制，admin 在角色管理勾选；角色权限首次配置后重启不还原 |
| 通知 | 审批流转/结果实时站内通知；`GET /notifications/recent` 按 id 倒序返回未读供增量检测；Windows/Android 客户端据此弹系统通知（30s 轮询 + 前台 LocalNotifications / 后台 WorkManager 15min） |
| 薪酬 | 实发 = 基本 + 津贴 + 奖金 − 扣款（自动计算）；同人同月唯一；draft→confirmed→paid |
| 招聘 | 候选人状态机 applied→interview→offer→hired/rejected |
| 公告 | 置顶/草稿/发布；已读记录 announcement_reads（单条/全部已读），进入公告页自动全部已读 |
| 更新 | 客户端启动 5s 首检 + 30~60min 随机轮询 `update/check`；更新包存服务器 /opt/hrms/updates/；win 1.0.5 / android 1.0.4 |
| 通用 | 全部软删除；分页 page/page_size（默认 1/20，上限 100）；统一响应 {code,message,data} |

---

## 6. 部署结构

### 6.1 本地仓库

```
HRMS/
├── start.bat / start.sh     # 一键启动（前端+后端+浏览器）
├── backend/                 # Go 后端（:8080）
│   ├── cmd/server/          # 入口
│   └── internal/            # config/database/models/repository/service/handler/middleware/router/response
├── frontend/                # React 前端（:5173，/api 代理到 :8080）
│   ├── src/                 # api/components/hooks/layouts/pages/router/store/utils
│   └── android/             # Capacitor Android 工程（远程加载线上前端）
├── desktop/                 # Electron Windows 桌面端（electron/main.js、updater.js、notify.js）
├── prompts/                 # Claude Code 批次提示词（开发过程产物）
├── ARCHITECTURE.md          # 概要架构
├── DESIGN.md                # 本文档
├── PROGRESS.md              # 构建进度
├── TESTING.md               # 测试流程
└── README.md                # 使用说明
```

### 6.2 生产服务器（阿里云 ECS 115.29.197.130）

```
Ubuntu 24.04
├── PostgreSQL 18            # 低内存调优配置
├── /opt/hrms/hrms           # Go 后端（systemd hrms.service，:8080）
├── /opt/hrms/uploads/       # 上传文件
├── /opt/hrms/updates/       # 更新包（win exe / android apk，Nginx /updates/ 静态下载）
├── Nginx                    # 80 端口反代 + SPA 回退
└── UFW                      # 仅放行 22/80/443
```
