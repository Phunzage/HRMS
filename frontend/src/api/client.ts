import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '../store/auth'
import { refreshToken } from './auth'

// 后端统一响应体
interface ApiBody<T = unknown> {
  code: number
  message: string
  data: T
}

// 分页数据结构
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 请求配置扩展：标记 401 重试是否已执行（避免死循环）
declare module 'axios' {
  export interface InternalAxiosRequestConfig {
    _retry?: boolean
  }
}

// API 基址：开发环境走 vite proxy（空字符串 + /api/v1 → localhost:8080）；
// 生产/桌面端构建时通过 VITE_API_BASE 注入绝对地址（如 https://hrms.phunzage.fun）；
// 留空则走相对路径 —— 网页版/桌面端/Android 均由同源域名加载，无需注入
const API_BASE = import.meta.env.VITE_API_BASE || ''

// axios 实例：baseURL /api/v1
const client = axios.create({
  baseURL: `${API_BASE}/api/v1`,
  timeout: 15000,
})

// 桌面端（file:// 加载）下响应里的相对上传路径（/uploads/...）无法直接访问，
// 配置了绝对 API 基址时递归补全为绝对地址，保证图片/附件在桌面端可显示
function absolutizeUploads(value: unknown): unknown {
  if (typeof value === 'string' && value.startsWith('/uploads/')) {
    return `${API_BASE}${value}`
  }
  if (Array.isArray(value)) return value.map(absolutizeUploads)
  if (value && typeof value === 'object') {
    for (const key of Object.keys(value)) {
      ;(value as Record<string, unknown>)[key] = absolutizeUploads(
        (value as Record<string, unknown>)[key],
      )
    }
  }
  return value
}

// 请求拦截器：从 store 取 token，注入 Authorization
client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：
// 1. code === 0 时返回 data.data（解包统一响应）
// 2. 业务错误（code !== 0）与网络错误统一 reject(Error(message))
// 3. 401 时若有 refresh_token 先刷新并重放原请求（最多 1 次），失败则清空跳登录页
client.interceptors.response.use(
  (res) => {
    const body = res.data as ApiBody
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        return absolutizeUploads(body.data) as unknown as typeof res
      }
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return absolutizeUploads(res.data) as unknown as typeof res
  },
  async (error: AxiosError<ApiBody>) => {
    const status = error.response?.status
    const config = error.config as (InternalAxiosRequestConfig & { _retry?: boolean }) | undefined

    // 401 且尚未重试：尝试用 refresh_token 换新 token 并重放原请求
    // 刷新接口自身（/auth/refresh）失败时直接退出，避免无限递归
    const isRefreshEndpoint = config?.url?.includes('/auth/refresh')
    if (status === 401 && config && !config._retry && !isRefreshEndpoint) {
      const refresh_token = useAuthStore.getState().refresh_token
      if (refresh_token) {
        config._retry = true
        try {
          const res = await refreshToken(refresh_token)
          useAuthStore.getState().setTokens(res.token, res.refresh_token)
          config.headers.Authorization = `Bearer ${res.token}`
          return client(config)
        } catch {
          // 刷新失败（token 失效/网络异常）→ 走下方登出逻辑
        }
      }
    }

    // 仅 401（含刷新失败/无 refresh_token）才清空登录态并跳登录页；
    // 其他 HTTP 错误（400/403/404/500 等）不登出不跳转，仅把错误抛给调用方，
    // 由页面自身的 catch/toast 处理，避免误登出
    if (status === 401) {
      useAuthStore.getState().logout()
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    const msg = error.response?.data?.message || error.message || '网络错误'
    return Promise.reject(new Error(msg))
  },
)

export default client
