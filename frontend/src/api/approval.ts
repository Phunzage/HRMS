import client from './client'

// 我的审批任务项（GET /approvals/mine 返回，JOIN 业务表带出申请人/摘要/业务明细）
export interface ApprovalTaskItem {
  id: number // 任务 ID（审批动作凭此）
  instance_id: number
  step_order: number
  step_name: string
  approver_type: string // manager / hr
  biz_type: string // leave / attendance_correction
  biz_id: number // 业务记录 ID
  applicant_name: string
  department_name?: string
  summary: string // 业务摘要（申请人 + 类型 + 时间范围）
  status: string // pending / approved / rejected
  link: string
  // 请假业务明细（biz_type=leave 时返回）
  leave_type?: string
  days?: number
  reason?: string
  // 补卡业务明细（biz_type=attendance_correction 时返回）
  work_date?: string
  correction_type?: string
  correction_time?: string
  created_at: string
}

// 审批实例
export interface ApprovalInstance {
  id: number
  flow_id: number
  biz_type: string
  biz_id: number
  status: string // pending / approved / rejected / cancelled
  current_step: number
  created_by: number
  created_at: string
  updated_at: string
}

// 审批进度步骤项（GET /approvals/biz 的 tasks）
export interface ApprovalStep {
  id: number
  instance_id: number
  step_order: number
  step_name: string
  approver_type: string
  approver_id: number | null
  approver_name: string
  status: string // pending / approved / rejected
  comment: string
  acted_at: string | null
  created_at: string
}

// 审批进度（实例状态 + 步骤时间线）
export interface ApprovalProgress {
  instance: ApprovalInstance
  tasks: ApprovalStep[]
}

// 我的审批任务列表（待审批 / 历史）
// status：pending=待我审批；done=我处理的（通过+驳回）
export function fetchMyApprovals(params: { status?: string } = {}) {
  return client.get('/approvals/mine', { params }) as Promise<ApprovalTaskItem[]>
}

// 按业务类型+业务 ID 查询审批实例与进度（前端详情页/时间线展示）
export function fetchApprovalProgress(bizType: string, bizId: number) {
  return client.get('/approvals/biz', {
    params: { biz_type: bizType, biz_id: bizId },
  }) as Promise<ApprovalProgress>
}

// 审批通过响应（审批实例 + 下一步骤信息，前端 toast 动态提示「等待 XX 审批」用）
export interface ApproveResult extends ApprovalInstance {
  /** 下一步骤名称（如 人事专员审批；终审通过=空串） */
  next_step_name?: string
  /** 下一步骤类型（manager/role/...；终审通过=空串） */
  next_step_type?: string
}

// 审批通过（comment 可选，返回更新后的实例 + 下一步骤名称，前端据 status / next_step_name 动态提示）
export function approveTask(taskId: number, comment = '') {
  return client.put(`/approvals/tasks/${taskId}/approve`, { comment }) as Promise<ApproveResult>
}

// 驳回（comment 必填）
export function rejectTask(taskId: number, comment: string) {
  return client.put(`/approvals/tasks/${taskId}/reject`, { comment }) as Promise<null>
}
