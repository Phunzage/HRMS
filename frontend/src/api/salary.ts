import client from './client'
import type { PageResult } from './client'

// 工资记录（列表附带员工、部门名称）
export interface SalaryRecord {
  id: number
  employee_id: number
  month: string
  base_salary: number
  allowance: number
  bonus: number
  deduction: number
  net_salary: number
  status: string
  remark: string
  employee_name?: string
  department_name?: string
  created_at: string
  updated_at: string
}

// 创建/更新工资单请求体
export interface SalaryPayload {
  employee_id: number
  month: string
  base_salary: number
  allowance?: number
  bonus?: number
  deduction?: number
  remark?: string
}

// 工资单分页列表
export function fetchSalaries(
  params: {
    month?: string
    status?: string
    employee_id?: number
    department_id?: number
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/salaries', { params }) as Promise<PageResult<SalaryRecord>>
}

// 我的工资单（自助）
export function fetchMySalaries(params: { month?: string; page?: number; page_size?: number } = {}) {
  return client.get('/salaries/mine', { params }) as Promise<PageResult<SalaryRecord>>
}

// 创建工资单
export function createSalary(data: SalaryPayload) {
  return client.post('/salaries', data) as Promise<SalaryRecord>
}

// 更新工资单（仅 draft 状态）
export function updateSalary(id: number, data: Partial<SalaryPayload>) {
  return client.put(`/salaries/${id}`, data) as Promise<SalaryRecord>
}

// 确认工资单（draft → confirmed）
export function confirmSalary(id: number) {
  return client.put(`/salaries/${id}/confirm`) as Promise<null>
}

// 发放工资（confirmed → paid）
export function paySalary(id: number) {
  return client.put(`/salaries/${id}/pay`) as Promise<null>
}
