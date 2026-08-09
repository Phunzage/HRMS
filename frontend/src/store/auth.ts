import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// 用户信息（登录返回的 UserBrief 与 /auth/me 返回的 MeResult 公共字段）
export interface AuthUser {
  id: number
  username: string
  full_name: string
  role_code: string
  employee_id: number | null
  email?: string
  /** 当前用户全部权限点 code（后端 /auth/me 与登录响应返回；旧持久化数据可能缺失） */
  permissions?: string[]
}

interface AuthState {
  token: string | null
  refresh_token: string | null
  user: AuthUser | null
  /** 登录成功后写入 token 对 + user */
  setAuth: (token: string, user: AuthUser, refreshToken?: string) => void
  /** 刷新用户信息（如 /auth/me 拉取） */
  setUser: (user: AuthUser) => void
  /** 仅更新 token 对（刷新令牌后调用，保留原 user） */
  setTokens: (token: string, refreshToken: string) => void
  /** 退出登录：清空 token 与 user */
  logout: () => void
}

// 全局认证状态，持久化到 localStorage（key: hrms_token）
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      refresh_token: null,
      user: null,
      setAuth: (token, user, refreshToken) => set({ token, user, refresh_token: refreshToken ?? null }),
      setUser: (user) => set({ user }),
      setTokens: (token, refreshToken) => set({ token, refresh_token: refreshToken }),
      logout: () => set({ token: null, refresh_token: null, user: null }),
    }),
    {
      name: 'hrms_token',
    },
  ),
)
