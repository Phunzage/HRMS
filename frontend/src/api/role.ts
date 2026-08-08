import client from './client'

// 权限点
export interface Permission {
  id: number
  name: string
  code: string
  module: string
}

// 权限点分组（模块 → 权限点数组）
export interface PermissionGroup {
  module: string
  items: Permission[]
}

// 角色列表项（含该角色已授予的权限点 code 数组）
export interface RoleItem {
  id: number
  name: string
  code: string
  description: string
  permissions: string[]
  created_at: string
  updated_at: string
}

// 角色列表（含权限数组，前端用于勾选回显）
export function listRoles() {
  return client.get('/roles') as Promise<RoleItem[]>
}

// 全部权限点，按模块分组
export function listPermissions() {
  return client.get('/roles/permissions') as Promise<PermissionGroup[]>
}

// 创建角色
export function createRole(data: {
  name: string
  code: string
  description?: string
  permissions: string[]
}) {
  return client.post('/roles', data) as Promise<RoleItem>
}

// 更新角色（permissions 全量替换）
export function updateRole(
  id: number,
  data: { name?: string; description?: string; permissions: string[] },
) {
  return client.put(`/roles/${id}`, data) as Promise<null>
}

// 删除角色：角色下有用户时后端返回 400
export function deleteRole(id: number) {
  return client.delete(`/roles/${id}`) as Promise<null>
}
