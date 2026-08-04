import client from './client'

// 部门人数分布项
export interface DeptCount {
  department_name: string
  count: number
}

// 按月计数项（招聘趋势）
export interface MonthCount {
  month: string
  count: number
}

// 按月金额项（工资总额趋势）
export interface MonthAmount {
  month: string
  total_net: number
}

// 仪表盘统计结果
export interface DashboardStats {
  total_employees: number
  active_employees: number
  month_new_employees: number
  month_resigned: number
  month_attendance_rate: number
  today_leave_count: number
  pending_leave_count: number
  pending_salary_count: number
  open_job_count: number
  candidate_count: number
  department_distribution: DeptCount[]
  recruit_trend: MonthCount[]
  salary_trend: MonthAmount[]
}

// 仪表盘统计
export function fetchDashboardStats() {
  return client.get('/dashboard/stats') as Promise<DashboardStats>
}
