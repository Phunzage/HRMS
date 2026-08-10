import { useEffect, useState } from 'react'
import {
  fetchPositions,
  fetchDepartments,
  createPosition,
  updatePosition,
  deletePosition,
  type Position,
  type Department,
} from '../api/employee'
import type { PageResult } from '../api/client'
import { usePerm } from '../hooks/usePerm'
import { toast } from '../store/toast'
import { flattenDepartments, deptOptionLabel } from '../utils/tree'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'

const PAGE_SIZE = 10

// 岗位表单数据
interface PositionForm {
  name: string
  code: string
  department_id: string
  level: number
  remark: string
}

const emptyForm: PositionForm = {
  name: '',
  code: '',
  department_id: '',
  level: 1,
  remark: '',
}

export default function PositionPage() {
  // 岗位增删改需 position:write 权限点（种子仅 hr，可在角色管理动态调整）
  const { has } = usePerm()
  const canManage = has('position:write')

  const [keyword, setKeyword] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [departmentId, setDepartmentId] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<Position> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [deptOptions, setDeptOptions] = useState<{ id: number; name: string; level: number }[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Position | null>(null)
  const [form, setForm] = useState<PositionForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchPositions({
      keyword: keyword || undefined,
      department_id: departmentId ? Number(departmentId) : undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取岗位列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [keyword, departmentId, page, reloadKey])

  useEffect(() => {
    // 非关键下拉：部门加载失败时静默降级为空数组
    fetchDepartments()
      .then((tree: Department[]) => setDeptOptions(flattenDepartments(tree)))
      .catch(() => setDeptOptions([]))
  }, [])

  const reload = () => setReloadKey((k) => k + 1)

  const handleSearch = () => {
    setKeyword(searchInput.trim())
    setPage(1)
  }

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm)
    setModalOpen(true)
  }

  const openEdit = (pos: Position) => {
    setEditing(pos)
    setForm({
      name: pos.name,
      code: pos.code,
      department_id: String(pos.department_id),
      level: pos.level,
      remark: pos.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const payload = {
      name: form.name,
      code: form.code,
      department_id: Number(form.department_id),
      level: Number(form.level),
      remark: form.remark,
    }
    setSaving(true)
    try {
      if (editing) {
        await updatePosition(editing.id, payload)
        toast.success('岗位已更新')
      } else {
        await createPosition(payload)
        toast.success('岗位创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (pos: Position) => {
    if (!window.confirm(`确认删除岗位「${pos.name}」？`)) return
    deletePosition(pos.id)
      .then(() => {
        toast.success('岗位已删除')
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
              placeholder="搜索岗位名称 / 编码"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <button type="button" className="btn btn-primary btn-sm" onClick={handleSearch}>
              搜索
            </button>
          </div>
          <select
            className="select select-bordered select-sm w-44"
            value={departmentId}
            onChange={(e) => {
              setDepartmentId(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部部门</option>
            {deptOptions.map((d) => (
              <option key={d.id} value={d.id}>
                {deptOptionLabel(d)}
              </option>
            ))}
          </select>
          {canManage && (
            <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
              ＋ 新增岗位
            </button>
          )}
        </div>
      </div>

      {/* 表格 */}
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
                    <th>岗位名称</th>
                    <th>编码</th>
                    <th>所属部门</th>
                    <th>级别</th>
                    <th>备注</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((pos) => (
                    <tr key={pos.id}>
                      <td className="font-medium">{pos.name}</td>
                      <td className="font-mono text-xs">{pos.code}</td>
                      <td>{pos.department_name || '-'}</td>
                      <td>{pos.level}</td>
                      <td className="max-w-40 truncate text-base-content/60">{pos.remark || '-'}</td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex flex-wrap justify-end gap-1">
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(pos)}>
                              编辑
                            </button>
                            <button type="button" className="btn btn-ghost btn-xs text-error" onClick={() => handleDelete(pos)}>
                              删除
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canManage ? 6 : 5} className="py-8 text-center text-base-content/50">
                        暂无岗位数据
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
        title={editing ? `编辑岗位 - ${editing.name}` : '新增岗位'}
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
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="岗位名称" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </Field>
          <Field label="岗位编码" required>
            <input
              type="text"
              className="input input-bordered input-sm"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
              placeholder="如 backend_engineer"
              required
            />
          </Field>
          <Field label="所属部门" required>
            <select
              className="select select-bordered select-sm"
              value={form.department_id}
              onChange={(e) => setForm({ ...form, department_id: e.target.value })}
              required
            >
              <option value="">请选择部门</option>
              {deptOptions.map((d) => (
                <option key={d.id} value={d.id}>
                  {deptOptionLabel(d)}
                </option>
              ))}
            </select>
          </Field>
          <Field label="岗位级别">
            <input
              type="number"
              className="input input-bordered input-sm"
              value={form.level}
              onChange={(e) => setForm({ ...form, level: Number(e.target.value) })}
            />
          </Field>
          <Field label="备注" className="sm:col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm w-full"
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
