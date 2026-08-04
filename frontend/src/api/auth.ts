import client from './client'
import type { AuthUser } from '../store/auth'

// 登录成功返回体：token + 用户简要信息
export interface LoginResult {
  token: string
  user: AuthUser
}

// /auth/me 返回体（含邮箱、角色、关联员工档案）
export interface MeResult extends AuthUser {
  email: string
  role: { id: number; name: string; code: string } | null
  employee: unknown
}

// 登录
export function login(username: string, password: string) {
  return client.post('/auth/login', { username, password }) as Promise<LoginResult>
}

// 当前登录用户信息
export function me() {
  return client.get('/auth/me') as Promise<MeResult>
}

// 修改密码
export function changePassword(oldPassword: string, newPassword: string) {
  return client.put('/auth/password', {
    old_password: oldPassword,
    new_password: newPassword,
  }) as Promise<null>
}
