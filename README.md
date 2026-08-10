# HRMS 人力资源管理系统

前后端分离的人力资源管理系统（HRMS），覆盖组织人事、考勤、请假审批流、薪酬、招聘与统计仪表盘等核心模块。前端使用 React 18 + TypeScript + Vite + TailwindCSS + daisyUI，后端使用 Go 1.25 + Gin + GORM v2 + PostgreSQL，基于 JWT + RBAC 实现统一认证与细粒度权限控制。

系统已上线并支持**三端形态**：
- **网页版**：浏览器访问线上服务器（移动端已全面适配，抽屉导航/表格横滑/响应式断点）
- **Windows 桌面端**：Electron 封装（loadURL 加载线上前端），支持系统通知与自动更新
- **Android 端**：Capacitor APK（远程加载模式，WebView 指向线上前端），前台通知 + 后台 WorkManager 轮询

核心亮点：**可配置审批流**（admin 在「审批流程」页配置请假/补卡的级别与审批人类型）、**系统级推送通知**（Windows 通知中心 / Android 系统通知）、**移动端全面适配**。

## 功能清单

- **认证与权限**：图形验证码登录 / JWT 双 Token（access+refresh 滚动续期）/ 登录失败锁定 / 修改密码 / RBAC 权限点控制（30 个权限点，admin / hr / manager / employee 四角色 + 自定义角色与权限勾选）
- **组织架构**：部门树形管理（负责人）、岗位管理（部门 / 级别）
- **员工档案**：员工增删改查、自动生成工号（`EMP0001`）、可同时创建登录账号、头像上传、Excel 花名册导出
- **考勤管理**：员工自助打卡（自动判定迟到/早退/正常）、HR 手工补录、补卡申请审批流、考勤月报（出勤/迟到/早退/旷工/请假汇总）
- **可配置审批流**：通用审批流引擎，请假/补卡统一走 `approval_flows/steps/instances/tasks` 四表；admin 可配置审批级别（1~5 级）与每级审批人类型（部门经理 / 指定角色 / 指定岗位 / 按权限审批），默认两级（部门经理 → HR 角色池终审）
- **请假审批**：提交请假（年假/病假/事假/产假/婚假/其他）→ 按配置逐级流转 → 通过/驳回，自动校验并扣减假期余额；**审批通过自动联动考勤标记请假**
- **薪酬管理**：按员工+月份生成工资单、**按月批量生成**、实发工资自动计算、`draft → confirmed → paid` 状态机、员工自助查看工资条、Excel 工资表导出
- **招聘管理**：招聘职位发布与候选人管理（简历跟踪、面试状态流转）
- **绩效管理**：考核周期（2026-H1/H2）、评分自动映射 S/A/B/C/D 评级、`draft → submitted → approved` 状态流
- **公告管理**：公告发布/置顶/搜索，全员可见，已读联动，仪表盘展示最新公告
- **站内通知 + 系统推送**：请假/补卡提交与审批结果实时站内通知（顶栏铃铛未读提醒）；Windows/Android 客户端支持**系统级通知**（增量检测，30s 轮询）
- **角标提醒**：待审批数 / 未读公告 / 未读通知导航角标（30s + 窗口聚焦轮询）
- **统一待办中心**：/approvals 聚合待我审批与我处理的审批任务，审批进度时间线可追踪
- **系统管理**：用户管理（角色分配/重置密码/禁用/数据范围）、角色管理（权限分组勾选）、审批流程配置（审批级别/审批人）、操作审计日志（登录/增删改全留痕）
- **数据范围权限**：HR 可绑定负责部门，仅可见该部门及子部门数据（ResolveScope 机制保留；当前所有 hr 账号 dept_scope_id 为空 = 全公司口径）
- **统计仪表盘**：在职人数、本月入职/离职、出勤率、待审批请假/薪酬、招聘/薪酬趋势、部门分布、最新公告
- **暗色模式**：浅色/深色一键切换并持久化

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 + Gin + GORM v2 |
| 数据库 | PostgreSQL 18 |
| 认证 | JWT（golang-jwt/jwt/v5）+ bcrypt |
| 前端 | React 18 + TypeScript + Vite |
| 样式 | TailwindCSS + daisyUI |
| 状态管理 | zustand |
| 请求 | Axios（拦截器注入 Bearer Token） |
| Windows 桌面端 | Electron + electron-builder |
| Android 端 | Capacitor + WorkManager + @capacitor/local-notifications |

