import { useState, type FormEvent } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { login } from '../api/auth'
import { useAuthStore } from '../store/auth'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const setAuth = useAuthStore((s) => s.setAuth)
  const navigate = useNavigate()
  const location = useLocation()

  // 登录成功后的跳转目标（优先返回被守卫拦截前的页面）
  const from = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname || '/'

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const data = await login(username.trim(), password)
      setAuth(data.token, data.user)
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-base-200 p-4">
      <div className="card w-full max-w-sm bg-base-100 shadow-xl">
        <div className="card-body">
          <h2 className="card-title justify-center text-2xl">HRMS 人力资源管理系统</h2>
          <p className="text-center text-sm text-base-content/60">请登录您的账号</p>

          <form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-4">
            {error && <div className="alert alert-error py-2 text-sm">{error}</div>}

            <label className="form-control w-full">
              <div className="label">
                <span className="label-text">用户名</span>
              </div>
              <input
                type="text"
                className="input input-bordered w-full"
                placeholder="请输入用户名"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
              />
            </label>

            <label className="form-control w-full">
              <div className="label">
                <span className="label-text">密码</span>
              </div>
              <input
                type="password"
                className="input input-bordered w-full"
                placeholder="请输入密码"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
              />
            </label>

            <button type="submit" className="btn btn-primary mt-2" disabled={loading}>
              {loading && <span className="loading loading-spinner loading-sm" />}
              {loading ? '登录中...' : '登录'}
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
