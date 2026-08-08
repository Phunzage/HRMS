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

// 岗位（列表附带所属部门名称）
export interface Position {
  id: number
  name: string
  code: string
  department_id: number
  level: number
  remark: string
  department_name?: string
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

// 自助更新本人头像（无需 employee:write 权限，供员工个人中心使用）
export function updateMyAvatar(avatar: string) {
  return client.put('/employees/me/avatar', { avatar }) as Promise<null>
}

// 删除员工（软删除）
export function deleteEmployee(id: number) {
  return client.delete(`/employees/${id}`) as Promise<null>
}

// 部门树
export function fetchDepartments() {
  return client.get('/departments') as Promise<Department[]>
}

// 创建部门
export function createDepartment(data: {
  name: string
  parent_id?: number
  manager_id?: number | null
  sort_order?: number
  remark?: string
}) {
  return client.post('/departments', data) as Promise<Department>
}

// 更新部门（manager_id 传 null 表示清空负责人）
export function updateDepartment(id: number, data: Record<string, unknown>) {
  return client.put(`/departments/${id}`, data) as Promise<null>
}

// 删除部门（有子部门或员工时后端拒绝）
export function deleteDepartment(id: number) {
  return client.delete(`/departments/${id}`) as Promise<null>
}

// 岗位分页列表（后端返回分页结构）
export function fetchPositions(
  params: { keyword?: string; department_id?: number; page?: number; page_size?: number } = {},
) {
  return client.get('/positions', { params }) as Promise<PageResult<Position>>
}

// 全部岗位（供下拉选项，拉取一页大分页）
export function fetchAllPositions() {
  return fetchPositions({ page: 1, page_size: 1000 })
}

// 全部在职员工（供下拉选项，拉取一页大分页）
export function fetchAllEmployees() {
  return fetchEmployees({ page: 1, page_size: 1000 })
}

// 创建岗位
export function createPosition(data: {
  name: string
  code: string
  department_id: number
  level?: number
  remark?: string
}) {
  return client.post('/positions', data) as Promise<Position>
}

// 更新岗位
export function updatePosition(id: number, data: Record<string, unknown>) {
  return client.put(`/positions/${id}`, data) as Promise<null>
}

// 删除岗位（被员工占用时后端拒绝）
export function deletePosition(id: number) {
  return client.delete(`/positions/${id}`) as Promise<null>
}
