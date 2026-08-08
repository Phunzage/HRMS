import client from './client'
import type { PageResult } from './client'

// 用户列表项（附带角色名称）
export interface UserItem {
  id: number
  username: string
  email: string
  full_name: string
  role_id: number
  role_name: string
  employee_id: number | null
  status: number // 1 启用 / 0 禁用
  created_at: string
  updated_at: string
}

// 角色下拉项（/users/roles 返回）
export interface RoleOption {
  id: number
  name: string
  code: string
  description: string
}

// 用户分页列表
export function listUsers(
  params: {
    keyword?: string
    role_id?: number
    status?: number
    page?: number
    page_size?: number
  } = {},
) {
  return client.get('/users', { params }) as Promise<PageResult<UserItem>>
}

// 创建用户
export function createUser(data: {
  username: string
  password: string
  full_name?: string
  email?: string
  role_id: number
}) {
  return client.post('/users', data) as Promise<UserItem>
}

// 更新用户基本信息
export function updateUser(
  id: number,
  data: { full_name?: string; email?: string; role_id?: number; status?: number },
) {
  return client.put(`/users/${id}`, data) as Promise<null>
}

// 重置用户密码
export function resetPassword(id: number, password: string) {
  return client.put(`/users/${id}/password`, { password }) as Promise<null>
}

// 删除用户（软删除）
export function deleteUser(id: number) {
  return client.delete(`/users/${id}`) as Promise<null>
}

// 全部角色（供下拉选择）
export function listRoles() {
  return client.get('/users/roles') as Promise<RoleOption[]>
}
