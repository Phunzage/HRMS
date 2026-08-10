import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { syncTokenToNative } from '../api/tokenBridge'

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

// token 变更时同步到原生层（Android TokenStore 插件写 SharedPreferences，供后台 NotifyWorker 读取）；
// 覆盖登录、刷新令牌、退出登录与持久化 rehydrate 恢复三种路径，原生平台才真正写入
useAuthStore.subscribe((state, prev) => {
  if (state.token !== prev.token) {
    syncTokenToNative(state.token)
  }
})
