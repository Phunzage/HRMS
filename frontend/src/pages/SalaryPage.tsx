import { useEffect, useState } from 'react'
import {
  fetchSalaries,
  createSalary,
  updateSalary,
  confirmSalary,
  paySalary,
  batchGenerate,
  type SalaryRecord,
  type SalaryPayload,
} from '../api/salary'
import { fetchAllEmployees, type Employee } from '../api/employee'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { SALARY_STATUS } from '../utils/status'
import { currentMonth, formatMoney } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'

const PAGE_SIZE = 10

// 工资单表单数据（金额用字符串便于输入，提交时转数字）
interface SalaryForm {
  employee_id: string
  month: string
  base_salary: string
  allowance: string
  bonus: string
  deduction: string
  remark: string
}

const emptyForm: SalaryForm = {
  employee_id: '',
  month: '',
  base_salary: '',
  allowance: '0',
  bonus: '0',
  deduction: '0',
  remark: '',
}

export default function SalaryPage() {
  const role = useAuthStore((s) => s.user?.role_code)
  const canManage = role === 'admin' || role === 'hr'

  const [month, setMonth] = useState(currentMonth())
  const [employeeId, setEmployeeId] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<SalaryRecord> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [employees, setEmployees] = useState<Employee[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<SalaryRecord | null>(null)
  const [form, setForm] = useState<SalaryForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  // 批量生成弹窗
  const [batchOpen, setBatchOpen] = useState(false)
  const [batchMonth, setBatchMonth] = useState(currentMonth())
  const [batchLoading, setBatchLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchSalaries({
      month,
      employee_id: employeeId ? Number(employeeId) : undefined,
      status: status || undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取工资单失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [month, employeeId, status, page, reloadKey])

  useEffect(() => {
    if (!canManage) return
    fetchAllEmployees()
      .then((res) => setEmployees(res.list))
      .catch(() => toast.error('加载员工列表失败'))
  }, [canManage])

  const reload = () => setReloadKey((k) => k + 1)

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, month })
    setModalOpen(true)
  }

  const openEdit = (record: SalaryRecord) => {
    setEditing(record)
    setForm({
      employee_id: String(record.employee_id),
      month: record.month,
      base_salary: String(record.base_salary),
      allowance: String(record.allowance),
      bonus: String(record.bonus),
      deduction: String(record.deduction),
      remark: record.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const payload: SalaryPayload = {
      employee_id: Number(form.employee_id),
      month: form.month,
      base_salary: Number(form.base_salary) || 0,
      allowance: Number(form.allowance) || 0,
      bonus: Number(form.bonus) || 0,
      deduction: Number(form.deduction) || 0,
      remark: form.remark,
    }
    setSaving(true)
    try {
      if (editing) {
        await updateSalary(editing.id, payload)
        toast.success('工资单已更新')
      } else {
        await createSalary(payload)
        toast.success('工资单创建成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleConfirm = (record: SalaryRecord) => {
    confirmSalary(record.id)
      .then(() => {
        toast.success('工资单已确认')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '确认失败'))
  }

  const handlePay = (record: SalaryRecord) => {
    if (!window.confirm(`确认发放 ${record.employee_name || ''} ${record.month} 工资？`)) return
    paySalary(record.id)
      .then(() => {
        toast.success('工资已发放')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '发放失败'))
  }

  const handleBatchGenerate = async () => {
    if (!batchMonth) {
      toast.error('请选择月份')
      return
    }
    setBatchLoading(true)
    try {
      const result = await batchGenerate(batchMonth)
      toast.success(`批量生成成功：新增 ${result.generated} 张，跳过 ${result.skipped} 张`)
      setBatchOpen(false)
      setMonth(batchMonth)
      setPage(1)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '批量生成失败')
    } finally {
      setBatchLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          <label className="form-control w-44">
            <div className="label">
              <span className="label-text">月份</span>
            </div>
            <input
              type="month"
              className="input input-bordered input-sm"
              value={month}
              onChange={(e) => {
                setMonth(e.target.value)
                setPage(1)
              }}
            />
          </label>
          {canManage && (
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
              <option value="confirmed">已确认</option>
              <option value="paid">已发放</option>
            </select>
          </label>
          {canManage && (
            <div className="ml-auto flex items-center gap-2">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setBatchOpen(true)}>
                批量生成
              </button>
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                ＋ 新增工资单
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
                    <th>月份</th>
                    <th>员工</th>
                    <th>部门</th>
                    <th className="text-right">基本工资</th>
                    <th className="text-right">津贴</th>
                    <th className="text-right">奖金</th>
                    <th className="text-right">扣款</th>
                    <th className="text-right">实发</th>
                    <th>状态</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((record) => (
                    <tr key={record.id}>
                      <td className="font-mono text-xs">{record.month}</td>
                      <td className="font-medium">{record.employee_name || `#${record.employee_id}`}</td>
                      <td>{record.department_name || '-'}</td>
                      <td className="text-right tabular-nums">{formatMoney(record.base_salary)}</td>
                      <td className="text-right tabular-nums">{formatMoney(record.allowance)}</td>
                      <td className="text-right tabular-nums">{formatMoney(record.bonus)}</td>
                      <td className="text-right tabular-nums text-error">{formatMoney(record.deduction)}</td>
                      <td className="text-right font-semibold tabular-nums">{formatMoney(record.net_salary)}</td>
                      <td>
                        <StatusBadge map={SALARY_STATUS} value={record.status} />
                      </td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            {record.status === 'draft' && (
                              <>
                                <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(record)}>
                                  编辑
                                </button>
                                <button type="button" className="btn btn-info btn-xs" onClick={() => handleConfirm(record)}>
                                  确认
                                </button>
                              </>
                            )}
                            {record.status === 'confirmed' && (
                              <button type="button" className="btn btn-success btn-xs" onClick={() => handlePay(record)}>
                                发放
                              </button>
                            )}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canManage ? 10 : 9} className="py-8 text-center text-base-content/50">
                        该月份暂无工资单
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

      {/* 批量生成 Modal */}
      <Modal
        open={batchOpen}
        title="批量生成工资单"
        onClose={() => setBatchOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setBatchOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={batchLoading} onClick={handleBatchGenerate}>
              {batchLoading && <span className="loading loading-spinner loading-sm" />}
              开始生成
            </button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          <Field label="工资月份" required>
            <input
              type="month"
              className="input input-bordered input-sm"
              value={batchMonth}
              onChange={(e) => setBatchMonth(e.target.value)}
              required
            />
          </Field>
          <p className="text-sm text-base-content/60">
            将为该月份尚无工资单的在职员工批量生成「草稿」工资单，已有工资单的员工自动跳过。
          </p>
        </div>
      </Modal>

      {/* 新增/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? `编辑工资单 - ${editing.employee_name ?? ''}` : '新增工资单'}
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
          <Field label="月份" required>
            <input
              type="month"
              className="input input-bordered input-sm"
              value={form.month}
              disabled={!!editing}
              onChange={(e) => setForm({ ...form, month: e.target.value })}
              required
            />
          </Field>
          <Field label="基本工资" required>
            <input
              type="number"
              step="0.01"
              min="0"
              className="input input-bordered input-sm"
              value={form.base_salary}
              onChange={(e) => setForm({ ...form, base_salary: e.target.value })}
              required
            />
          </Field>
          <Field label="津贴">
            <input
              type="number"
              step="0.01"
              min="0"
              className="input input-bordered input-sm"
              value={form.allowance}
              onChange={(e) => setForm({ ...form, allowance: e.target.value })}
            />
          </Field>
          <Field label="奖金">
            <input
              type="number"
              step="0.01"
              min="0"
              className="input input-bordered input-sm"
              value={form.bonus}
              onChange={(e) => setForm({ ...form, bonus: e.target.value })}
            />
          </Field>
          <Field label="扣款">
            <input
              type="number"
              step="0.01"
              min="0"
              className="input input-bordered input-sm"
              value={form.deduction}
              onChange={(e) => setForm({ ...form, deduction: e.target.value })}
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
