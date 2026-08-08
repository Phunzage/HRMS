import { useEffect, useState } from 'react'
import {
  fetchPerformance,
  createPerformance,
  updatePerformance,
  submitPerformance,
  approvePerformance,
  deletePerformance,
  type PerformanceItem,
} from '../api/performance'
import { fetchAllEmployees, type Employee } from '../api/employee'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { PERFORMANCE_STATUS, PERFORMANCE_RATING } from '../utils/status'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'

const PAGE_SIZE = 10

// 当前考核周期：上半年 → YYYY-H1，下半年 → YYYY-H2
function currentPeriod(): string {
  const now = new Date()
  return `${now.getFullYear()}-${now.getMonth() + 1 <= 6 ? 'H1' : 'H2'}`
}

// 周期下拉选项（去年 ~ 今年）
function periodOptions(): string[] {
  const y = new Date().getFullYear()
  const arr: string[] = []
  for (let yy = y - 1; yy <= y; yy++) {
    arr.push(`${yy}-H1`, `${yy}-H2`)
  }
  return arr
}

// 绩效表单
interface PerfForm {
  employee_id: string
  period: string
  score: string
  comments: string
}

const emptyForm: PerfForm = {
  employee_id: '',
  period: currentPeriod(),
  score: '',
  comments: '',
}

