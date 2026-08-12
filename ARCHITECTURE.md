# HRMS 人力资源管理系统 — 架构设计文档

> 架构师：Hermes Agent（指挥官）｜ 实现：Claude Code（deepseek-flash）
> 更新日期：2026-08-11
> 参考项目：github.com/pbrong/hrms（Go 单体 HRMS）、github.com/frappe/hrms（模块化开源 HRMS 的模块划分思路）

## 1. 总体架构

前后端分离单体架构（Monolith-first，部署简单），未来可按模块拆分微服务。系统支持**三端形态**——浏览器网页版、Windows 桌面端（Electron）、Android 端（Capacitor WebView），三者均访问**同一服务器**；生产采用 **Docker 容器化**，一套代码服务多家公司（各公司独立数据库 + 独立后端容器 + 独立入口）。

```
┌──────────────────────────┬──────────────────────────┬─────────────────────────────┐
│   浏览器 (React SPA)      │  Windows 桌面端 (Electron)│   Android (Capacitor WebView) │
│  React18 + TS + Tailwind │  loadURL 线上前端         │  远程加载模式（server.url）   │
│  React Router+Axios+zustand│ 系统通知 notify IPC     │  LocalNotifications +  WorkManager│
│  移动端抽屉/横滑适配       │  自动更新 updater.js     │  TokenStore 桥接 + 原生更新    │
└──────────────────────┬───┴──────────────────────┬───┴─────────────────────────────┘
                       │        REST /api/v1 (JSON, JWT Bearer)
┌──────────────────────▼────────────────────────────────────────────────────────────┐
│                        Nginx 反代（Ubuntu 24.04，生产）                            │
│   80：IP 直连默认服务 + 域名块 ｜ 443/8443：Cloudflare 回源预留（域名备案后启用）     │
│   /uploads 静态 + /updates 更新包 + SPA 回退                                        │
└──────────────────────┬────────────────────────────────────────────────────────────┘
              ┌────────┴────────────┬─────────────────────┐
              │                     │                     │
   ┌──────────▼──────────┐  ┌───────▼──────────┐  ┌───────▼─────────┐
   │  hrms-backend 容器   │  │ hrms-backend-    │  │  新公司容器      │
   │  :8080（主公司）     │  │ testco :8081     │  │  :8082 起        │
   │  handler→service→    │  │ handler→service→ │  │ new-company.sh   │
   │  repository→GORM     │  │ repository→GORM  │  │                  │
   └──────────┬──────────┘  └───────┬──────────┘  └───────┬─────────┘
              │                     │                     │
   ┌──────────▼─────────────────────▼─────────────────────▼─────────┐
   │         hrms-db 容器（PostgreSQL 16）                           │
   │         hrms(主)  hrms_testco  hrms_<公司标识> …（24 张表）     │
   └────────────────────────────────────────────────────────────────┘
```

每个后端容器内部均为 Gin HTTP 服务分层：middleware（CORS → Logger → JWT Auth → RBAC）→ handler → service → repository → GORM；前端生产构建走相对路径 `/api/v1`，由 Nginx 按域名/端口反代到对应公司后端。

## 2. 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| 后端 | Go 1.25 + Gin | HTTP 框架 |
| ORM | GORM v2 | PostgreSQL 驱动 |
| 数据库 | PostgreSQL（开发 18 / 生产 16 容器） | 主库 hrms + 多公司库 |
| 认证 | JWT (golang-jwt/jwt/v5) + bcrypt | access+refresh 双 token |
| 配置 | godotenv + 环境变量 | .env 管理密钥 |
| 日志 | 标准库 log/slog | 结构化日志 |
| 前端 | React 18 + TypeScript + Vite | 构建工具 |
| 样式 | TailwindCSS + daisyUI | 快速成型 |
| 状态 | zustand | 轻量全局状态 |
| 请求 | Axios（拦截器注入 JWT） | API 客户端 |
| Windows 桌面端 | Electron + electron-builder | loadURL 线上前端 + 系统通知 + 自动更新 |
| Android 端 | Capacitor + WorkManager + @capacitor/local-notifications | 远程加载 + 前台/后台系统通知 |
| 部署 | Docker / docker-compose | 多阶段镜像 hrms-backend:latest + postgres:16-alpine，多公司多库 |
| 服务器 | Ubuntu 24.04 + Nginx + UFW + 阿里云安全组 | 生产部署 |
| 边缘（预留） | Cloudflare 代理回源 | 443/8443 回源，域名备案后启用 |

## 3. 目录结构

```
backend/                    # Go 后端
├── cmd/server/main.go          # 入口
├── internal/
│   ├── config/config.go        # 配置加载 (.env)
│   ├── database/database.go    # GORM 连接 + AutoMigrate(24表) + Seed
│   ├── models/                 # GORM 模型（每模块一文件）
│   ├── repository/             # 数据访问层
│   ├── service/                # 业务逻辑层（含通用审批流引擎）
│   ├── handler/                # HTTP 处理器（每模块一文件）
│   ├── middleware/             # JWT / RBAC / CORS / 日志
│   ├── router/router.go        # 路由注册
│   └── response/response.go    # 统一响应 {code,message,data} + 分页
├── .env.example
└── go.mod

frontend/                   # React 前端
├── src/                        # api/components/hooks/layouts/pages/router/store/utils
└── android/                    # Capacitor Android 工程（远程加载线上前端）

desktop/                    # Electron Windows 桌面端
└── electron/                   # main.js / preload.js / updater.js / notify.js
```

## 4. 数据库设计（核心表）

