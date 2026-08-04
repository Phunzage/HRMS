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
}

interface AuthState {
  token: string | null
  user: AuthUser | null
  /** 登录成功后写入 token + user */
  setAuth: (token: string, user: AuthUser) => void
  /** 刷新用户信息（如 /auth/me 拉取） */
  setUser: (user: AuthUser) => void
  /** 退出登录：清空 token 与 user */
  logout: () => void
}

// 全局认证状态，持久化到 localStorage（key: hrms_token）
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setAuth: (token, user) => set({ token, user }),
      setUser: (user) => set({ user }),
      logout: () => set({ token: null, user: null }),
    }),
    {
      name: 'hrms_token',
    },
  ),
)
