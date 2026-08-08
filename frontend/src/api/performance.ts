import client from './client'
import type { PageResult } from './client'

// 绩效记录（列表附带员工/部门/考核人信息）
export interface PerformanceItem {
  id: number
  employee_id: number
  reviewer_id: number
  period: string // 2026-H1 / 2026-H2
  score: number
  rating: string // S/A/B/C/D
  comments: string
  status: string // draft / submitted / approved
  submitted_at: string | null
  approved_at: string | null
  created_at: string
  updated_at: string
  employee_name?: string
  department_name?: string
  emp_no?: string
  reviewer_name?: string
  rating_name?: string
}

// 绩效分页列表
export function fetchPerformance(
  params: {
    period?: string
    status?: string
    employee_id?: number
    department_id?: number
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/performance', { params }) as Promise<PageResult<PerformanceItem>>
}

// 我的绩效记录
export function fetchMyPerformance(params: { page?: number; page_size?: number } = {}) {
  return client.get('/performance/mine', { params }) as Promise<PageResult<PerformanceItem>>
}

// 创建绩效
export function createPerformance(data: {
  employee_id: number
  period: string
  score: number
  comments?: string
}) {
  return client.post('/performance', data) as Promise<PerformanceItem>
}

// 更新绩效（仅 draft）
export function updatePerformance(id: number, data: { score?: number; comments?: string }) {
  return client.put(`/performance/${id}`, data) as Promise<PerformanceItem>
}

// 提交绩效（draft → submitted）
export function submitPerformance(id: number) {
  return client.put(`/performance/${id}/submit`) as Promise<null>
}

// 批准绩效（submitted → approved，hr/admin）
export function approvePerformance(id: number) {
  return client.put(`/performance/${id}/approve`) as Promise<null>
}

// 删除绩效（仅 draft）
export function deletePerformance(id: number) {
  return client.delete(`/performance/${id}`) as Promise<null>
}
