import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/auth'
import { me } from '../api/auth'

// 加载中全屏提示（避免切换路由时白屏）
function FullscreenLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-base-200">
      <span className="loading loading-spinner loading-lg text-primary" />
    </div>
  )
}

// RequireAuth 路由守卫：
// - 无 token → 重定向 /login（记录来源，登录后跳回）
// - 有 token 但 user 为空（如持久化缺失）→ 调 /auth/me 补全用户信息
export default function RequireAuth({ children }: { children: ReactNode }) {
  const token = useAuthStore((s) => s.token)
  const user = useAuthStore((s) => s.user)
  const setUser = useAuthStore((s) => s.setUser)
  const location = useLocation()
  const [loading, setLoading] = useState(false)
  const fetched = useRef(false)

  useEffect(() => {
    if (token && !user && !fetched.current) {
      fetched.current = true
      setLoading(true)
      me()
        .then(setUser)
        .catch(() => {
          // 401 已由响应拦截器处理（清 token + 跳登录）；其他错误保持现状
        })
        .finally(() => setLoading(false))
    }
  }, [token, user, setUser])

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  if (loading) {
    return <FullscreenLoading />
  }

  return <>{children}</>
}

// GuestOnly 访客守卫：已登录用户访问 /login 时重定向到首页
export function GuestOnly({ children }: { children: ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (token) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}
