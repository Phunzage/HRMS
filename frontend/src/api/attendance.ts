import client from './client'
import type { PageResult } from './client'

// 考勤记录（列表附带员工、部门名称）
export interface Attendance {
  id: number
  employee_id: number
  work_date: string
  check_in: string | null
  check_out: string | null
  status: string
  remark: string
  employee_name?: string
  department_name?: string
  created_at: string
  updated_at: string
}

export interface AttendanceQuery {
  employee_id?: number
  work_date?: string
  month?: string
  status?: string
  page?: number
  page_size?: number
}

// 考勤分页列表
export function fetchAttendance(params: AttendanceQuery = {}) {
  return client.get('/attendance', { params }) as Promise<PageResult<Attendance>>
}

// 上班打卡
export function checkIn() {
  return client.post('/attendance/check-in') as Promise<Attendance>
}

// 下班打卡
export function checkOut() {
  return client.post('/attendance/check-out') as Promise<Attendance>
}

// 手工补录
export function manualAttendance(data: {
  employee_id: number
  work_date: string
  check_in?: string
  check_out?: string
  status?: string
  remark?: string
}) {
  return client.post('/attendance/manual', data) as Promise<Attendance>
}

// 更新考勤记录
export function updateAttendance(id: number, data: Record<string, unknown>) {
  return client.put(`/attendance/${id}`, data) as Promise<null>
}

// 删除考勤记录
export function deleteAttendance(id: number) {
  return client.delete(`/attendance/${id}`) as Promise<null>
}
