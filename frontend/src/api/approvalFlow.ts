import client from './client'

// 审批人类型：manager=部门经理 / permission=按权限审批 / hr=HR终审(旧) / position=指定岗位(旧)
export type ApproverType = 'manager' | 'permission' | 'hr' | 'position'

// 审批流程步骤配置项（编辑 Modal 内步骤行；id 仅已有步骤返回，新增步骤无）
export interface ApprovalStepItem {
  id?: number
  step_order: number
  approver_type: ApproverType
  approver_ref: number // position 时为岗位 ID，否则 0
  approver_permission: string // permission 时为审批权限点 code（如 leave:approve），否则空
  name: string
}

// 审批流程配置（含步骤，按 step_order 升序）
export interface ApprovalFlowItem {
  id: number
  code: string
  name: string
  status: string // active / inactive
  steps: ApprovalStepItem[]
}

// 拥有某审批权限点的角色
export interface ApprovalPermissionRole {
  id: number
  name: string
}

// 审批权限点及拥有它的角色列表（GET /approval-flows 顶层 approval_permissions）
export interface ApprovalPermission {
  code: string
  name: string
  roles: ApprovalPermissionRole[]
}

// 审批流程列表 + 审批权限点（流程卡片与编辑 Modal 权限下拉共用）
export interface ApprovalFlowListResult {
  list: ApprovalFlowItem[]
  approval_permissions: ApprovalPermission[]
}

// 审批人类型 → 中文名称（页面 badge 与下拉选项共用）
export const APPROVER_TYPE_LABEL: Record<ApproverType, string> = {
  manager: '部门经理',
  permission: '按权限审批',
  hr: 'HR 终审',
  position: '指定岗位',
}

// 新配置可选的审批人类型（hr/position 仅兼容旧配置，不再提供新选项）
export const NEW_APPROVER_TYPES: ApproverType[] = ['manager', 'permission']

// 审批权限点 code → 名称兜底（GET /approval-flows 未返回时使用）
export const APPROVAL_PERMISSION_LABELS: Record<string, string> = {
  'leave:approve': '请假审批',
  'attendance_correction:approve': '补卡审批',
}

// 查询所有审批流程及步骤 + 审批权限点（GET /approval-flows）
export function listApprovalFlows(): Promise<ApprovalFlowListResult> {
  return client.get('/approval-flows') as Promise<ApprovalFlowListResult>
}

// 更新指定编码的审批流程（PUT /approval-flows/:code，返回更新后的流程详情）
export function updateApprovalFlow(
  code: string,
  body: { name: string; steps: ApprovalStepItem[] },
): Promise<ApprovalFlowItem> {
  return client.put(`/approval-flows/${code}`, body) as Promise<ApprovalFlowItem>
}