## 目录结构

```
HRMS/
├── backend/                    # Go 后端
│   ├── cmd/server/main.go      # 入口
│   ├── internal/
│   │   ├── config/             # 配置加载 (.env)
│   │   ├── database/           # GORM 连接 + AutoMigrate + Seed
│   │   ├── models/             # GORM 模型（24 张表）
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务逻辑层
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # JWT / RBAC 中间件
│   │   ├── router/             # 路由注册
│   │   └── response/           # 统一响应封装
│   └── .env.example
├── frontend/                   # React 前端
│   ├── src/
│   │   ├── api/                # API 调用封装
│   │   ├── components/         # 通用组件
│   │   ├── pages/              # 页面组件
│   │   ├── router/             # 路由配置
│   │   ├── store/              # zustand 全局状态
│   │   ├── hooks/              # usePerm / useSystemNotify
│   │   └── layouts/            # 布局组件（含移动端抽屉侧边栏）
│   └── android/                # Capacitor Android 工程（远程加载模式）
├── desktop/                    # Electron Windows 桌面端
│   └── electron/               # main / preload / updater / notify
├── prompts/                    # Claude Code 批次提示词（开发过程产物）
├── ARCHITECTURE.md             # 架构设计文档
├── DESIGN.md                   # 系统设计文档
├── TESTING.md                  # 测试流程指南
├── PROGRESS.md                 # 构建进度报告
└── start.bat / start.sh        # 本地一键启动
```

## 环境要求

| 依赖 | 版本 |
|------|------|
| Go | 1.25+ |
| Node.js | 20+ |
| PostgreSQL | 18 |

## 快速开始（本地开发）

> 生产环境已部署至线上服务器，业务使用请直接访问服务器（见「部署与分发」）。以下为本地开发/联调方式。

### 1. 创建数据库

```sql
CREATE DATABASE hrms;
```

### 2. 启动后端

```bash
cd backend
cp .env.example .env   # 修改 DB_PASSWORD 为你的数据库密码
go run ./cmd/server
```

后端默认监听 `http://localhost:8080`，首次启动自动建表（AutoMigrate）并写入种子数据（内置角色、权限、账号、部门、岗位、示例员工、审批流程定义）。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认监听 `http://localhost:5173`，Vite 已将 `/api` 代理到后端 `http://localhost:8080`。

### 4. （可选）桌面端 / Android 本地构建

Windows 桌面端与 Android 包默认加载**线上前端**，本地开发一般无需构建：

```bash
cd desktop
npm install && npm run dist        # 构建 Windows 安装包（electron-builder）
# Android 见 frontend/android（Capacitor 工程，远程加载 http://115.29.197.130）
```

## 部署与分发

- **服务器**：阿里云 ECS `http://115.29.197.130`（Ubuntu 24.04，PostgreSQL + Go systemd + Nginx 反代，UFW 仅开放 22/80/443）
- **网页版**：浏览器直接访问线上地址即可（移动端已适配）
- **Windows 桌面端**：`hrms-desktop Setup` 安装包（Electron 封装，加载线上前端，支持系统通知）
- **Android 端**：`HRMS.apk`（Capacitor 远程加载模式，WebView 加载线上前端）
- **更新机制**：客户端启动后周期调用 `GET /api/v1/update/check?platform=win|android&build=N` 增量检测，有新版则下载更新包并引导安装；更新包存放于服务器 `/opt/hrms/updates/`，由 admin 在系统管理中登记 `app_versions` 记录
- **当前版本**：Windows 1.0.5 / Android 1.0.4

## 内置账号

| 账号 | 密码 | 角色 | 说明 |
|------|------|------|------|
| admin | admin123 | admin | 系统管理员：用户/角色/审计管理 + 公告 + 审批流程配置（无业务权限） |
| hr | hr123 | hr | 人事专员：全公司人事业务（员工/考勤/薪酬/招聘/绩效/公告 + 审批终审） |
| zhangwei | 123456 | manager | 部门经理（张伟）：本部门请假/补卡一级审批 |
| liuyang | 123456 | employee | 普通员工（刘洋）：自助服务（打卡/请假/工资条/绩效） |

## API 概览

