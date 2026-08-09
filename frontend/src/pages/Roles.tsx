import { useEffect, useState } from 'react'
import {
  listRoles,
  listPermissions,
  createRole,
  updateRole,
  deleteRole,
  type RoleItem,
  type Permission,
  type PermissionGroup,
} from '../api/role'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import Modal from '../components/Modal'
import Field from '../components/FormField'

// 模块编码 → 中文分组名（组织/人事/考勤/审批/请假/薪酬/招聘/系统 归组展示）
const MODULE_LABELS: Record<string, string> = {
  department: '组织架构',
  position: '组织架构',
  employee: '人事管理',
  attendance: '考勤管理',
  approval: '审批',
  leave: '请假管理',
  salary: '薪酬管理',
  recruitment: '招聘管理',
  dashboard: '系统管理',
  user: '系统管理',
  role: '系统管理',
  audit: '系统管理',
}

// 分组展示顺序
const MODULE_ORDER = ['组织架构', '人事管理', '考勤管理', '审批', '请假管理', '薪酬管理', '招聘管理', '系统管理']

// 将后端按模块编码返回的分组，合并为按中文分组的展示结构
function buildGroups(groups: PermissionGroup[]) {
  const merged: { label: string; items: Permission[] }[] = []
  for (const g of groups) {
    const label = MODULE_LABELS[g.module] ?? g.module
    const existing = merged.find((m) => m.label === label)
    if (existing) {
      existing.items.push(...g.items)
    } else {
      merged.push({ label, items: [...g.items] })
    }
  }
  merged.sort((a, b) => {
    const ia = MODULE_ORDER.indexOf(a.label)
    const ib = MODULE_ORDER.indexOf(b.label)
    return (ia === -1 ? MODULE_ORDER.length : ia) - (ib === -1 ? MODULE_ORDER.length : ib)
  })
  return merged
}

// 角色表单数据
interface RoleForm {
  name: string
  code: string
  description: string
  permissions: string[]
}

const emptyForm: RoleForm = {
  name: '',
  code: '',
  description: '',
  permissions: [],
}

export default function RolesPage() {
  const role = useAuthStore((s) => s.user?.role_code)
  const canManage = role === 'admin' // 仅 admin 可新增/编辑/删除，hr 只读

  const [roles, setRoles] = useState<RoleItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  // 权限分组（勾选面板）
  const [permGroups, setPermGroups] = useState<ReturnType<typeof buildGroups>>([])

  // 弹窗
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<RoleItem | null>(null)
  const [form, setForm] = useState<RoleForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    listRoles()
      .then((res) => {
        if (!cancelled) setRoles(res ?? [])
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取角色列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [reloadKey])

  // 仅 admin 需要权限点面板，加载一次
  useEffect(() => {
    if (!canManage) return
    listPermissions()
      .then((groups) => setPermGroups(buildGroups(groups)))
      .catch(() => toast.error('加载权限点列表失败'))
  }, [canManage])

  const reload = () => setReloadKey((k) => k + 1)

  // 勾选/取消一个权限点
  const togglePermission = (code: string) => {
    setForm((f) => ({
      ...f,
      permissions: f.permissions.includes(code)
        ? f.permissions.filter((c) => c !== code)
        : [...f.permissions, code],
    }))
  }

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  const openEdit = (item: RoleItem) => {
    setEditing(item)
    setForm({
      name: item.name,
      code: item.code,
      description: item.description,
      permissions: [...item.permissions],
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    if (!form.name.trim()) {
      toast.error('请输入角色名称')
      return
    }
    if (!form.code.trim()) {
      toast.error('请输入角色编码')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await updateRole(editing.id, {
          name: form.name.trim(),
          description: form.description,
          permissions: form.permissions,
        })
        toast.success('角色已更新')
      } else {
        await createRole({
          name: form.name.trim(),
          code: form.code.trim(),
          description: form.description,
          permissions: form.permissions,
        })
        toast.success('角色创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (item: RoleItem) => {
    if (!window.confirm(`确认删除角色「${item.name}」？删除后拥有该角色的账号将无法登录。`)) return
    deleteRole(item.id)
      .then(() => {
        toast.success('角色已删除')
        reload()
      })
      .catch((err: unknown) => {
        // 角色下有用户时后端返回 400，按要求用 alert 展示
        window.alert(err instanceof Error ? err.message : '删除失败')
      })
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-center gap-3 p-4">
          <span className="text-sm text-base-content/60">共 {roles.length} 个角色</span>
          {canManage && (
            <div className="ml-auto">
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                ＋ 新增角色
              </button>
            </div>
          )}
        </div>
      </div>

      {/* 角色表格 */}
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
                    <th>角色名</th>
                    <th>编码</th>
                    <th>描述</th>
                    <th className="text-center">权限数</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {roles.map((item) => (
                    <tr key={item.id}>
                      <td className="font-medium">{item.name}</td>
                      <td className="font-mono text-xs">{item.code}</td>
                      <td className="max-w-64 truncate">{item.description || '-'}</td>
                      <td className="text-center">
                        <span className="badge badge-sm badge-ghost">{item.permissions.length}</span>
                      </td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(item)}>
                              编辑
                            </button>
                            {item.code !== 'admin' && (
                              <button
                                type="button"
                                className="btn btn-ghost btn-xs text-error"
                                onClick={() => handleDelete(item)}
                              >
                                删除
                              </button>
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!roles.length && (
                    <tr>
                      <td colSpan={canManage ? 5 : 4} className="py-8 text-center text-base-content/50">
                        暂无角色
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* 新增/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? `编辑角色 - ${editing.name}` : '新增角色'}
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
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <Field label="角色名称" required>
              <input
                type="text"
                className="input input-bordered input-sm"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                required
              />
            </Field>
            <Field label="角色编码" required>
              <input
                type="text"
                className="input input-bordered input-sm"
                value={form.code}
                disabled={!!editing}
                placeholder="如 finance"
                onChange={(e) => setForm({ ...form, code: e.target.value })}
                required
              />
            </Field>
            <Field label="描述" className="col-span-2">
              <input
                type="text"
                className="input input-bordered input-sm"
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </Field>
          </div>

          {/* 权限勾选区：按模块分组 */}
          <div>
            <div className="mb-2 text-sm font-semibold text-base-content/80">权限</div>
            {permGroups.length === 0 ? (
              <div className="text-sm text-base-content/50">正在加载权限点…</div>
            ) : (
              <div className="flex flex-col gap-3">
                {permGroups.map((group) => (
                  <div key={group.label} className="rounded-box border border-base-300 p-3">
                    <div className="mb-2 text-sm font-semibold">{group.label}</div>
                    <div className="grid grid-cols-2 gap-2">
                      {group.items.map((p) => (
                        <label key={p.code} className="flex cursor-pointer items-center gap-2">
                          <input
                            type="checkbox"
                            className="checkbox checkbox-sm checkbox-primary"
                            checked={form.permissions.includes(p.code)}
                            onChange={() => togglePermission(p.code)}
                          />
                          <span>{p.name}</span>
                          <span className="font-mono text-xs text-base-content/50">{p.code}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </Modal>
    </div>
  )
}
