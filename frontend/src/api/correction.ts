import client from './client'
import type { PageResult } from './client'

// 补卡申请（列表附带申请人、部门名称）
export interface CorrectionItem {
  id: number
  employee_id: number
  work_date: string
  reason: string
  correction_type: string // check_in / check_out
  correction_time: string // HH:MM
  status: string // pending / approved / rejected / cancelled
  approver_id: number | null
  approved_at: string | null
  reject_reason: string
  created_at: string
  updated_at: string
  employee_name?: string
  department_name?: string
}

// 提交补卡申请
export function submitCorrection(data: {
  work_date: string
  correction_type: string
  correction_time: string
  reason: string
}) {
  return client.post('/attendance/corrections', data) as Promise<CorrectionItem>
}

// 补卡申请分页列表（按角色范围过滤）
export function fetchCorrections(
  params: {
    status?: string
    employee_id?: number
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/attendance/corrections', { params }) as Promise<PageResult<CorrectionItem>>
}

// 我的补卡记录
export function fetchMyCorrections(params: { page?: number; page_size?: number } = {}) {
  return client.get('/attendance/corrections/mine', { params }) as Promise<PageResult<CorrectionItem>>
}

// 审批动作已迁移至 /approvals 通用接口（见 approval.ts），此处仅保留取消

// 取消自己的 pending 补卡申请
export function cancelCorrection(id: number) {
  return client.put(`/attendance/corrections/${id}/cancel`) as Promise<null>
}