统一前缀 `/api/v1`，除登录、验证码、健康检查与更新检查外均需请求头 `Authorization: Bearer <token>`。统一响应格式：`{"code": 0, "message": "ok", "data": ...}`，分页结构 `{list, total, page, page_size}`。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/health` | 健康检查（公开） |
| GET | `/api/v1/auth/captcha` | 图形验证码（公开） |
| POST | `/api/v1/auth/login` | 登录（用户名 + 密码 + 验证码） |
| POST | `/api/v1/auth/refresh` | 刷新 access token |
| GET | `/api/v1/auth/me` | 当前登录用户信息（含 permissions 列表） |
| PUT | `/api/v1/auth/password` | 修改密码 |

### 组织架构

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/departments` | 部门树 / 列表 |
| POST | `/api/v1/departments` | 创建部门 |
| PUT | `/api/v1/departments/:id` | 更新部门 |
| DELETE | `/api/v1/departments/:id` | 删除部门 |
| GET | `/api/v1/positions` | 岗位分页列表 |
| POST | `/api/v1/positions` | 创建岗位 |
| PUT | `/api/v1/positions/:id` | 更新岗位 |
| DELETE | `/api/v1/positions/:id` | 删除岗位 |

### 员工

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/employees` | 员工分页列表（keyword/部门/岗位/状态筛选） |
| POST | `/api/v1/employees` | 创建员工（可同时创建登录账号） |
| GET | `/api/v1/employees/:id` | 员工详情 |
| PUT | `/api/v1/employees/:id` | 更新员工 |
| DELETE | `/api/v1/employees/:id` | 删除员工（软删除） |
| GET | `/api/v1/employees/me` | 我的档案（自助） |
| PUT | `/api/v1/employees/me/avatar` | 更新我的头像（自助） |
| GET | `/api/v1/employees/export` | 员工花名册导出（Excel） |

### 考勤

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/attendance` | 考勤分页列表 |
| GET | `/api/v1/attendance/report` | 考勤月报 |
| POST | `/api/v1/attendance/check-in` | 上班打卡 |
| POST | `/api/v1/attendance/check-out` | 下班打卡 |
| POST | `/api/v1/attendance/manual` | 手工补录（HR） |
| PUT | `/api/v1/attendance/:id` | 修改考勤记录 |
| DELETE | `/api/v1/attendance/:id` | 删除考勤记录 |
| POST | `/api/v1/attendance/corrections` | 提交补卡申请 |
| GET | `/api/v1/attendance/corrections` | 补卡申请列表 |
| GET | `/api/v1/attendance/corrections/mine` | 我的补卡申请（自助） |
| PUT | `/api/v1/attendance/corrections/:id/cancel` | 取消补卡申请 |

### 请假 + 通用审批流

请假申请与审批已迁移至**通用审批流引擎**（多级、按 admin 配置推进）。提交/列表走 `/leaves`，审批动作走 `/approvals`：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/leaves` | 提交请假申请 |
| GET | `/api/v1/leaves` | 请假分页列表 |
| GET | `/api/v1/leaves/mine` | 我的申请列表（自助） |
| GET | `/api/v1/leaves/balance` | 我的假期余额（自助） |
| PUT | `/api/v1/leaves/:id/cancel` | 取消申请 |
| GET | `/api/v1/approvals/mine` | 待我审批 / 我处理的审批任务 |
| GET | `/api/v1/approvals/biz` | 按业务类型（leave/attendance_correction）查询实例 |
| PUT | `/api/v1/approvals/tasks/:taskId/approve` | 审批通过 |
| PUT | `/api/v1/approvals/tasks/:taskId/reject` | 审批驳回 |
| GET | `/api/v1/approvals/:instanceId/tasks` | 审批实例任务列表（进度时间线） |

### 审批流程配置（admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/approval-flows` | 查看请假/补卡审批流程配置 |
| PUT | `/api/v1/approval-flows/:code` | 全量保存流程步骤（1~5 级，类型/角色/岗位/权限） |

