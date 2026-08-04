# 任务批次 6：前端脚手架 + 登录 + 布局

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。后端 API 已全部完成（认证/组织/员工/考勤/请假/薪酬/招聘/统计）。本批次创建 **React 前端项目**。

## 目标
Vite + React 18 + TypeScript + TailwindCSS + daisyUI + React Router + zustand + axios 的前端骨架，含登录页、主布局、路由守卫。

## 一、脚手架创建
```
cd frontend && npm create vite@latest . -- --template react-ts
```
（如目录冲突，先清空 frontend 目录再创建。npm 可用，无需 sudo。）

## 二、安装依赖
```
npm install axios react-router-dom zustand
npm install -D tailwindcss @tailwindcss/vite
npm install daisyui
```
Tailwind 用 v4（`@tailwindcss/vite` 插件方式）：`vite.config.ts` 中加 `tailwindcss()` 插件 + `@plugin "daisyui";` 写入 `src/index.css` 的 `@import "tailwindcss";` 之后；`src/index.css` 里加 `@plugin "daisyui";` 和主题配置 `@plugin "daisyui/theme" { name: "light"; ... }` 或直接 daisyui 默认主题。

## 三、目录结构
```
frontend/src/
├── main.tsx            # 入口：BrowserRouter + AuthProvider
├── App.tsx             # 路由定义
├── index.css           # tailwind + daisyui
├── api/
│   ├── client.ts       # axios 实例 baseURL '/api/v1'，请求拦截器注入 token，响应拦截器统一处理 code!=0 与 401 跳登录
│   ├── auth.ts         # login / me / changePassword
│   ├── employee.ts     # 员工/部门/岗位接口
│   ├── attendance.ts   # 考勤接口
│   ├── leave.ts        # 请假接口
│   ├── salary.ts       # 薪酬接口
│   └── recruitment.ts  # 招聘接口
├── store/
│   └── auth.ts         # zustand：token/user 持久化 localStorage
├── router/
│   └── guard.tsx       # RequireAuth 组件：无 token 跳 /login；有 token 调 /auth/me 拿 user
├── layouts/
│   └── MainLayout.tsx  # 侧边栏导航 + 顶栏（用户信息/退出）
├── pages/
│   ├── Login.tsx       # 登录页（居中卡片、用户名密码、错误提示）
│   ├── Dashboard.tsx   # 仪表盘（统计卡片）
│   ├── EmployeeList.tsx    # 员工列表（占位，后续批次实现）
│   ├── DepartmentTree.tsx  # 部门管理（占位）
│   ├── AttendancePage.tsx  # 考勤（占位）
│   ├── LeavePage.tsx       # 请假（占位）
│   ├── SalaryPage.tsx      # 薪酬（占位）
│   └── RecruitmentPage.tsx # 招聘（占位）
```

## 四、关键实现
1. **api/client.ts**：axios.create({baseURL:'/api/v1'})；请求拦截器从 localStorage 取 token 加 Authorization；响应拦截器：data.code===0 时返回 data.data；否则 reject（message 用后端返回）；401 时清 token 跳 /login。
2. **store/auth.ts**：zustand + persist（localStorage key 'hrms_token'）：state {token, user}，actions {setAuth, logout}。
3. **router/guard.tsx**：RequireAuth 包裹需要登录的路由；未登录重定向 /login；登录后访问 /login 重定向 /。
4. **layouts/MainLayout.tsx**：左侧边栏（daisyUI menu）：仪表盘、员工管理、部门岗位、考勤、请假审批、薪酬、招聘；顶栏右侧显示当前用户 full_name + 角色 + 退出按钮。用 Outlet 渲染子路由。
5. **pages/Login.tsx**：卡片式居中，daisyUI input + btn，提交调 auth.login，成功后 setAuth + navigate('/')。
6. **pages/Dashboard.tsx**：调 /dashboard/stats 渲染统计卡片 grid（员工总数、在职、本月新增、出勤率、待审批请假、在招职位等），部门人数分布简单条状图（纯 div 宽度百分比即可，不引图表库）。
7. **vite.config.ts**：server.proxy 配置 `/api` → `http://localhost:8080`。
8. 中文界面，组件用 daisyUI class（btn、input、card、menu、table、badge、alert）。

## 五、完成后必须执行
1. `cd frontend && npm run build` 构建通过（TypeScript 无错误）
2. `npm run dev` 启动 → 用 curl 确认 `http://localhost:5173/` 返回 HTML、`http://localhost:5173/api/v1/health` 代理到后端返回 db up
3. 停掉 dev server

## 汇报格式
创建的文件清单、build 结果、dev server 验证结果。