### 组织与权限
- `users` — 系统账号（username, password_hash, email, status, employee_id, role_id, dept_scope_id）
- `roles` — 角色（admin / hr / manager / employee，可自定义）
- `permissions` — 权限点（30 个，如 employee:read、leave:approve、approval_flow:write）
- `role_permissions` — 角色-权限关联
- `departments` — 部门（name, parent_id 树形, manager_id, sort）
- `positions` — 岗位（name, code, department_id, level）

### 人事核心
- `employees` — 员工档案（emp_no, name, gender, birth_date, id_card, phone, email, department_id, position_id, hire_date, status, address, avatar）
- `attendances` — 考勤（employee_id, work_date, check_in, check_out, status）
- `leave_requests` — 请假申请（employee_id, leave_type, start_date, end_date, days, reason, status）
- `leave_balances` — 假期余额（employee_id, leave_type, year, total, used）
- `salary_records` — 工资单（employee_id, month, base_salary, allowance, bonus, deduction, net_salary, status）
- `job_openings` — 招聘职位（title, department_id, position_id, headcount, requirement, status）
- `candidates` — 候选人（name, phone, email, job_opening_id, status, interview_date, remark）
- `performance_reviews` — 绩效（employee_id, period, score, rating S/A/B/C/D, status）

### 审批流引擎（通用，请假/补卡共用）
- `approval_flows` — 流程定义（code=leave|attendance_correction, name, status）
- `approval_steps` — 步骤（flow_id, step_order 1~5, approver_type: manager|role|position|permission, approver_role_id, approver_position_id）
- `approval_instances` — 实例（biz_type, biz_id, current_step, status, applicant_id）
- `approval_tasks` — 任务（instance_id, step_order, approver_id, status, comment, acted_at）

### 通知 / 更新
- `notifications` — 站内通知（receiver_id, title, content, is_read）
- `announcements` / `announcement_reads` — 公告 + 已读
- `app_versions` — 客户端版本（platform, version, build_number, url, force_update）

## 5. 核心业务规则

1. **RBAC（定稿）**：admin=系统管理（用户/角色/审计/公告/审批流程配置，**无业务权限**）；hr=全公司人事业务（含审批终审）；manager=部门审批；employee=自助。审批权由权限点 `leave:approve` / `attendance_correction:approve` 控制，admin 可勾选任意角色。
2. **通用审批流引擎**：请假/补卡统一走 `approval_flows/steps/instances/tasks` 四表，1~5 级按 admin 配置推进；审批人解析 = 部门经理递归向上查父部门 / 角色池 / 岗位池 / 权限池；未指派任务按审批权限点匹配。
3. **考勤状态**：normal / late / early_leave / absent，由打卡时间自动计算；请假审批通过联动标记 leave。
4. **薪酬计算**：net = base + allowance + bonus - deduction，按月生成，HR 确认后发放。
5. **员工编号**：emp_no 唯一，格式 `EMP` + 4 位序号，创建时自动生成。
6. **请假天数校验**：不能超过该类型剩余假期余额。
7. **seed 权限行为**：角色权限仅在角色「从未配置过任何权限」时初始化一次；admin 修改后**重启不还原**。
8. **系统通知**：后端 `GET /notifications/recent` 增量检测；Windows 走 Electron notify IPC，Android 前台 LocalNotifications + 后台 WorkManager 轮询（TokenStore 桥接）。

## 6. API 约定

- 前缀 `/api/v1`，JWT Bearer 认证（登录/验证码/更新检查除外）
- 统一响应：`{"code": 0, "message": "ok", "data": ...}`；错误 code 非 0
- 分页：请求 `?page=1&page_size=20`，响应 `{list, total, page, page_size}`
- 时间格式 RFC3339，日期格式 `YYYY-MM-DD`

## 7. 里程碑

| 里程碑 | 内容 | 状态 |
|--------|------|------|
| M1 | 后端脚手架：配置/DB/迁移/seed/健康检查 | ✅ 已完成 |
| M2 | 模型 + RBAC + 认证（登录/JWT/中间件） | ✅ 已完成 |
| M3 | 员工/部门/岗位 CRUD + 文件上传 | ✅ 已完成 |
| M4 | 考勤 + 请假审批流 | ✅ 已完成 |
| M5 | 薪酬 + 招聘 + 统计仪表盘 | ✅ 已完成 |
| M6 | 前端脚手架 + 登录/布局 | ✅ 已完成 |
| M7 | 前端各模块页面 | ✅ 已完成 |
| M8 | 联调 + 构建验证 + 文档 | ✅ 已完成 |
| M9~M45 | **十轮增强**（2026-08-04 ~ 08-10）：P0-P2 功能补全（用户/角色/审计、Excel、测试、安全加固）、通用审批流引擎、权限重构（admin/hr 职责分离）、可配置审批流（多级+角色/岗位/权限池）、审批链修复（递归经理）、移动端全面适配（19 页）、服务器部署上线、Windows/Android 客户端、自动更新、系统级推送通知 | ✅ 已完成 |
| M46~M50 | **部署架构升级**（2026-08-11）：Docker 基础化（Dockerfile 多阶段 + compose + 数据迁移 + systemd 保留可回滚）、前端相对路径 + Nginx 双块 + 多公司模板（new-company.sh/remove-company.sh）、Cloudflare 回源通道（8443 + 自签证书 + 安全组）、客户端域名切换（win 1.0.7 / android 1.0.5）、回退 IP 直连（Cloudflare 525 待 ICP 备案后处理） | ✅ 已完成 |

> 详细批次记录见 `PROGRESS.md`（M1~M50 全量时间线 + 每轮增强说明）。
