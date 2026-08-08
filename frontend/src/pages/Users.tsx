import { useEffect, useState } from 'react'
import {
  listUsers,
  createUser,
  updateUser,
  resetPassword,
  deleteUser,
  listRoles,
  type UserItem,
  type RoleOption,
} from '../api/user'
import { fetchDepartments } from '../api/employee'
import { flattenDepartments, deptOptionLabel, type DeptOption } from '../utils/tree'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { formatDateTime } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'

const PAGE_SIZE = 10

// 用户表单数据（role_id / dept_scope_id 用字符串便于下拉受控，提交时转数字）
interface UserForm {
  username: string
  password: string
  full_name: string
  email: string
  role_id: string
  dept_scope_id: string
}

const emptyForm: UserForm = {
  username: '',
  password: '',
  full_name: '',
  email: '',
  role_id: '',
  dept_scope_id: '',
}

// 状态徽章：1 启用（绿）/ 0 禁用（灰）
function StatusCell({ status }: { status: number }) {
  return (
    <span className={`badge badge-sm ${status === 1 ? 'badge-success' : 'badge-ghost'}`}>
      {status === 1 ? '启用' : '禁用'}
    </span>
  )
}

export default function UsersPage() {
  const role = useAuthStore((s) => s.user?.role_code)
  const canManage = role === 'admin' || role === 'hr'

  // 筛选条件
  const [keyword, setKeyword] = useState('')
  const [search, setSearch] = useState('')
  const [roleId, setRoleId] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<UserItem> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // 角色下拉
  const [roles, setRoles] = useState<RoleOption[]>([])
  // 负责部门下拉（部门树扁平化）
  const [deptOptions, setDeptOptions] = useState<DeptOption[]>([])

  // 新增/编辑弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<UserItem | null>(null)
  const [form, setForm] = useState<UserForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  // 重置密码弹窗
  const [resetOpen, setResetOpen] = useState(false)
  const [resetUser, setResetUser] = useState<UserItem | null>(null)
  const [resetPwd, setResetPwd] = useState('')
  const [resetting, setResetting] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    listUsers({
      keyword: keyword || undefined,
      role_id: roleId ? Number(roleId) : undefined,
      status: status === '' ? undefined : Number(status),
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取用户列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [keyword, roleId, status, page, reloadKey])

  // 角色下拉数据（admin/hr 均需，列表展示角色名）；加载失败时静默降级为空数组
  useEffect(() => {
    listRoles()
      .then((res) => setRoles(res ?? []))
      .catch(() => setRoles([]))
  }, [])

  // 负责部门下拉数据（部门树扁平化，供 hr 数据范围选择）；加载失败时静默降级为空数组
  useEffect(() => {
    fetchDepartments()
      .then((tree) => setDeptOptions(flattenDepartments(tree)))
      .catch(() => setDeptOptions([]))
  }, [])

  const reload = () => setReloadKey((k) => k + 1)

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  const openEdit = (user: UserItem) => {
    setEditing(user)
    setForm({
      username: user.username,
      password: '',
      full_name: user.full_name,
      email: user.email,
      role_id: String(user.role_id),
      dept_scope_id: user.dept_scope_id ? String(user.dept_scope_id) : '',
    })
    setModalOpen(true)
  }

  // 当前表单所选角色的 code（用于判断是否显示「负责部门」）
  const selectedRoleCode = roles.find((r) => r.id === Number(form.role_id))?.code

  const handleSave = async () => {
    if (!form.username.trim()) {
      toast.error('请输入用户名')
      return
    }
    if (!form.role_id) {
      toast.error('请选择角色')
      return
    }
    if (!editing && form.password.length < 6) {
      toast.error('密码长度不能少于6位')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        const payload: { full_name: string; email: string; role_id: number; dept_scope_id?: number | null } = {
          full_name: form.full_name,
          email: form.email,
          role_id: Number(form.role_id),
        }
        // 所选角色为 hr 时提交数据范围部门（可清空）
        if (selectedRoleCode === 'hr') {
          payload.dept_scope_id = form.dept_scope_id ? Number(form.dept_scope_id) : null
        }
        await updateUser(editing.id, payload)
        toast.success('用户已更新')
      } else {
        await createUser({
          username: form.username.trim(),
          password: form.password,
          full_name: form.full_name,
          email: form.email,
          role_id: Number(form.role_id),
        })
        toast.success('用户创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const openReset = (user: UserItem) => {
    setResetUser(user)
    setResetPwd('')
    setResetOpen(true)
  }

  const handleReset = async () => {
    if (!resetUser) return
    if (resetPwd.length < 6) {
      toast.error('密码长度不能少于6位')
      return
    }
    setResetting(true)
    try {
      await resetPassword(resetUser.id, resetPwd)
      toast.success('密码已重置')
      setResetOpen(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '重置失败')
    } finally {
      setResetting(false)
    }
  }

  const handleDelete = (user: UserItem) => {
    if (!window.confirm(`删除后「${user.username}」该账号将无法登录，确认删除？`)) return
    deleteUser(user.id)
      .then(() => {
        toast.success('用户已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          <div className="flex min-w-64 flex-1 gap-2">
            <input
              type="text"
              className="input input-bordered input-sm w-full"
              placeholder="搜索用户名 / 姓名"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  setKeyword(search.trim())
                  setPage(1)
                }
              }}
            />
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={() => {
                setKeyword(search.trim())
                setPage(1)
              }}
            >
              搜索
            </button>
          </div>
          <select
            className="select select-bordered select-sm w-40"
            value={roleId}
            onChange={(e) => {
              setRoleId(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部角色</option>
            {roles.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
          <select
            className="select select-bordered select-sm w-32"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部状态</option>
            <option value="1">启用</option>
            <option value="0">禁用</option>
          </select>
          {canManage && (
            <div className="ml-auto">
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                ＋ 新增用户
              </button>
            </div>
          )}
        </div>
      </div>

      {/* 用户表格 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4">
          {error ? (
            <div className="alert alert-error">
              <span>{error}</span>
            </div>
          ) : loading ? (
            <div className="flex items-center justify-center py-16">
              <span className="loading loading-spinner loading-lg text-primary" />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>用户名</th>
                    <th>姓名</th>
                    <th>邮箱</th>
                    <th>角色</th>
                    <th>负责部门</th>
                    <th>状态</th>
                    <th>创建时间</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((user) => (
                    <tr key={user.id}>
                      <td className="font-mono text-xs font-medium">{user.username}</td>
                      <td className="font-medium">{user.full_name || '-'}</td>
                      <td className="max-w-48 truncate">{user.email || '-'}</td>
                      <td>
                        <span className="badge badge-sm badge-primary">{user.role_name || '-'}</span>
                      </td>
                      <td className="text-base-content/60">{user.dept_scope_name || '-'}</td>
                      <td>
                        <StatusCell status={user.status} />
                      </td>
                      <td className="text-base-content/60">{formatDateTime(user.created_at)}</td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(user)}>
                              编辑
                            </button>
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openReset(user)}>
                              重置密码
                            </button>
                            <button
                              type="button"
                              className="btn btn-ghost btn-xs text-error"
                              onClick={() => handleDelete(user)}
                            >
                              删除
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canManage ? 8 : 7} className="py-8 text-center text-base-content/50">
                        暂无用户
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
          {data && (
            <div className="mt-2">
              <Pagination page={page} pageSize={PAGE_SIZE} total={data.total} onChange={setPage} />
            </div>
          )}
        </div>
      </div>

      {/* 新增/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? `编辑用户 - ${editing.username}` : '新增用户'}
        onClose={() => setModalOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setModalOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={saving} onClick={handleSave}>
              {saving && <span className="loading loading-spinner loading-sm" />}
              保存
            </button>
          </>
        }
      >
        <div className="grid grid-cols-2 gap-3">
          <Field label="用户名" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.username}
              disabled={!!editing}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="登录账号"
              required
            />
          </Field>
          {!editing && (
            <Field label="密码" required>
              <input
                type="password"
                className="input input-bordered input-sm"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                placeholder="至少 6 位"
                required
              />
            </Field>
          )}
          <Field label="姓名">
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.full_name}
              onChange={(e) => setForm({ ...form, full_name: e.target.value })}
            />
          </Field>
          <Field label="邮箱">
            <input
              type="email"
              className="input input-bordered input-sm"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </Field>
          <Field label="角色" required>
            <select
              className="select select-bordered select-sm"
              value={form.role_id}
              onChange={(e) => setForm({ ...form, role_id: e.target.value })}
              required
            >
              <option value="">请选择角色</option>
              {roles.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}（{r.code}）
                </option>
              ))}
            </select>
          </Field>
          {editing && selectedRoleCode === 'hr' && (
            <Field label="负责部门">
              <select
                className="select select-bordered select-sm"
                value={form.dept_scope_id}
                onChange={(e) => setForm({ ...form, dept_scope_id: e.target.value })}
              >
                <option value="">全部部门（不限）</option>
                {deptOptions.map((d) => (
                  <option key={d.id} value={d.id}>
                    {deptOptionLabel(d)}
                  </option>
                ))}
              </select>
            </Field>
          )}
        </div>
      </Modal>

      {/* 重置密码 Modal */}
      <Modal
        open={resetOpen}
        title={`重置密码 - ${resetUser?.username ?? ''}`}
        onClose={() => setResetOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setResetOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={resetting} onClick={handleReset}>
              {resetting && <span className="loading loading-spinner loading-sm" />}
              确认重置
            </button>
          </>
        }
      >
        <Field label="新密码" required>
          <input
            type="password"
            className="input input-bordered input-sm w-full"
            value={resetPwd}
            onChange={(e) => setResetPwd(e.target.value)}
            placeholder="至少 6 位"
            autoFocus
            required
          />
        </Field>
      </Modal>
    </div>
  )
}
