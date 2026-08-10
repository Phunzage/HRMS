import { useEffect, useState } from 'react'
import {
  fetchAttendance,
  checkIn,
  checkOut,
  manualAttendance,
  updateAttendance,
  deleteAttendance,
  type Attendance,
} from '../api/attendance'
import { fetchAllEmployees, type Employee } from '../api/employee'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { usePerm } from '../hooks/usePerm'
import { toast } from '../store/toast'
import { ATTENDANCE_STATUS } from '../utils/status'
import { currentMonth, formatDate, formatDateTime, toBackendDateTime, toDatetimeLocal } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'
import AttendanceCorrections from './AttendanceCorrections'
import AttendanceReport from './AttendanceReport'

const PAGE_SIZE = 10

// 补录/编辑表单数据
interface AttForm {
  employee_id: string
  work_date: string
  check_in: string
  check_out: string
  status: string
  remark: string
}

const emptyForm: AttForm = {
  employee_id: '',
  work_date: '',
  check_in: '',
  check_out: '',
  status: 'normal',
  remark: '',
}

export default function AttendancePage() {
  const user = useAuthStore((s) => s.user)
  const role = user?.role_code
  const { has, hasAny } = usePerm()
  // 补录/编辑/删除需 attendance:write 权限点（种子仅 hr/admin，可在角色管理动态调整）
  const canManage = has('attendance:write')
  // 员工筛选下拉：admin/hr/manager 可浏览过滤；employee 仅本人（后端已强制过滤，前端隐藏下拉）
  const canFilter = role !== 'employee'
  // 打卡自助需关联员工档案
  const canCheck = !!user?.employee_id
  // 考勤月报：有 attendance:read 即可看（employee 后端已限本人）
  const canReport = hasAny(['attendance:read'])

  // Tab：考勤记录 / 补卡申请 / 考勤月报
  const [activeTab, setActiveTab] = useState<'records' | 'corrections' | 'report'>('records')

  // 查询条件
  const [month, setMonth] = useState(currentMonth())
  const [employeeId, setEmployeeId] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [reloadKey, setReloadKey] = useState(0)

  const [data, setData] = useState<PageResult<Attendance> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [employees, setEmployees] = useState<Employee[]>([])
  const [checking, setChecking] = useState<'in' | 'out' | null>(null)

  // Modal 状态
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Attendance | null>(null)
  const [form, setForm] = useState<AttForm>(emptyForm)
  const [saving, setSaving] = useState(false)

  // 加载考勤列表
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchAttendance({
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
        if (!cancelled) setError(err instanceof Error ? err.message : '获取考勤记录失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [month, employeeId, status, page, reloadKey])

  // 加载员工下拉（admin/hr）；非关键筛选下拉加载失败时静默降级为空数组
  useEffect(() => {
    if (!canFilter) return
    fetchAllEmployees()
      .then((res) => setEmployees(res.list ?? []))
      .catch(() => setEmployees([]))
  }, [canFilter])

  const reload = () => setReloadKey((k) => k + 1)

  const handleCheck = (type: 'in' | 'out') => {
    setChecking(type)
    const action = type === 'in' ? checkIn() : checkOut()
    action
      .then(() => {
        toast.success(type === 'in' ? '上班打卡成功' : '下班打卡成功')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '打卡失败'))
      .finally(() => setChecking(null))
  }

  const openCreate = () => {
    setEditing(null)
    setForm({ ...emptyForm, work_date: new Date().toISOString().slice(0, 10) })
    setModalOpen(true)
  }

  const openEdit = (att: Attendance) => {
    setEditing(att)
    setForm({
      employee_id: String(att.employee_id),
      work_date: att.work_date.slice(0, 10),
      check_in: toDatetimeLocal(att.check_in),
      check_out: toDatetimeLocal(att.check_out),
      status: att.status,
      remark: att.remark,
    })
    setModalOpen(true)
  }

  const handleSave = async () => {
    const payload = {
      employee_id: Number(form.employee_id),
      work_date: form.work_date,
      check_in: toBackendDateTime(form.check_in) || '',
      check_out: toBackendDateTime(form.check_out) || '',
      status: form.status,
      remark: form.remark,
    }
    setSaving(true)
    try {
      if (editing) {
        await updateAttendance(editing.id, payload)
        toast.success('考勤记录已更新')
      } else {
        await manualAttendance(payload)
        toast.success('考勤补录成功')
      }
      setModalOpen(false)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = (att: Attendance) => {
    if (!window.confirm(`确认删除该考勤记录？`)) return
    deleteAttendance(att.id)
      .then(() => {
        toast.success('考勤记录已删除')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '删除失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tab 栏（窄屏横向滚动） */}
      <div className="overflow-x-auto">
        <div role="tablist" className="tabs tabs-boxed w-max">
          <button
            type="button"
            role="tab"
            className={`tab ${activeTab === 'records' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('records')}
          >
            考勤记录
          </button>
          <button
            type="button"
            role="tab"
            className={`tab ${activeTab === 'corrections' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('corrections')}
          >
            补卡申请
          </button>
          {canReport && (
            <button
              type="button"
              role="tab"
              className={`tab ${activeTab === 'report' ? 'tab-active' : ''}`}
              onClick={() => setActiveTab('report')}
            >
              考勤月报
            </button>
          )}
        </div>
      </div>

      {activeTab === 'corrections' && <AttendanceCorrections />}
      {activeTab === 'report' && canReport && <AttendanceReport />}

      {activeTab === 'records' && (
      <>
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
          {canFilter && (
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
              <option value="normal">正常</option>
              <option value="late">迟到</option>
              <option value="early_leave">早退</option>
              <option value="absent">缺勤</option>
            </select>
          </label>

          <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
            {canCheck && (
              <>
                <button
                  type="button"
                  className="btn btn-success btn-sm"
                  disabled={checking !== null}
                  onClick={() => handleCheck('in')}
                >
                  {checking === 'in' && <span className="loading loading-spinner loading-sm" />}
                  打卡上班
                </button>
                <button
                  type="button"
                  className="btn btn-warning btn-sm"
                  disabled={checking !== null}
                  onClick={() => handleCheck('out')}
                >
                  {checking === 'out' && <span className="loading loading-spinner loading-sm" />}
                  打卡下班
                </button>
              </>
            )}
            {canManage && (
              <button type="button" className="btn btn-primary btn-sm" onClick={openCreate}>
                ＋ 补录考勤
              </button>
            )}
          </div>
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
                    <th>日期</th>
                    <th>员工</th>
                    <th>部门</th>
                    <th>上班时间</th>
                    <th>下班时间</th>
                    <th>状态</th>
                    {canManage && <th className="text-right">操作</th>}
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((att) => (
                    <tr key={att.id}>
                      <td>{formatDate(att.work_date)}</td>
                      <td className="font-medium">{att.employee_name || `#${att.employee_id}`}</td>
                      <td>{att.department_name || '-'}</td>
                      <td className="font-mono text-xs">{formatDateTime(att.check_in)}</td>
                      <td className="font-mono text-xs">{formatDateTime(att.check_out)}</td>
                      <td>
                        <StatusBadge map={ATTENDANCE_STATUS} value={att.status} />
                      </td>
                      {canManage && (
                        <td className="text-right">
                          <div className="flex justify-end gap-1">
                            <button type="button" className="btn btn-ghost btn-xs" onClick={() => openEdit(att)}>
                              编辑
                            </button>
                            <button type="button" className="btn btn-ghost btn-xs text-error" onClick={() => handleDelete(att)}>
                              删除
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={canManage ? 7 : 6} className="py-8 text-center text-base-content/50">
                        该月份暂无考勤记录
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

      {/* 补录/编辑 Modal */}
      <Modal
        open={modalOpen}
        title={editing ? '编辑考勤记录' : '补录考勤'}
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
          <Field label="员工" required>
            <select
              className="select select-bordered select-sm"
              value={form.employee_id}
              onChange={(e) => setForm({ ...form, employee_id: e.target.value })}
              required
            >
              <option value="">请选择员工</option>
              {employees.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="日期" required>
            <input
              type="date"
              className="input input-bordered input-sm"
              value={form.work_date}
              onChange={(e) => setForm({ ...form, work_date: e.target.value })}
              required
            />
          </Field>
          <Field label="上班时间">
            <input
              type="datetime-local"
              className="input input-bordered input-sm"
              value={form.check_in}
              onChange={(e) => setForm({ ...form, check_in: e.target.value })}
            />
          </Field>
          <Field label="下班时间">
            <input
              type="datetime-local"
              className="input input-bordered input-sm"
              value={form.check_out}
              onChange={(e) => setForm({ ...form, check_out: e.target.value })}
            />
          </Field>
          <Field label="状态">
            <select
              className="select select-bordered select-sm"
              value={form.status}
              onChange={(e) => setForm({ ...form, status: e.target.value })}
            >
              <option value="normal">正常</option>
              <option value="late">迟到</option>
              <option value="early_leave">早退</option>
              <option value="absent">缺勤</option>
            </select>
          </Field>
          <Field label="备注" className="sm:col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={2}
              value={form.remark}
              onChange={(e) => setForm({ ...form, remark: e.target.value })}
            />
          </Field>
        </div>
      </Modal>
      </>
      )}
    </div>
  )
}
