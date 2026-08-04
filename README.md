# HRMS 人力资源管理系统

前后端分离的人力资源管理系统（HRMS），覆盖组织人事、考勤、请假审批流、薪酬、招聘与统计仪表盘等核心模块。前端使用 React 18 + TypeScript + Vite + TailwindCSS + daisyUI，后端使用 Go 1.25 + Gin + GORM v2 + PostgreSQL，基于 JWT + RBAC 实现统一认证与细粒度权限控制。

## 功能清单

- **认证与权限**：JWT 登录 / 修改密码 / 四角色 RBAC 权限点控制（admin / hr / manager / employee）
- **组织架构**：部门树形管理、岗位管理（部门 / 级别）
- **员工档案**：员工增删改查、自动生成工号（`EMP0001`）、可同时创建登录账号
- **考勤管理**：员工自助打卡（自动判定迟到/早退/正常）、HR 手工补录与修改、按员工/日期/月份/状态筛选
- **请假审批流**：提交请假（年假/病假/事假/产假/婚嫁/其他）→ 直属经理审批 → 通过/驳回，自动校验并扣减假期余额
- **薪酬管理**：按员工+月份生成工资单、实发工资自动计算、`draft → confirmed → paid` 状态机、员工自助查看工资条
- **招聘管理**：招聘职位发布与候选人管理（简历跟踪、面试状态流转）
- **统计仪表盘**：在职人数、本月入职/离职、待审批请假/薪酬、招聘趋势、薪酬趋势、部门分布等指标

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

## 目录结构

```
HRMS/
├── backend/                    # Go 后端
│   ├── cmd/server/main.go      # 入口
│   ├── internal/
│   │   ├── config/             # 配置加载 (.env)
│   │   ├── database/           # GORM 连接 + AutoMigrate + Seed
│   │   ├── models/             # GORM 模型
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务逻辑层
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # JWT / RBAC 中间件
│   │   ├── router/             # 路由注册
│   │   └── response/           # 统一响应封装
│   └── .env.example
├── frontend/                   # React 前端
│   └── src/
│       ├── api/                # API 调用封装
│       ├── components/         # 通用组件
│       ├── pages/              # 页面组件
│       ├── router/             # 路由配置
│       ├── store/              # zustand 全局状态
│       └── layouts/            # 布局组件
├── ARCHITECTURE.md             # 架构设计文档
└── PROGRESS.md                 # 构建进度报告
```

## 环境要求

| 依赖 | 版本 |
|------|------|
| Go | 1.25+ |
| Node.js | 20+ |
| PostgreSQL | 18 |

## 快速开始

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

后端默认监听 `http://localhost:8080`，首次启动自动建表（AutoMigrate）并写入种子数据（内置角色、权限、账号、部门、岗位、示例员工）。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认监听 `http://localhost:5173`，Vite 已将 `/api` 代理到后端 `http://localhost:8080`。

## 内置账号

| 账号 | 密码 | 角色 | 说明 |
|------|------|------|------|
| admin | admin123 | 管理员 | 系统管理员，拥有全部权限 |
| hr | hr123 | 人事 | 人事专员，负责组织/考勤/薪酬/招聘 |
| zhangwei | 123456 | 经理 | 部门经理（张伟），负责本部门请假审批 |
| liuyang | 123456 | 员工 | 普通员工（刘洋），自助服务 |

## API 概览

统一前缀 `/api/v1`，除登录与健康检查外均需请求头 `Authorization: Bearer <token>`。统一响应格式：`{"code": 0, "message": "ok", "data": ...}`，分页结构 `{list, total, page, page_size}`。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录（用户名 + 密码） |
| GET | `/api/v1/auth/me` | 当前登录用户信息 |
| PUT | `/api/v1/auth/password` | 修改密码 |
| GET | `/api/v1/health` | 健康检查 |

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

### 考勤

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/attendance` | 考勤分页列表 |
| POST | `/api/v1/attendance/check-in` | 上班打卡 |
| POST | `/api/v1/attendance/check-out` | 下班打卡 |
| POST | `/api/v1/attendance/manual` | 手工补录（HR） |
| PUT | `/api/v1/attendance/:id` | 修改考勤记录 |
| DELETE | `/api/v1/attendance/:id` | 删除考勤记录 |

### 请假审批流

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/leaves` | 提交请假申请 |
| GET | `/api/v1/leaves` | 请假分页列表 |
| GET | `/api/v1/leaves/mine` | 我的申请列表（自助） |
| GET | `/api/v1/leaves/balance` | 我的假期余额（自助） |
| PUT | `/api/v1/leaves/:id/approve` | 审批通过 |
| PUT | `/api/v1/leaves/:id/reject` | 审批驳回 |
| PUT | `/api/v1/leaves/:id/cancel` | 取消申请 |

### 薪酬

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/salaries` | 工资单分页列表 |
| POST | `/api/v1/salaries` | 创建工资单（自动计算实发） |
| PUT | `/api/v1/salaries/:id` | 更新工资单 |
| PUT | `/api/v1/salaries/:id/confirm` | 确认工资单 |
| PUT | `/api/v1/salaries/:id/pay` | 发放工资 |
| GET | `/api/v1/salaries/mine` | 我的工资条（自助） |

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

### 仪表盘

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/dashboard/stats` | 统计指标（在职/入职/离职/待审批/趋势/分布） |

## 验证与检查

```bash
# 后端
cd backend && go build ./... && go vet ./... && gofmt -l .

# 前端
cd frontend && npm run build
```
