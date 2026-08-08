import client from './client'
import type { AuthUser } from '../store/auth'

// 登录成功返回体：token 对 + 用户简要信息
export interface LoginResult {
  token: string
  refresh_token: string
  expires_in: number
  user: AuthUser
}

// /auth/me 返回体（含邮箱、角色、关联员工档案）
export interface MeResult extends AuthUser {
  email: string
  role: { id: number; name: string; code: string } | null
  employee: unknown
}

// 图形验证码
export interface CaptchaResult {
  captcha_id: string
  captcha_image: string // base64 data URI，可直接作为 img src
}

// 获取图形验证码（无需登录）
export function getCaptcha() {
  return client.get('/auth/captcha') as Promise<CaptchaResult>
}

// 登录（需携带验证码）
export function login(username: string, password: string, captchaId: string, captchaCode: string) {
  return client.post('/auth/login', {
    username,
    password,
    captcha_id: captchaId,
    captcha_code: captchaCode,
  }) as Promise<LoginResult>
}

// 用 refresh_token 换取新的 token 对（公开接口）
export function refreshToken(refreshToken: string) {
  return client.post('/auth/refresh', { refresh_token: refreshToken }) as Promise<LoginResult>
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
