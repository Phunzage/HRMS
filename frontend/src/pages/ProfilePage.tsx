import { useEffect, useState, type FormEvent } from 'react'
import { me, changePassword, type MeResult } from '../api/auth'
import { fetchMyProfile, type Employee } from '../api/employee'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { EMPLOYEE_STATUS, GENDER } from '../utils/status'
import { formatDate } from '../utils/format'
import Field from '../components/FormField'

// 角色编码 → 中文名称
const roleNames: Record<string, string> = {
  admin: '管理员',
  hr: '人事',
  manager: '经理',
  employee: '员工',
}

// 修改密码表单
interface PwdForm {
  old_password: string
  new_password: string
  confirm_password: string
}

const emptyPwd: PwdForm = {
  old_password: '',
  new_password: '',
  confirm_password: '',
}

export default function ProfilePage() {
  const user = useAuthStore((s) => s.user)

  const [meInfo, setMeInfo] = useState<MeResult | null>(null)
  const [profile, setProfile] = useState<Employee | null>(null)
  const [profileError, setProfileError] = useState('')
  const [loading, setLoading] = useState(true)

  const [pwd, setPwd] = useState<PwdForm>(emptyPwd)
  const [pwdError, setPwdError] = useState('')
  const [saving, setSaving] = useState(false)

  // 加载用户信息 + 员工档案
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.allSettled([me(), fetchMyProfile()])
      .then(([meRes, profileRes]) => {
        if (cancelled) return
        if (meRes.status === 'fulfilled') setMeInfo(meRes.value)
        if (profileRes.status === 'fulfilled') {
          setProfile(profileRes.value)
        } else {
          setProfileError('当前账号未关联员工档案')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const handleChangePassword = async (e: FormEvent) => {
    e.preventDefault()
    setPwdError('')
    if (pwd.new_password.length < 6) {
      setPwdError('新密码长度不能少于 6 位')
      return
    }
    if (pwd.new_password !== pwd.confirm_password) {
      setPwdError('两次输入的新密码不一致')
      return
    }
    setSaving(true)
    try {
      await changePassword(pwd.old_password, pwd.new_password)
      toast.success('密码修改成功')
      setPwd(emptyPwd)
    } catch (err) {
      setPwdError(err instanceof Error ? err.message : '修改密码失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 账号信息 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title text-base">账号信息</h3>
          <div className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
            <div>
              <div className="text-base-content/50">用户名</div>
              <div className="font-medium">{meInfo?.username ?? user?.username ?? '-'}</div>
            </div>
            <div>
              <div className="text-base-content/50">姓名</div>
              <div className="font-medium">{meInfo?.full_name ?? user?.full_name ?? '-'}</div>
            </div>
            <div>
              <div className="text-base-content/50">角色</div>
              <div className="font-medium">
                {meInfo?.role?.name ?? roleNames[user?.role_code ?? ''] ?? user?.role_code ?? '-'}
              </div>
            </div>
            <div>
              <div className="text-base-content/50">邮箱</div>
              <div className="font-medium">{meInfo?.email || '-'}</div>
            </div>
          </div>
        </div>
      </div>

      {/* 员工档案 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title text-base">员工档案</h3>
          {profile ? (
            <div className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
              <div>
                <div className="text-base-content/50">工号</div>
                <div className="font-mono font-medium">{profile.emp_no}</div>
              </div>
              <div>
                <div className="text-base-content/50">姓名</div>
                <div className="font-medium">{profile.name}</div>
              </div>
              <div>
                <div className="text-base-content/50">性别</div>
                <div>{GENDER[profile.gender] ?? profile.gender}</div>
              </div>
              <div>
                <div className="text-base-content/50">状态</div>
                <span className={`badge badge-sm ${EMPLOYEE_STATUS[profile.status]?.badge ?? 'badge-ghost'}`}>
                  {EMPLOYEE_STATUS[profile.status]?.label ?? profile.status}
                </span>
              </div>
              <div>
                <div className="text-base-content/50">部门</div>
                <div>{profile.department_name || '-'}</div>
              </div>
              <div>
                <div className="text-base-content/50">岗位</div>
                <div>{profile.position_name || '-'}</div>
              </div>
              <div>
                <div className="text-base-content/50">手机</div>
                <div>{profile.phone || '-'}</div>
              </div>
              <div>
                <div className="text-base-content/50">邮箱</div>
                <div>{profile.email || '-'}</div>
              </div>
              <div>
                <div className="text-base-content/50">入职日期</div>
                <div>{formatDate(profile.hire_date)}</div>
              </div>
              <div>
                <div className="text-base-content/50">出生日期</div>
                <div>{formatDate(profile.birth_date)}</div>
              </div>
              <div className="col-span-2">
                <div className="text-base-content/50">地址</div>
                <div>{profile.address || '-'}</div>
              </div>
            </div>
          ) : (
            <p className="text-sm text-base-content/50">{profileError || '暂无员工档案信息'}</p>
          )}
        </div>
      </div>

      {/* 修改密码 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title text-base">修改密码</h3>
          <form onSubmit={handleChangePassword} className="flex max-w-md flex-col gap-3">
            {pwdError && <div className="alert alert-error py-2 text-sm">{pwdError}</div>}
            <Field label="旧密码" required>
              <input
                type="password"
                className="input input-bordered input-sm"
                value={pwd.old_password}
                onChange={(e) => setPwd({ ...pwd, old_password: e.target.value })}
                autoComplete="current-password"
                required
              />
            </Field>
            <Field label="新密码" required>
              <input
                type="password"
                className="input input-bordered input-sm"
                value={pwd.new_password}
                onChange={(e) => setPwd({ ...pwd, new_password: e.target.value })}
                autoComplete="new-password"
                placeholder="至少 6 位"
                required
              />
            </Field>
            <Field label="确认新密码" required>
              <input
                type="password"
                className="input input-bordered input-sm"
                value={pwd.confirm_password}
                onChange={(e) => setPwd({ ...pwd, confirm_password: e.target.value })}
                autoComplete="new-password"
                required
              />
            </Field>
            <div className="mt-1">
              <button type="submit" className="btn btn-primary btn-sm" disabled={saving}>
                {saving && <span className="loading loading-spinner loading-sm" />}
                修改密码
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
