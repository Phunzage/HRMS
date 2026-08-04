# HRMS — Project Context for Claude Code

人力资源管理系统。你在本项目中担任**实现工程师**，架构师（Hermes）会分批向你下发任务提示词。严格遵守每批任务的指令，不要擅自扩大范围、不要改变既定架构。

## 技术栈（固定，不得更改）
- 后端：Go 1.25 + Gin + GORM v2 + PostgreSQL 18
- 前端：React 18 + TypeScript + Vite + TailwindCSS + daisyUI
- 认证：JWT (golang-jwt/jwt/v5) + bcrypt

## 数据库连接
- host: localhost:5432, db: hrms, user: postgres, password: 3143
- DSN: `postgres://postgres:3143@localhost:5432/hrms?sslmode=disable`
- 配置放 `.env`（参考 `.env.example`），**不要把密码硬编码进源码**

## 后端规范
- 分层：`handler → service → repository → gorm`，禁止 handler 直接操作 DB
- 统一响应：`{"code":0,"message":"ok","data":...}`，分页 `{list,total,page,page_size}`
- 路由前缀 `/api/v1`，JWT Bearer 认证
- 模型用 GORM AutoMigrate 建表，内置账号用 seed 写入（不要外置 SQL 脚本）
- 内置账号：admin/admin123（角色 admin），可再建 hr/manager/employee 测试账号
- 密码必须 bcrypt 哈希存储
- 用 `log/slog` 打日志，错误处理统一走响应封装

## 前端规范
- Vite + React Router；axios 实例统一封装，请求拦截器注入 `Authorization: Bearer <token>`
- 页面组件放 `src/pages/`，通用组件放 `src/components/`，API 调用放 `src/api/`
- Tailwind + daisyUI 组件库（`theme: "light"`）
- 中文界面

## 运行方式
- 后端：`cd backend && go run ./cmd/server`
- 前端：`cd frontend && npm run dev`（默认 5173，vite proxy `/api` → `localhost:8080`）
- 后端监听 8080

## 质量要求
- 每个任务完成后运行 `go build ./...`（后端）确保编译通过；前端任务运行 `npm run build` 确保构建通过
- 遇到权限问题直接用高权限模式，不要停下询问
- 保持代码整洁、注释用中文、命名清晰
