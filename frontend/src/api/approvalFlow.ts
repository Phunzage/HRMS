import client from './client'

// 审批流程步骤配置项（编辑 Modal 内步骤行；id 仅已有步骤返回，新增步骤无）
export interface ApprovalStepItem {
  id?: number
  step_order: number
  approver_type: 'manager' | 'hr' | 'position' // manager=部门经理 / hr=HR终审 / position=指定岗位
  approver_ref: number // position 时为岗位 ID，否则 0
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

// 审批人类型 → 中文名称（页面 badge 与下拉选项共用）
export const APPROVER_TYPE_LABEL: Record<ApprovalStepItem['approver_type'], string> = {
  manager: '部门经理',
  hr: 'HR 终审',
  position: '指定岗位',
}

// 查询所有审批流程及步骤（GET /approval-flows → data.list）
export function listApprovalFlows(): Promise<ApprovalFlowItem[]> {
  return (client.get('/approval-flows') as Promise<{ list: ApprovalFlowItem[] }>).then((res) => res.list)
}

// 更新指定编码的审批流程（PUT /approval-flows/:code，返回更新后的流程详情）
export function updateApprovalFlow(
  code: string,
  body: { name: string; steps: ApprovalStepItem[] },
): Promise<ApprovalFlowItem> {
  return client.put(`/approval-flows/${code}`, body) as Promise<ApprovalFlowItem>
}
