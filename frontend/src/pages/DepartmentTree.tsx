import { useEffect, useState } from 'react'
import { Folder, FolderOpen } from 'lucide-react'
import {
  fetchDepartments,
  fetchAllEmployees,
  createDepartment,
  updateDepartment,
  deleteDepartment,
  type Department,
  type Employee,
} from '../api/employee'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { flattenDepartments, deptOptionLabel } from '../utils/tree'
import Modal from '../components/Modal'
import Field from '../components/FormField'

// 部门 Modal 表单数据
interface DeptForm {
  name: string
  parent_id: string
  manager_id: string
  sort_order: number
  remark: string
}

const emptyForm: DeptForm = {
  name: '',
  parent_id: '0',
  manager_id: '',
  sort_order: 0,
  remark: '',
}

// 递归渲染部门节点
interface DeptNodeProps {
  node: Department
  level: number
  canManage: boolean
  managerMap: Record<number, string>
  onAdd: (dept: Department) => void
  onEdit: (dept: Department) => void
  onDelete: (dept: Department) => void
}

function DeptNode({ node, level, canManage, managerMap, onAdd, onEdit, onDelete }: DeptNodeProps) {
  return (
    <div>
      <div
        className="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-base-200"
        style={{ paddingLeft: level * 24 + 8 }}
      >
        <span className="text-base-content/50">
          {node.children?.length ? (
            <FolderOpen className="h-4 w-4 text-base-content/50" />
          ) : (
            <Folder className="h-4 w-4 text-base-content/50" />
          )}
        </span>
        <span className="font-medium">{node.name}</span>
        <span className="text-xs text-base-content/50">
          负责人：{node.manager_id ? managerMap[node.manager_id] || `#${node.manager_id}` : '未设置'}
        </span>
        <span className="badge badge-sm badge-ghost">排序 {node.sort_order}</span>
        {canManage && (
          <span className="ml-auto flex gap-1">
            <button type="button" className="btn btn-ghost btn-xs" onClick={() => onAdd(node)}>
              新增子部门
            </button>
            <button type="button" className="btn btn-ghost btn-xs" onClick={() => onEdit(node)}>
              编辑
            </button>
            <button type="button" className="btn btn-ghost btn-xs text-error" onClick={() => onDelete(node)}>
              删除
            </button>
          </span>
        )}
      </div>
      {node.children?.map((child) => (
        <DeptNode
          key={child.id}
          node={child}
          level={level + 1}
          canManage={canManage}
          managerMap={managerMap}
          onAdd={onAdd}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}

export default function DepartmentTree() {
  const role = useAuthStore((s) => s.user?.role_code)
  const canManage = role === 'admin' || role === 'hr'

  const [tree, setTree] = useState<Department[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  const [deptOptions, setDeptOptions] = useState<{ id: number; name: string; level: number }[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])
  const [managerMap, setManagerMap] = useState<Record<number, string>>({})

  // Modal 状态
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Department | null>(null)
  const [form, setForm] = useState<DeptForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  // 加载部门树
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchDepartments()
      .then((treeData) => {
        if (cancelled) return
        setTree(treeData)
        setDeptOptions(flattenDepartments(treeData))
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取部门列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [reloadKey])

  // 加载员工列表（负责人下拉 + 名称映射）
  useEffect(() => {
    fetchAllEmployees()
      .then((res) => {
        setEmployees(res.list)
        const map: Record<number, string> = {}
        for (const e of res.list) {
          map[e.id] = e.name
        }
        setManagerMap(map)
      })
      .catch(() => toast.error('加载员工列表失败'))
  }, [])

  const reload = () => setReloadKey((k) => k + 1)

  const openCreate = (parent?: Department) => {
    setEditing(null)
    setForm({ ...emptyForm, parent_id: parent ? String(parent.id) : '0' })
    setModalOpen(true)
  }

  const openEdit = (dept: Department) => {
    setEditing(dept)
    setForm({
      name: dept.name,
      parent_id: String(dept.parent_id),
      manager_id: dept.manager_id ? String(dept.manager_id) : '',
      sort_order: dept.sort_order,
      remark: dept.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const payload = {
      name: form.name,
      parent_id: Number(form.parent_id),
      manager_id: form.manager_id ? Number(form.manager_id) : null,
      sort_order: Number(form.sort_order),
      remark: form.remark,
    }
    setSaving(true)
    try {
      if (editing) {
        await updateDepartment(editing.id, payload)
        toast.success('部门已更新')
      } else {
        await createDepartment(payload)
        toast.success('部门创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (dept: Department) => {
    if (!window.confirm(`确认删除部门「${dept.name}」？`)) return
    deleteDepartment(dept.id)
      .then(() => {
        toast.success('部门已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
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
          ) : tree.length === 0 ? (
            <p className="py-8 text-center text-base-content/50">暂无部门数据</p>
          ) : (
            <div className="flex flex-col gap-1">
              {tree.map((dept) => (
                <DeptNode
                  key={dept.id}
                  node={dept}
                  level={0}
                  canManage={canManage}
                  managerMap={managerMap}
                  onAdd={openCreate}
                  onEdit={openEdit}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {canManage && (
        <div className="flex justify-end">
          <button type="button" className="btn btn-primary btn-sm" onClick={() => openCreate()}>
            ＋ 新增顶级部门
          </button>
        </div>
      )}

      {/* 新增/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? `编辑部门 - ${editing.name}` : '新增部门'}
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
          <Field label="部门名称" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </Field>
          <Field label="上级部门">
            <select
              className="select select-bordered select-sm"
              value={form.parent_id}
              onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
            >
              <option value="0">无（顶级部门）</option>
              {deptOptions
                .filter((d) => !editing || d.id !== editing.id)
                .map((d) => (
                  <option key={d.id} value={d.id}>
                    {deptOptionLabel(d)}
                  </option>
                ))}
            </select>
          </Field>
          <Field label="部门负责人">
            <select
              className="select select-bordered select-sm"
              value={form.manager_id}
              onChange={(e) => setForm({ ...form, manager_id: e.target.value })}
            >
              <option value="">未设置</option>
              {employees.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}（{e.emp_no}）
                </option>
              ))}
            </select>
          </Field>
          <Field label="排序">
            <input
              type="number"
              className="input input-bordered input-sm"
              value={form.sort_order}
              onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })}
            />
          </Field>
          <Field label="备注" className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={2}
              value={form.remark}
              onChange={(e) => setForm({ ...form, remark: e.target.value })}
            />
          </Field>
        </div>
      </Modal>
    </div>
  )
}
