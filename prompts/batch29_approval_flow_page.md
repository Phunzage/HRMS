# 任务批次 29：admin 审批流程配置页（前端）

你在 `C:\Users\Phunzage\Documents\Code\HRMS`（当前工作目录）。批次 28 已完成后端：ApprovalStep/ApprovalTask 支持 position 岗位池、审批引擎按岗位匹配、admin 管理 API（GET/PUT /api/v1/approval-flows）、权限点 approval_flow:read/write。本批次实现**前端配置页**。中文界面，daisyUI + Tailwind + lucide-react（禁止 emoji）。

## 一、API 封装（frontend/src/api/approval.ts 扩展或新建 approvalFlow.ts）

```ts
export interface ApprovalStepItem {
  id?: number
  step_order: number
  approver_type: 'manager' | 'hr' | 'position'
  approver_ref: number   // position 时为岗位 ID，否则 0
  name: string
}
export interface ApprovalFlowItem {
  id: number
  code: string
  name: string
  status: string
  steps: ApprovalStepItem[]
}
export function listApprovalFlows(): Promise<ApprovalFlowItem[]>   // GET /approval-flows → data.list
export function updateApprovalFlow(code: string, body: { name: string; steps: ApprovalStepItem[] }): Promise<ApprovalFlowItem>  // PUT /approval-flows/:code
```
类型常量：`APPROVER_TYPE_LABEL = { manager: '部门经理', hr: 'HR 终审', position: '指定岗位' }` 导出供页面使用。

## 二、审批流程配置页（新建 frontend/src/pages/ApprovalFlows.tsx）

### 页面结构
- 标题「审批流程配置」+ 说明文字（「配置请假/补卡的审批级别与每级审批人，修改仅影响新申请，已进行中的审批不受影响」用 alert alert-info 展示）
- 两张流程卡片（请假流程 / 补卡申请流程），每张：
  - 卡片头：流程名 + 状态 badge（active=启用）+ 「编辑配置」按钮（Workflow 图标）
  - 卡片体：**步骤时间线/列表**，每行：序号徽章（1,2,3…）+ 步骤名 + 审批人类型 badge（部门经理=badge-info / HR终审=badge-primary / 指定岗位=badge-warning，岗位名显示在 name 括号里或 badge 旁）
  - 底部显示当前级别数「共 N 级审批」
- 空状态：流程无步骤时显示「未配置审批步骤」

### 编辑 Modal（核心交互）
- 打开时加载该流程 steps 到本地 state（深拷贝，不直接改原始）
- **动态步骤行**（可增删）：
  - 每行：序号（自动 1..N）+ 步骤名称 input + 审批人类型 select（部门经理/HR终审/指定岗位）+ **岗位下拉**（仅当类型=指定岗位时显示，从 GET /positions 拉取全岗位，显示「岗位名」）
  - 「添加一级」按钮（列表尾部，Plus 图标）追加一行（step_order 自动 +1）
  - 每行右侧「删除」按钮（Trash2 图标，仅当行数 >1 时可用，至少保留 1 级）
- **步骤名自动建议**：类型选择后自动填充默认名（manager→「部门经理审批」、hr→「HR 终审」、position→「{岗位名}审批」），用户可改
- 保存校验（前端）：
  - 每行步骤名非空
  - 类型=position 时必须已选岗位
  - 提交 PUT /approval-flows/:code
- 成功后 toast + 刷新列表

### 权限
- 菜单项「审批流程」仅 admin 可见（MainLayout 系统管理分组：仪表盘/公告管理/用户管理/角色管理/审计日志/**审批流程**）
- 路由 /approval-flows 注册；页面内 role!=='admin' 时显示无权限提示

## 三、MainLayout 菜单增加

系统管理分组末尾加：「审批流程」→ /approval-flows，图标 GitBranch 或 Workflow，roles: ['admin']

## 四、完成后必须执行
1. `cd frontend && npm run build` 0 错误 0 警告
2. 启动前后端实测（admin JWT 注入法）：
   - admin 登录 → 侧边栏出现「审批流程」；点击进入页面显示两张流程卡片（各 2 级：部门经理→HR终审）
   - 编辑请假流程 → 添加第 3 级（类型=指定岗位，选「人事专员」）→ 保存成功，卡片显示 3 级（部门经理→人事专员审批→HR终审）
   - 刷新页面后配置保持
   - 步骤删除/添加交互正常；类型切换显示/隐藏岗位下拉
3. 停掉两个进程，端口释放

## 汇报格式
新增/修改文件、build 结果、页面截图描述（文字）、三级流程配置保存与持久化实测、菜单回归。
