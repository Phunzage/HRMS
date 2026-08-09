# 任务批次 34：VP 角色菜单修复 + 审批提示动态化

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。用户实测两个问题，根因已定位，按方案修复。

## 问题 1：VP（副总经理）登录后侧边栏无任何菜单

**根因**：`frontend/src/layouts/MainLayout.tsx` 菜单项 `roles` 数组全是**小写**（`'hr'/'manager'/'employee'/'admin'`），而自定义角色（如用户创建的 VP 副总经理）的 `role_code` 是**大写 'VP'**——`item.roles.includes(user.role_code)` 大小写不匹配 → 全部菜单被过滤 → VP 侧边栏空白。但审批任务按权限点（leave:approve）匹配，所以 VP 能审批却没菜单。

**修复方案：菜单可见性改为「权限点优先 + roles 兜底」，统一大小写**

### 1. MainLayout.tsx 菜单项增加 `perms` 字段（权限点驱动）
给每个菜单项补充对应权限点（有权限点则按权限点判断，roles 仅作兜底/浏览性菜单）：
```ts
{ to: '/', label: '仪表盘', icon: LayoutDashboard, end: true, roles: ['admin', 'hr'], perms: ['dashboard:read'] },
{ to: '/employees', label: '员工管理', icon: Users, roles: ['hr', 'manager'], perms: ['employee:read'] },
{ to: '/departments', label: '部门管理', icon: Building2, roles: ['hr', 'manager'], perms: ['department:read'] },
{ to: '/positions', label: '岗位管理', icon: FolderKanban, roles: ['hr'], perms: ['position:read'] },
{ to: '/attendance', label: '考勤管理', icon: Clock, roles: ['hr', 'manager', 'employee'], perms: ['attendance:read'] },
{ to: '/leaves', label: '请假审批', icon: ClipboardList, roles: ['hr', 'manager', 'employee'], perms: ['leave:read'] },
{ to: '/approvals', label: '我的审批', icon: Inbox, roles: ['hr', 'manager'], perms: ['leave:approve', 'attendance_correction:approve'], badge: 'pending_approvals' },
{ to: '/salaries', label: '薪酬管理', icon: Wallet, roles: ['hr', 'employee'], perms: ['salary:read'] },
{ to: '/announcements', label: '公告管理', icon: Megaphone, roles: ['admin', 'hr', 'manager', 'employee'], perms: ['announcement:read'], badge: 'unread_announcements' },
{ to: '/performance', label: '绩效管理', icon: Target, roles: ['hr', 'manager', 'employee'], perms: ['performance:read'] },
{ to: '/recruitment', label: '招聘管理', icon: Briefcase, roles: ['hr'], perms: ['recruitment:read'] },
{ to: '/profile', label: '个人中心', icon: UserCircle, roles: ['hr', 'manager', 'employee'], perms: [] },  // 全部登录用户
```
系统管理菜单（users/roles/audit-logs/approval-flows）保持 `roles: ['admin']`（admin 专属，无对应业务权限点）。

### 2. 过滤逻辑（关键改动）
当前 `visibleMenu` 过滤（约 183-190 行）改为：
```ts
// 1. 有 perms 的菜单：用户拥有任一权限点即显示（hasAny）；perms 为空数组 = 登录即可见
// 2. 无 perms 的菜单：按 roles 过滤（大小写不敏感比较：role_code.toLowerCase()）
// 3. admin 专属（系统管理）保持 roles=['admin']
```
具体：
```ts
const canSee = (item: MenuItem) => {
  if (item.perms && item.perms.length > 0) return hasAny(item.perms)
  if (item.perms && item.perms.length === 0) return true  // 登录即可见（如个人中心）
  return !user?.role_code || item.roles.map(r => r.toLowerCase()).includes((user.role_code || '').toLowerCase())
}
```
- 注意：`usePerm` 已在 MainLayout 可用（批次 33 引入，确认 import）
- **hasAny 的依赖问题**：`user.permissions` 来自 /auth/me——确认登录/刷新后 store 有 permissions；若 usePerm 读 store 的 user.permissions 为空（旧 localStorage 无该字段），需在 guard/auth store 初始化时确保 /auth/me 拉取后写入
- VP 角色（code='VP'，有 employee:read 等 14 权限）→ 员工/部门/岗位/考勤/请假/我的审批/公告/绩效/薪酬/招聘全部可见 ✅

### 3. 角色显示
顶栏 `roleNames[user.role_code] || user.role_code`——VP 无映射显示 "VP"，可接受；如角色表有 name 可显示中文名（检查 roleNames 映射，若列表来自固定常量则保持）

## 问题 2：审批提示硬编码「等待 HR 终审」

**根因**：`ApprovalsPage.tsx:103`、`AttendanceCorrections.tsx:186`、`LeavePage.tsx:191` 三处 toast 硬编码 `'已通过，等待 HR 终审'`——但审批流程是可配置的（可能是 VP/人事专员终审）。

**修复：后端返回下一步骤名称，前端动态提示**

### 后端（internal/handler/approval.go + service/approval.go）
`ApproveTask` 响应 data 增加：
```go
NextStepName string `json:"next_step_name"` // 下一步骤名称（无下一步=空）
NextStepType string `json:"next_step_type"` // 下一步骤类型（manager/role/...，无=空）
```
- 在 ApproveTask service 中已解析 nextStep（约 162-182 行），计算其显示名：
  - manager → 「部门经理审批」
  - role → 「{角色名}审批」（查 roles.name）
  - hr → 「HR终审」；position → 岗位名；permission → 权限点名
- 无下一步（终审通过）→ 空字符串

### 前端三处
- `ApprovalsPage.tsx`：`approveTask` 响应中若 `data.next_step_name` 非空 → `toast.success('已通过，等待' + next_step_name)`；否则 `toast.success('审批已通过')`
- `LeavePage.tsx`、`AttendanceCorrections.tsx`：同样逻辑（它们也调 approveTask）
- `src/api/approval.ts`：`approveTask` 返回类型增加 next_step_name 字段
- 若审批接口响应结构统一（approveTask 返回整个 data），确认三处页面取到的是同一字段

## 完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告；`cd backend && go build ./...`、`go vet ./...`
2. 启动实测（JWT 注入法）：
   - **VP 场景**：丁宣（ding，role VP）登录 → 侧边栏显示：员工管理/部门管理/岗位管理/考勤管理/请假审批/我的审批/薪酬管理/公告管理/绩效管理/招聘管理/个人中心（按 VP 的 14 权限）✅；无系统管理、无仪表盘（VP 无 dashboard:read）——**确认 dashboard:read 是否应给 VP？若用户希望 VP 看仪表盘，在 seed 给 VP 加 dashboard:read 或本批次说明。默认按权限点：VP 无则无**
   - 张佳提交请假（算法组→刘尧一级）→ 刘尧通过 → toast 显示「已通过，等待 人事专员审批」（若请假二级是 role=人事）或对应实际配置
   - 二级通过 → toast「审批已通过」（终审）
   - 补卡同样验证
3. 停掉进程，端口释放

## 汇报格式
菜单权限点改造清单、VP 登录实测菜单快照、审批提示动态化前后对比、build 结果。
