# 任务批次 8：联调测试 + 端到端验证 + README + 最终提交

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。全部代码已完成。本批次做**最终联调与收尾**。

## 一、端到端联调
1. 启动后端：`cd backend && go run ./cmd/server`（后台）
2. 启动前端：`cd frontend && npm run dev`（后台）
3. 用 curl 走一遍完整业务流程（记录每步结果）：
   - admin/admin123 登录拿 token
   - 用 token 依次调：/dashboard/stats、/employees、/departments、/positions、/attendance、/leaves、/salaries、/recruitment/jobs、/recruitment/candidates
   - 验证前端页面：curl http://localhost:5173/ 返回 200 与 HTML
   - 验证 vite proxy：curl http://localhost:5173/api/v1/health 返回 db up
4. 停掉两个进程

## 二、发现的问题
- 联调中发现任何接口错误、字段不匹配、构建警告，**必须修复**后再继续
- 修复后重新 build 验证

## 三、README.md（项目根目录）
编写完整 README：
- 项目简介（中文）+ 功能清单
- 技术栈
- 目录结构
- 环境要求（Go 1.25+、Node 20+、PostgreSQL 18）
- 快速开始：1) 建库 `CREATE DATABASE hrms;` 2) 后端 `cd backend && cp .env.example .env`（填入密码）`go run ./cmd/server` 3) 前端 `cd frontend && npm install && npm run dev`
- 内置账号表：admin/admin123、hr/hr123、zhangwei/123456（经理）、liuyang/123456（员工）
- API 概览表

## 四、收尾
1. `cd backend && gofmt -l .` 检查格式，`go vet ./...` 静态检查，全绿
2. `cd frontend && npm run build` 最终构建通过
3. git：`.gitignore` 检查（排除 node_modules、.env、dist、二进制），`git add -A && git commit`，提交信息「feat: HRMS 完整实现」
4. 更新 ARCHITECTURE.md 里程碑状态为已完成

## 汇报格式
联调结果汇总（每个接口状态码）、修复了哪些问题、README 位置、最终 commit hash。
