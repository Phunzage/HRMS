import axios, { AxiosError } from 'axios'
import { useAuthStore } from '../store/auth'

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

// axios 实例：baseURL /api/v1（开发环境由 vite proxy 转发到 8080）
const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

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
// 3. 401 时清空 token 并跳转登录页
client.interceptors.response.use(
  (res) => {
    const body = res.data as ApiBody
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        return body.data as unknown as typeof res
      }
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return res.data as unknown as typeof res
  },
  (error: AxiosError<ApiBody>) => {
    if (error.response?.status === 401) {
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