### 薪酬

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/salaries` | 工资单分页列表 |
| POST | `/api/v1/salaries` | 创建工资单（自动计算实发） |
| POST | `/api/v1/salaries/batch-generate` | 按月批量生成 |
| PUT | `/api/v1/salaries/:id` | 更新工资单 |
| PUT | `/api/v1/salaries/:id/confirm` | 确认工资单 |
| PUT | `/api/v1/salaries/:id/pay` | 发放工资 |
| GET | `/api/v1/salaries/mine` | 我的工资条（自助） |
| GET | `/api/v1/salaries/export` | 工资表导出（Excel） |

### 招聘

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/recruitment/jobs` | 招聘职位列表 |
| POST | `/api/v1/recruitment/jobs` | 发布招聘职位 |
| PUT | `/api/v1/recruitment/jobs/:id` | 更新职位 |
| DELETE | `/api/v1/recruitment/jobs/:id` | 删除职位 |
| GET | `/api/v1/recruitment/candidates` | 候选人列表 |
| POST | `/api/v1/recruitment/candidates` | 添加候选人 |
| PUT | `/api/v1/recruitment/candidates/:id` | 更新候选人（状态流转） |
| DELETE | `/api/v1/recruitment/candidates/:id` | 删除候选人 |

### 公告 / 绩效

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/announcements` | 公告列表（含 is_read） |
| POST | `/api/v1/announcements` | 发布公告 |
| PUT | `/api/v1/announcements/:id` | 更新公告 |
| DELETE | `/api/v1/announcements/:id` | 删除公告 |
| PUT | `/api/v1/announcements/:id/read` | 标记公告已读 |
| PUT | `/api/v1/announcements/read-all` | 全部标记已读 |
| GET | `/api/v1/performance` | 绩效分页列表 |
| GET | `/api/v1/performance/mine` | 我的绩效（自助） |
| POST | `/api/v1/performance` | 创建绩效 |
| PUT | `/api/v1/performance/:id/submit` | 提交绩效 |
| PUT | `/api/v1/performance/:id/approve` | 批准绩效 |
| PUT | `/api/v1/performance/:id` | 更新绩效 |
| DELETE | `/api/v1/performance/:id` | 删除绩效 |

### 通知 / 角标

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/notifications` | 站内通知列表 |
| GET | `/api/v1/notifications/recent` | 未读通知增量检测（按 id 倒序，客户端系统通知用） |
| GET | `/api/v1/notifications/unread-count` | 未读数 |
| PUT | `/api/v1/notifications/:id/read` | 标记已读 |
| PUT | `/api/v1/notifications/read-all` | 全部标记已读 |
| GET | `/api/v1/badges` | 角标汇总（pending_approvals / unread_announcements / unread_notifications） |

### 系统管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users` | 用户列表 |
| POST | `/api/v1/users` | 创建用户（角色分配） |
| PUT | `/api/v1/users/:id` | 更新用户（角色/状态/数据范围） |
| PUT | `/api/v1/users/:id/password` | 重置密码 |
| DELETE | `/api/v1/users/:id` | 删除用户 |
| GET | `/api/v1/users/roles` | 角色下拉（自助） |
| GET | `/api/v1/roles` | 角色列表 |
| GET | `/api/v1/roles/permissions` | 权限点列表 |
| POST | `/api/v1/roles` | 创建角色 |
| PUT | `/api/v1/roles/:id` | 更新角色（权限勾选） |
| DELETE | `/api/v1/roles/:id` | 删除角色 |
| GET | `/api/v1/audit-logs` | 操作审计日志 |

### 仪表盘 / 更新 / 上传

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/dashboard/stats` | 统计指标（在职/入职/离职/待审批/趋势/分布） |
| GET | `/api/v1/update/check` | 客户端版本检查（`?platform=win\|android&build=N`，公开） |
| GET | `/api/v1/update/latest` | 最新版本信息（公开） |
| POST | `/api/v1/update/versions` | 登记更新包（admin） |
| GET | `/api/v1/update/versions` | 更新包列表（admin） |
| POST | `/api/v1/upload` | 图片上传（≤5MB） |
| GET | `/uploads/*` | 静态文件（头像等） |

## 验证与检查

```bash
# 后端：构建 + 静态检查 + 自动化测试（65 例，需本机 hrms_test 库）
cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1

# 前端
cd frontend && npm run build
```

## 一键启动（本地）

双击 `start.bat`（或 git-bash 执行 `./start.sh`）：自动检查环境 → 启动后端(8080) + 前端(5173) → 打开浏览器。停止：关闭两个服务窗口。

## 测试账号

见上方「内置账号」表。测试要点：admin 走系统管理（用户/角色/审计/公告/审批流程配置）；hr 验证全公司人事业务（考勤补录/薪酬/招聘/审批终审）；zhangwei 验证请假与补卡一级审批；liuyang 验证自助打卡/请假/工资条/头像/暗色模式。完整测试流程（含三端）见 `TESTING.md`，系统设计见 `DESIGN.md`。
