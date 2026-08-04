import client from './client'
import type { PageResult } from './client'

// 员工档案（列表/详情附带部门、岗位名称）
export interface Employee {
  id: number
  emp_no: string
  user_id: number | null
  name: string
  gender: number
  birth_date: string | null
  id_card: string
  phone: string
  email: string
  department_id: number
  position_id: number
  hire_date: string | null
  status: number
  address: string
  avatar: string
  remark: string
  department_name?: string
  position_name?: string
  created_at: string
  updated_at: string
}

// 部门节点（树形）
export interface Department {
  id: number
  name: string
  parent_id: number
  manager_id: number | null
  sort_order: number
  remark: string
  created_at: string
  updated_at: string
  children: Department[]
}

// 岗位
export interface Position {
  id: number
  name: string
  code: string
  department_id: number
  level: number
  remark: string
}

export interface EmployeeQuery {
  keyword?: string
  department_id?: number
  position_id?: number
  status?: number
  page?: number
  page_size?: number
}

// 员工分页列表
export function fetchEmployees(params: EmployeeQuery = {}) {
  return client.get('/employees', { params }) as Promise<PageResult<Employee>>
}

// 员工详情
export function fetchEmployee(id: number) {
  return client.get(`/employees/${id}`) as Promise<Employee>
}

// 我的档案（自助）
export function fetchMyProfile() {
  return client.get('/employees/me') as Promise<Employee>
}

// 创建员工（可同时创建系统账号）
export function createEmployee(data: Record<string, unknown>) {
  return client.post('/employees', data) as Promise<{
    employee: Employee
    username?: string
    default_password?: string
  }>
}

// 更新员工
export function updateEmployee(id: number, data: Record<string, unknown>) {
  return client.put(`/employees/${id}`, data) as Promise<null>
}

// 删除员工（软删除）
export function deleteEmployee(id: number) {
  return client.delete(`/employees/${id}`) as Promise<null>
}

// 部门树
export function fetchDepartments() {
  return client.get('/departments') as Promise<Department[]>
}

// 岗位列表
export function fetchPositions() {
  return client.get('/positions') as Promise<Position[]>
}
