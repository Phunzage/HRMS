import client from './client'
import type { PageResult } from './client'

// 请假申请（列表附带请假人、部门名称）
export interface LeaveRequest {
  id: number
  employee_id: number
  leave_type: string
  start_date: string
  end_date: string
  days: number
  reason: string
  status: string
  approver_id: number | null
  approved_at: string | null
  reject_reason: string
  employee_name?: string
  department_name?: string
  created_at: string
  updated_at: string
}

// 假期余额项
export interface LeaveBalance {
  leave_type: string
  total: number
  used: number
  remaining: number
}

// 我的请假列表（含余额信息）
export interface LeaveMineResult extends PageResult<LeaveRequest> {
  balance: LeaveBalance[]
}

// 提交请假申请请求体
export interface SubmitLeavePayload {
  leave_type: string
  start_date: string // YYYY-MM-DD
  end_date: string // YYYY-MM-DD
  reason: string
}

// 请假分页列表（按角色范围过滤）
export function fetchLeaves(
  params: { status?: string; employee_id?: number; page?: number; page_size?: number } = {},
) {
  return client.get('/leaves', { params }) as Promise<PageResult<LeaveRequest>>
}

// 我的申请列表（含余额）
export function fetchMyLeaves(params: { page?: number; page_size?: number } = {}) {
  return client.get('/leaves/mine', { params }) as Promise<LeaveMineResult>
}

// 我的假期余额
export function fetchLeaveBalance() {
  return client.get('/leaves/balance') as Promise<LeaveBalance[]>
}

// 提交请假申请
export function submitLeave(data: SubmitLeavePayload) {
  return client.post('/leaves', data) as Promise<LeaveRequest>
}

// 审批动作已迁移至 /approvals 通用接口（见 approval.ts），此处仅保留取消

// 取消申请
export function cancelLeave(id: number) {
  return client.put(`/leaves/${id}/cancel`) as Promise<null>
}