export default function PerformancePage() {
  const role = useAuthStore((s) => s.user?.role_code)
  // 操作权限：employee 只读；manager 可建/改/提交/删（不可批准）
  const canOperate = role === 'admin' || role === 'hr' || role === 'manager'
  const canApprove = role === 'admin' || role === 'hr'
  const canFilterEmployee = role === 'admin' || role === 'hr'

  const [period, setPeriod] = useState('')
  const [status, setStatus] = useState('')
  const [employeeId, setEmployeeId] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<PerformanceItem> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [employees, setEmployees] = useState<Employee[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PerformanceItem | null>(null)
  const [form, setForm] = useState<PerfForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchPerformance({
      period: period || undefined,
      status: status || undefined,
      employee_id: employeeId ? Number(employeeId) : undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取绩效列表失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [period, status, employeeId, page, reloadKey])

  useEffect(() => {
    if (!canOperate) return
    // 非关键筛选下拉：员工列表加载失败时静默降级为空数组
    fetchAllEmployees()
      .then((res) => setEmployees(res.list))
      .catch(() => setEmployees([]))
  }, [canOperate])

  const reload = () => setReloadKey((k) => k + 1)

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, period: currentPeriod() })
    setModalOpen(true)
  }

  const openEdit = (item: PerformanceItem) => {
    setEditing(item)
    setForm({
      employee_id: String(item.employee_id),
      period: item.period,
      score: String(item.score),
      comments: item.comments,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const score = Number(form.score)
    if (Number.isNaN(score) || score < 0 || score > 100) {
      toast.error('考核分数应在 0-100 之间')
      return
    }
    if (!form.employee_id) {
      toast.error('请选择员工')
      return
    }
    setSaving(true)
    try {
      const payload = { score, comments: form.comments }
      if (editing) {
        await updatePerformance(editing.id, payload)
        toast.success('绩效已更新')
      } else {
        await createPerformance({
          employee_id: Number(form.employee_id),
          period: form.period,
          score,
          comments: form.comments,
        })
        toast.success('绩效创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleSubmit = (item: PerformanceItem) => {
    if (!window.confirm(`确认提交 ${item.employee_name || ''} ${item.period} 绩效？`)) return
    submitPerformance(item.id)
      .then(() => {
        toast.success('绩效已提交')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '提交失败'))
  }

  const handleApprove = (item: PerformanceItem) => {
    if (!window.confirm(`确认批准 ${item.employee_name || ''} ${item.period} 绩效？`)) return
    approvePerformance(item.id)
      .then(() => {
        toast.success('绩效已批准')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '批准失败'))
  }

  const handleDelete = (item: PerformanceItem) => {
    if (!window.confirm(`确认删除 ${item.employee_name || ''} ${item.period} 绩效？`)) return
    deletePerformance(item.id)
      .then(() => {
        toast.success('绩效已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          <label className="form-control w-32">
            <div className="label">
              <span className="label-text">考核周期</span>
            </div>
            <select
              className="select select-bordered select-sm"
              value={period}
              onChange={(e) => {
                setPeriod(e.target.value)
                setPage(1)
              }}
            >
              <option value="">全部周期</option>
              {periodOptions().map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </label>
          <label className="form-control w-32">
            <div className="label">
              <span className="label-text">状态</span>
            </div>
            <select
              className="select select-bordered select-sm"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value)
                setPage(1)
              }}
            >
              <option value="">全部状态</option>
              <option value="draft">草稿</option>
              <option value="submitted">待审批</option>
              <option value="approved">已批准</option>
            </select>
          </label>
          {canFilterEmployee && (
            <label className="form-control w-44">
              <div className="label">
                <span className="label-text">员工</span>
              </div>
              <select
                className="select select-bordered select-sm"
                value={employeeId}
                onChange={(e) => {
                  setEmployeeId(e.target.value)
                  setPage(1)
                }}
              >
                <option value="">全部员工</option>
                {employees.map((e) => (
                  <option key={e.id} value={e.id}>
                    {e.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          {canOperate && (
            <div className="ml-auto">
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                ＋ 新增绩效
              </button>
            </div>
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
                    <th>员工</th>
                    <th>部门</th>
                    <th>周期</th>
                    <th className="text-right">分数</th>
                    <th>评级</th>
                    <th>状态</th>
                    <th>评语</th>
                    {canOperate && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((item) => (
                    <tr key={item.id}>
                      <td className="font-medium">
                        {item.employee_name || `#${item.employee_id}`}
                        {item.emp_no && <span className="ml-1 font-mono text-xs text-base-content/40">{item.emp_no}</span>}
                      </td>
                      <td>{item.department_name || '-'}</td>
                      <td className="font-mono text-xs">{item.period}</td>
                      <td className="text-right font-semibold tabular-nums">{item.score}</td>
                      <td>
                        <StatusBadge map={PERFORMANCE_RATING} value={item.rating} />
                      </td>
                      <td>
                        <StatusBadge map={PERFORMANCE_STATUS} value={item.status} />
                      </td>
                      <td className="max-w-48 truncate text-base-content/60">{item.comments || '-'}</td>
                      {canOperate && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            {item.status === 'draft' && (
                              <>
                                <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(item)}>
                                  编辑
                                </button>
                                <button type="button" className="btn btn-info btn-xs" onClick={() => handleSubmit(item)}>
                                  提交
                                </button>
                                <button
                                  type="button"
                                  className="btn btn-ghost btn-xs text-error"
                                  onClick={() => handleDelete(item)}
                                >
                                  删除
                                </button>
                              </>
                            )}
                            {item.status === 'submitted' && canApprove && (
                              <button type="button" className="btn btn-success btn-xs" onClick={() => handleApprove(item)}>
                                批准
                              </button>
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canOperate ? 8 : 7} className="py-8 text-center text-base-content/50">
                        暂无绩效记录
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
        title={editing ? `编辑绩效 - ${editing.employee_name ?? ''}` : '新增绩效'}
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
          <Field label="员工" required>
            <select
              className="select select-bordered select-sm"
              value={form.employee_id}
              disabled={!!editing}
              onChange={(e) => setForm({ ...form, employee_id: e.target.value })}
              required
            >
              <option value="">请选择员工</option>
              {employees.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}（{e.emp_no}）
                </option>
              ))}
            </select>
          </Field>
          <Field label="考核周期" required>
            <select
              className="select select-bordered select-sm"
              value={form.period}
              disabled={!!editing}
              onChange={(e) => setForm({ ...form, period: e.target.value })}
              required
            >
              {periodOptions().map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </Field>
          <Field label="考核分数" required>
            <input
              type="number"
              step="0.5"
              min="0"
              max="100"
              className="input input-bordered input-sm"
              value={form.score}
              onChange={(e) => setForm({ ...form, score: e.target.value })}
              placeholder="0-100"
              required
            />
          </Field>
          <Field label="评语" className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={3}
              value={form.comments}
              onChange={(e) => setForm({ ...form, comments: e.target.value })}
            />
          </Field>
        </div>
      </Modal>
    </div>
  )
}
