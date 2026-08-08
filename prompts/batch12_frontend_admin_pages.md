# 任务批次 12：前端页面 — 用户管理 / 角色管理 / 审计日志

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 10/11 已完成后端：用户/角色管理 API、审计日志、请假考勤联动、Excel 导出。本批次实现**前端三个新页面**并接入菜单。中文界面，daisyUI + Tailwind + lucide-react 图标（**禁止 emoji 图标**，全部用 lucide）。

## 一、API 封装（`frontend/src/api/`）

### 1. 新建 `user.ts`
- `listUsers(params)` GET /users（分页+keyword+role_id+status）
- `createUser(data)` POST /users
- `updateUser(id, data)` PUT /users/:id
- `resetPassword(id, password)` PUT /users/:id/password
- `deleteUser(id)` DELETE /users/:id
- `listRoles()` GET /users/roles（下拉用）

### 2. 新建 `role.ts`
- `listRoles()` GET /roles（含 permissions 数组）
- `listPermissions()` GET /permissions（分组）
- `createRole(data)` POST /roles
- `updateRole(id, data)` PUT /roles/:id
- `deleteRole(id)` DELETE /roles/:id

### 3. 新建 `audit.ts`
- `listAuditLogs(params)` GET /audit-logs（分页+module/action/username/date 过滤）

## 二、新页面（`frontend/src/pages/`）

### 1. `Users.tsx` — 用户管理（admin/hr 可见）
- 工具栏：搜索框（用户名/姓名）、角色下拉、状态下拉、新增按钮
- daisyUI table：用户名/姓名/邮箱/角色 badge/状态 badge(启用绿/禁用灰)/创建时间/操作（编辑、重置密码、删除）
- 新增/编辑 Modal：用户名（编辑时只读）、姓名、邮箱、角色下拉；新增时有密码输入
- 重置密码 Modal：输入新密码（min 6）
- 删除：confirm 提示「删除后该账号无法登录」
- 权限：admin 全部操作；hr 也可操作（后端已控制 role:write）

### 2. `Roles.tsx` — 角色管理（admin 可见，hr 仅读）
- 表格：角色名/编码/描述/权限数/操作（编辑、删除）
- 新增/编辑 Modal：
  - 名称、编码（编辑时只读）、描述
  - 权限勾选区：按模块分组展示（如 组织/人事/考勤/请假/薪酬/招聘/系统），checkbox 勾选权限点（`permission:read/write` 成对，read 与 write 可独立勾选）
- 删除：有用户时后端返回 400 提示，前端 alert 展示；admin 角色隐藏删除按钮
- hr 登录时隐藏「新增/编辑/删除」按钮（role_code==='admin' 才显示）

### 3. `AuditLogs.tsx` — 审计日志（admin/hr 可见）
- 工具栏：模块下拉、操作下拉、用户名输入、日期范围（两个 type=date）、查询按钮
- daisyUI table：时间/操作者/模块/动作 badge/资源/详情/IP
- 分页
- 只读页面，无任何写操作

## 三、路由与菜单

1. `App.tsx` 增加路由：`/users` → Users、`/roles` → Roles、`/audit-logs` → AuditLogs（都包在 MainLayout 内）
2. `MainLayout.tsx` 菜单新增（用 lucide 图标）：
   - 系统管理分组下（或平级追加）：
     - 用户管理 → `/users`（icon: UserCog，roles: admin/hr）
     - 角色管理 → `/roles`（icon: ShieldCheck，roles: admin）
     - 审计日志 → `/audit-logs`（icon: ScrollText，roles: admin/hr）
   - 菜单 roles 数组与后端权限一致：hr 看不到「角色管理」
3. 若原菜单没有分组，可以用 daisyUI menu 的 `<li><span class="menu-title">系统管理</span></li>` 加分组标题

## 四、完成后必须执行
1. `cd frontend && npm run build` 构建通过，0 TS 错误
2. 启动前后端验证：
   - admin 登录 → /users 页面能列出用户、创建用户成功
   - /roles 能勾选权限创建角色
   - /audit-logs 能看到登录/创建等记录
   - hr 登录 → 菜单无「角色管理」，/users 可访问
   - liuyang（employee）登录 → 菜单无这三个入口
3. 停掉两个进程

## 汇报格式
新增/修改文件清单、构建结果、三个页面的功能验证结果、三种角色菜单可见性验证。
