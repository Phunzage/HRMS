# 任务批次 33：前端 — 审批流程页角色选择 + UI 权限与权限点同步

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 32 已完成后端：部门经理递归解析、未指派 manager 任务按权限可见、审批步骤支持 role 角色池（ApproverRoleID）、API 返回 approver_role_id/name。本批次实现**前端**。中文界面，daisyUI + Tailwind + lucide-react（禁止 emoji）。

## 一、审批流程配置页改「按角色审批」（ApprovalFlows.tsx + api/approvalFlow.ts）

### API 封装
- `ApprovalStepItem` 增加 `approver_role_id: number`、`approver_role_name?: string`
- 类型常量：`APPROVER_TYPE_LABEL` 增加 `role: '指定角色'`；步骤类型下拉选项改为：
  - `manager` 部门经理（申请人直属上级）
  - `role` 指定角色（如人事专员、副总经理）
  - 旧类型（hr/position/permission）仅兼容显示
- 需要角色列表：`listRoles()` 已存在于 api/role.ts（GET /roles，返回 {id,name,code}），直接复用

### 编辑 Modal 步骤行
- 类型下拉选「指定角色」时：显示**角色下拉**（全部角色，含部门经理/人事专员/副总经理等），存 approver_role_id
- 步骤名自动建议：选角色后自动填「{角色名}审批」（如「人事专员审批」「副总经理审批」），可改
- 选「部门经理」→ 步骤名自动「部门经理审批」
- 保存校验：role 类型必须已选角色；PUT body 带 approver_role_id

### 卡片步骤展示
- role 类型 badge：「指定角色·{角色名}」badge-secondary（如「指定角色·人事专员」）
- 步骤行下方小字「{角色名} 可审批」
- manager 保持「部门经理」badge-info

### 说明文案更新
- 页面顶部 alert 增加「审批优先级=步骤顺序：第 1 级先审，通过后进入第 2 级…」

## 二、UI 权限与权限点同步（核心）

**问题 5 根因**：前端大量 `canManage = role === 'hr'` / `role === 'admin'` 硬编码，与后端权限点（role_permissions）不同步——admin 给某角色加了权限但 UI 入口不出现/误出现。

### 方案：前端获取当前用户权限点列表
1. 后端已有 GET /auth/me 返回用户信息——**确认是否含 permissions 列表**；若无，批次 32 或本批次后端补充：/auth/me 增加 `permissions: string[]`（当前用户全部权限点 code，JOIN role_permissions→permissions）。若批次 32 未做，本批次让 claude 在后端补（小改动，handler/auth.go me 接口查权限点列表返回）
2. `frontend/src/store/auth.ts`：user 对象增加 `permissions: string[]`；auth store 从 /auth/me 或登录响应持久化
3. 新建 hook `frontend/src/hooks/usePerm.ts`：
```ts
export function usePerm() {
  const perms = useAuthStore((s) => s.user?.permissions ?? [])
  return {
    has: (code: string) => perms.includes(code),
    hasAny: (codes: string[]) => codes.some((c) => perms.includes(c)),
  }
}
```

### 各页面 canManage 改为权限点判断
| 页面 | 现硬编码 | 改为 |
|------|---------|------|
| Announcements.tsx | `role==='hr'||role==='admin'` | `has('announcement:write')`（发布/编辑/删除按钮） |
| AttendancePage.tsx | canManage `role==='hr'` | `has('attendance:write')`；canFilter/canReport `role admin/hr/manager` | `hasAny(['attendance:read'])` 保留（员工也有 read 但后端已限本人，筛选下拉对 employee 仍隐藏——用 `role!=='employee'` 判断筛选显示） |
| DepartmentTree.tsx | `role==='hr'` | `has('department:write')` |
| EmployeeList.tsx | `role==='hr'` | `hasAny(['employee:write'])` |
| PositionPage.tsx | `role==='hr'` | `has('position:write')` |
| SalaryPage.tsx | `role==='hr'` | `has('salary:write')` |
| PerformancePage.tsx | `role==='hr'` | `has('performance:write')` |
| RecruitmentPage.tsx | `role==='hr'` | `has('recruitment:write')` |
| LeavePage.tsx | showTodo 含 admin | `role!=='employee'` 或按权限 |
| ApprovalFlows.tsx | `role==='admin'` | `has('approval_flow:write')` |
| MainLayout 菜单 | roles 数组过滤 | 保留 roles 过滤（菜单级别）+ 敏感项（如审批流程）加权限校验（hasAny）——优先权限点，无权限点概念的角色菜单按 roles 即可 |

**原则**：有权限点的操作（写操作按钮）一律用 has(code)；无权限点概念的浏览性功能保留 roles 过滤。employee 的考勤筛选隐藏逻辑保留 `role !== 'employee'`。

## 三、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（JWT 注入法）：
   - admin 角色管理：给 manager 角色勾选 `announcement:write` → manager 登录公告页**出现「发布公告」按钮**；hr 无此权限时（如给 employee 勾 attendance:write 但无 announcement:write）公告页无发布按钮 ✅（问题 5 修复验证）
   - admin 审批流程页：请假流程 step2 改为「指定角色·人事专员」→ 保存 → 卡片显示「指定角色·人事专员」+「人事专员 可审批」
   - 页面写按钮权限联动：给 employee 勾 employee:write（测试）→ 员工管理页出现新增按钮（临时测试后还原）；hr 未勾 salary:write（测试）→ 薪酬页无生成按钮
3. 停掉两个进程，端口释放

## 汇报格式
/auth/me 权限点补充、usePerm hook、各页面 canManage 改造清单、问题 5 修复实测（manager 公告发布按钮）、审批流程页角色选择实测、build 结果。
