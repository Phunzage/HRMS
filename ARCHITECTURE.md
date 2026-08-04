# HRMS 人力资源管理系统 — 架构设计文档

> 架构师：Hermes Agent（指挥官）｜ 实现：Claude Code（deepseek-flash）
> 日期：2026-08-04
> 参考项目：github.com/pbrong/hrms（Go 单体 HRMS）、github.com/frappe/hrms（模块化开源 HRMS 的模块划分思路）

## 1. 总体架构

前后端分离单体架构（Monolith-first，部署简单、2 小时内可交付），未来可按模块拆分微服务。

```
┌─────────────────────────────────────────────────┐
│                   React SPA (Vite)              │
│  React18 + TypeScript + TailwindCSS + daisyUI   │
│  React Router + Axios + zustand                 │
└──────────────────────┬──────────────────────────┘
                       │ REST /api/v1 (JSON, JWT Bearer)
┌──────────────────────▼──────────────────────────┐
│              Gin HTTP 服务 (Go 1.25)             │
│  middleware: CORS → Logger → JWT Auth → RBAC    │
│  handler 层 → service 层 → repository 层 → GORM  │
└──────────────────────┬──────────────────────────┘
                       │ PostgreSQL 18 (hrms 库)
└──────────────────────▼──────────────────────────┘
```

## 2. 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| 后端 | Go 1.25 + Gin | HTTP 框架 |
| ORM | GORM v2 | PostgreSQL 驱动 |
| 认证 | JWT (golang-jwt/jwt/v5) + bcrypt | 无状态令牌 |
| 配置 | godotenv + 环境变量 | .env 管理密钥 |
| 日志 | 标准库 log/slog | 结构化日志 |
| 前端 | React 18 + TypeScript + Vite | 构建工具 |
| 样式 | TailwindCSS + daisyUI | 快速成型 |
| 状态 | zustand | 轻量全局状态 |
| 请求 | Axios（拦截器注入 JWT） | API 客户端 |

## 3. 后端目录结构

```
backend/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── config/config.go        # 配置加载 (.env)
│   ├── database/database.go    # GORM 连接 + AutoMigrate + Seed
│   ├── models/                 # GORM 模型（每模块一文件）
│   ├── repository/             # 数据访问层
│   ├── service/                # 业务逻辑层
│   ├── handler/                # HTTP 处理器（每模块一文件）
│   ├── middleware/             # JWT / RBAC / CORS / 日志
│   ├── router/router.go        # 路由注册
│   └── response/response.go    # 统一响应 {code,message,data} + 分页
├── .env.example
└── go.mod
```

## 4. 数据库设计（核心表）

### 组织与权限
- `users` — 系统账号（username, password_hash, email, status, employee_id, role_id）
- `roles` — 角色（admin / hr / manager / employee）
- `permissions` — 权限点（如 employee:read）
- `role_permissions` — 角色-权限关联
- `departments` — 部门（name, parent_id 树形, manager_id, sort）
- `positions` — 岗位（name, code, department_id, level）

### 人事核心
- `employees` — 员工档案（emp_no, name, gender, birth_date, id_card, phone, email, department_id, position_id, hire_date, status, address, avatar）
- `attendance` — 考勤（employee_id, work_date, check_in, check_out, status）
- `leave_requests` — 请假申请（employee_id, leave_type, start_date, end_date, days, reason, status, approver_id, approved_at）
- `leave_balances` — 假期余额（employee_id, leave_type, year, total, used）
- `salary_records` — 工资单（employee_id, month, base_salary, allowance, bonus, deduction, net_salary, status）
- `job_openings` — 招聘职位（title, department_id, position_id, headcount, requirement, status）
- `candidates` — 候选人（name, phone, email, job_opening_id, status, interview_date, remark）

## 5. 核心业务规则

1. **RBAC**：三种内置角色 admin(全部权限)、hr(人事/考勤/薪酬)、manager(本部门审批)、employee(自助)。
2. **审批流**：请假申请 → 直属经理审批 → 通过/驳回（状态机 pending → approved/rejected）。
3. **考勤状态**：normal / late / early_leave / absent，由打卡时间自动计算。
4. **薪酬计算**：net = base + allowance + bonus - deduction，按月生成，HR 确认后发放。
5. **员工编号**：emp_no 唯一，格式 `EMP` + 4 位序号，创建时自动生成。
6. **请假天数校验**：不能超过该类型剩余假期余额。

## 6. API 约定

- 前缀 `/api/v1`，JWT Bearer 认证（登录/注册除外）
- 统一响应：`{"code": 0, "message": "ok", "data": ...}`；错误 code 非 0
- 分页：请求 `?page=1&page_size=20`，响应 `{list, total, page, page_size}`
- 时间格式 RFC3339，日期格式 `YYYY-MM-DD`

## 7. 里程碑（两小时冲刺）

| 里程碑 | 内容 | 状态 |
|--------|------|------|
| M1 | 后端脚手架：配置/DB/迁移/seed/健康检查 | |
| M2 | 模型 + RBAC + 认证（登录/JWT/中间件） | |
| M3 | 员工/部门/岗位 CRUD + 文件上传 | |
| M4 | 考勤 + 请假审批流 | |
| M5 | 薪酬 + 招聘 + 统计仪表盘 | |
| M6 | 前端脚手架 + 登录/布局 | |
| M7 | 前端各模块页面 | |
| M8 | 联调 + 构建验证 + 文档 | |
