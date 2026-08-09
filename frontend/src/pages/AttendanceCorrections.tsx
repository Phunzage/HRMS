import { Fragment, useEffect, useMemo, useState, type ReactNode } from 'react'
import {
  submitCorrection,
  fetchCorrections,
  fetchMyCorrections,
  cancelCorrection,
  type CorrectionItem,
} from '../api/correction'
import {
  fetchMyApprovals,
  approveTask,
  rejectTask,
  type ApprovalTaskItem,
} from '../api/approval'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { CORRECTION_STATUS, CORRECTION_TYPE } from '../utils/status'
import { formatDate, formatDateTime } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'
import ProgressExpand from '../components/ProgressExpand'

const PAGE_SIZE = 10

// 提交补卡表单
interface CorrectionForm {
  work_date: string
  correction_type: string
  correction_time: string
  reason: string
}

const emptyForm: CorrectionForm = {
  work_date: '',
  correction_type: 'check_in',
  correction_time: '',
  reason: '',
}

export default function AttendanceCorrections() {
  const user = useAuthStore((s) => s.user)
  const role = user?.role_code
  // 审批/全部列表可见角色：admin/hr/manager 可审批（admin 可审 HR 池任务），员工仅看自己
  const canBrowse = role === 'admin' || role === 'hr' || role === 'manager'
  const canSubmit = !!user?.employee_id

  // 审批列表（admin/hr/manager）
  const [allData, setAllData] = useState<PageResult<CorrectionItem> | null>(null)
  const [allPage, setAllPage] = useState(1)
  const [allStatus, setAllStatus] = useState('')
  const [allLoading, setAllLoading] = useState(true)
  const [allError, setAllError] = useState('')

  // 我的补卡记录（employee）
  const [mine, setMine] = useState<PageResult<CorrectionItem> | null>(null)
  const [minePage, setMinePage] = useState(1)
  const [mineLoading, setMineLoading] = useState(true)
  const [mineError, setMineError] = useState('')

  // 我拥有的待审批任务（用于补卡列表仅对任务归属人显示审批按钮）
  const [myApprovals, setMyApprovals] = useState<ApprovalTaskItem[]>([])

  // 行内进度展开
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const [reloadKey, setReloadKey] = useState(0)

  // 提交弹窗
  const [submitOpen, setSubmitOpen] = useState(false)
  const [form, setForm] = useState<CorrectionForm>(emptyForm)
  const [submitting, setSubmitting] = useState(false)

  // 驳回弹窗（业务记录 + 对应审批任务 ID）
  const [rejectTarget, setRejectTarget] = useState<CorrectionItem | null>(null)
  const [rejectTaskId, setRejectTaskId] = useState<number | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [rejecting, setRejecting] = useState(false)

  // 补卡业务ID → 我的待审批任务ID 映射（仅任务归属人可操作）
  const myTaskByCorrId = useMemo(() => {
    const map: Record<number, number> = {}
    for (const a of myApprovals) {
      if (a.biz_type === 'attendance_correction') map[a.biz_id] = a.id
    }
    return map
  }, [myApprovals])

  // 审批/全部列表
  useEffect(() => {
    if (!canBrowse) return
    let cancelled = false
    setAllLoading(true)
    setAllError('')
    fetchCorrections({
      status: allStatus || undefined,
      page: allPage,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setAllData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setAllError(err instanceof Error ? err.message : '获取补卡申请失败')
      })
      .finally(() => {
        if (!cancelled) setAllLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [canBrowse, allStatus, allPage, reloadKey])

  // 我的补卡记录（employee）
  useEffect(() => {
    if (canBrowse) return
    let cancelled = false
    setMineLoading(true)
    setMineError('')
    fetchMyCorrections({ page: minePage, page_size: PAGE_SIZE })
      .then((res) => {
        if (!cancelled) setMine(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setMineError(err instanceof Error ? err.message : '获取补卡记录失败')
      })
      .finally(() => {
        if (!cancelled) setMineLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [canBrowse, minePage, reloadKey])

  // 我拥有的待审批任务（审批按钮归属校验，admin/hr 含 HR 池任务）
  useEffect(() => {
    if (!canBrowse) return
    let cancelled = false
    fetchMyApprovals({ status: 'pending' })
      .then((res) => {
        if (!cancelled) setMyApprovals(Array.isArray(res) ? res : [])
      })
      .catch(() => setMyApprovals([]))
    return () => {
      cancelled = true
    }
  }, [canBrowse, reloadKey])

  const reload = () => setReloadKey((k) => k + 1)

  const toggleProgress = (id: number) => setExpandedId((cur) => (cur === id ? null : id))

  const handleSubmit = async () => {
    if (!form.work_date || !form.correction_time) {
      toast.error('请填写补卡日期与时间')
      return
    }
    setSubmitting(true)
    try {
      await submitCorrection({
        work_date: form.work_date,
        correction_type: form.correction_type,
        correction_time: form.correction_time,
        reason: form.reason,
      })
      toast.success('已提交，等待部门经理审批')
      setSubmitOpen(false)
      setForm(emptyForm)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 审批通过（通用接口，返回实例状态判断是否还有下一步终审）
  const handleApprove = (taskId: number) => {
    approveTask(taskId)
      .then((inst) => {
        // 动态提示：下一步骤名称由后端返回（审批流程可配置），无下一步（终审通过）显示通用成功
        if (inst.next_step_name) {
          toast.success('已通过，等待' + inst.next_step_name)
        } else {
          toast.success('补卡审批已通过')
        }
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '操作失败'))
  }

  const handleReject = async () => {
    if (!rejectTarget || !rejectTaskId) return
    if (!rejectReason.trim()) {
      toast.error('请填写驳回原因')
      return
    }
    setRejecting(true)
    try {
      await rejectTask(rejectTaskId, rejectReason.trim())
      toast.success('已驳回补卡申请')
      setRejectTarget(null)
      setRejectTaskId(null)
      setRejectReason('')
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '操作失败')
    } finally {
      setRejecting(false)
    }
  }

  const handleCancel = (item: CorrectionItem) => {
    if (!window.confirm('确认取消该补卡申请？')) return
    cancelCorrection(item.id)
      .then(() => {
        toast.success('已取消补卡申请')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '取消失败'))
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          {canBrowse && (
            <label className="form-control w-32">
              <div className="label">
                <span className="label-text">状态</span>
              </div>
              <select
                className="select select-bordered select-sm"
                value={allStatus}
                onChange={(e) => {
                  setAllStatus(e.target.value)
                  setAllPage(1)
                }}
              >
                <option value="">全部状态</option>
                <option value="pending">待审批</option>
                <option value="approved">已通过</option>
                <option value="rejected">已驳回</option>
                <option value="cancelled">已取消</option>
              </select>
            </label>
          )}
          <div className="ml-auto flex items-center gap-2">
            {canSubmit && (
              <button type="button" className="btn btn-primary btn-sm" onClick={() => setSubmitOpen(true)}>
                ＋ 提交补卡申请
              </button>
            )}
          </div>
        </div>
      </div>

      {/* 列表 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4">
          {canBrowse ? (
            allError ? (
              <div className="alert alert-error">
                <span>{allError}</span>
              </div>
            ) : allLoading ? (
              <div className="flex items-center justify-center py-16">
                <span className="loading loading-spinner loading-lg text-primary" />
              </div>
            ) : (
              <>
                <CorrectionTable
                  list={allData?.list ?? []}
                  showApplicant
                  expandedId={expandedId}
                  renderExpanded={(item) => <ProgressExpand bizType="attendance_correction" bizId={item.id} />}
                  actions={(item) => {
                    const taskId = myTaskByCorrId[item.id]
                    return (
                      <div className="flex justify-end gap-1">
                        <button
                          type="button"
                          className="btn btn-ghost btn-xs"
                          onClick={() => toggleProgress(item.id)}
                        >
                          {expandedId === item.id ? '收起进度' : '进度'}
                        </button>
                        {item.status === 'pending' && taskId ? (
                          <>
                            <button
                              type="button"
                              className="btn btn-success btn-xs"
                              onClick={() => handleApprove(taskId)}
                            >
                              通过
                            </button>
                            <button
                              type="button"
                              className="btn btn-ghost btn-xs text-error"
                              onClick={() => {
                                setRejectTarget(item)
                                setRejectTaskId(taskId)
                                setRejectReason('')
                              }}
                            >
                              驳回
                            </button>
                          </>
                        ) : (
                          <span className="text-xs text-base-content/40">
                            {item.status === 'pending' ? '他人处理中' : '-'}
                          </span>
                        )}
                      </div>
                    )
                  }}
                />
                {allData && (
                  <div className="mt-2">
                    <Pagination page={allPage} pageSize={PAGE_SIZE} total={allData.total} onChange={setAllPage} />
                  </div>
                )}
              </>
            )
          ) : mineError ? (
            <div className="alert alert-error">
              <span>{mineError}</span>
            </div>
          ) : mineLoading ? (
            <div className="flex items-center justify-center py-16">
              <span className="loading loading-spinner loading-lg text-primary" />
            </div>
          ) : (
            <>
              <CorrectionTable
                list={mine?.list ?? []}
                showApplicant={false}
                expandedId={expandedId}
                renderExpanded={(item) => <ProgressExpand bizType="attendance_correction" bizId={item.id} />}
                actions={(item) => (
                  <div className="flex justify-end gap-1">
                    <button
                      type="button"
                      className="btn btn-ghost btn-xs"
                      onClick={() => toggleProgress(item.id)}
                    >
                      {expandedId === item.id ? '收起进度' : '进度'}
                    </button>
                    {item.status === 'pending' && (
                      <button type="button" className="btn btn-ghost btn-xs" onClick={() => handleCancel(item)}>
                        取消
                      </button>
                    )}
                  </div>
                )}
              />
              {mine && (
                <div className="mt-2">
                  <Pagination page={minePage} pageSize={PAGE_SIZE} total={mine.total} onChange={setMinePage} />
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* 提交补卡 Modal */}
      <Modal
        open={submitOpen}
        title="提交补卡申请"
        onClose={() => setSubmitOpen(false)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setSubmitOpen(false)}>
              取消
            </button>
            <button type="button" className="btn btn-primary" disabled={submitting} onClick={handleSubmit}>
              {submitting && <span className="loading loading-spinner loading-sm" />}
              提交
            </button>
          </>
        }
      >
        <div className="grid grid-cols-2 gap-3">
          <Field label="补卡日期" required>
            <input
              type="date"
              className="input input-bordered input-sm"
              value={form.work_date}
              max={new Date().toISOString().slice(0, 10)}
              onChange={(e) => setForm({ ...form, work_date: e.target.value })}
              required
            />
          </Field>
          <Field label="补卡类型" required>
            <select
              className="select select-bordered select-sm"
              value={form.correction_type}
              onChange={(e) => setForm({ ...form, correction_type: e.target.value })}
            >
              <option value="check_in">上班打卡</option>
              <option value="check_out">下班打卡</option>
            </select>
          </Field>
          <Field label="补卡时间" required>
            <input
              type="time"
              className="input input-bordered input-sm"
              value={form.correction_time}
              onChange={(e) => setForm({ ...form, correction_time: e.target.value })}
              required
            />
          </Field>
          <Field label="补卡原因" required className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={3}
              value={form.reason}
              onChange={(e) => setForm({ ...form, reason: e.target.value })}
              placeholder="请说明补卡原因"
              required
            />
          </Field>
        </div>
      </Modal>

      {/* 驳回 Modal */}
      <Modal
        open={!!rejectTarget}
        title="驳回补卡申请"
        onClose={() => {
          setRejectTarget(null)
          setRejectTaskId(null)
        }}
        footer={
          <>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                setRejectTarget(null)
                setRejectTaskId(null)
              }}
            >
              取消
            </button>
            <button type="button" className="btn btn-error" disabled={rejecting} onClick={handleReject}>
              {rejecting && <span className="loading loading-spinner loading-sm" />}
              确认驳回
            </button>
          </>
        }
      >
        <Field label="驳回原因" required>
          <textarea
            className="textarea textarea-bordered textarea-sm w-full"
            rows={3}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            placeholder="请输入驳回原因"
          />
        </Field>
      </Modal>
    </div>
  )
}

// 补卡申请通用表格（支持行内展开审批进度）
function CorrectionTable({
  list,
  showApplicant,
  actions,
  expandedId,
  renderExpanded,
}: {
  list: CorrectionItem[]
  showApplicant: boolean
  actions: (item: CorrectionItem) => ReactNode
  expandedId?: number | null
  renderExpanded?: (item: CorrectionItem) => ReactNode
}) {
  const rows = list ?? []
  return (
    <div className="overflow-x-auto">
      <table className="table table-sm">
        <thead>
          <tr>
            {showApplicant && <th>申请人</th>}
            {showApplicant && <th>部门</th>}
            <th>日期</th>
            <th>类型</th>
            <th>补卡时间</th>
            <th>原因</th>
            <th>状态</th>
            <th>提交时间</th>
            <th className="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((item) => (
            <Fragment key={item.id}>
              <tr>
                {showApplicant && <td className="font-medium">{item.employee_name || `#${item.employee_id}`}</td>}
                {showApplicant && <td>{item.department_name || '-'}</td>}
                <td>{formatDate(item.work_date)}</td>
                <td>{CORRECTION_TYPE[item.correction_type] ?? item.correction_type}</td>
                <td className="font-mono text-xs">{item.correction_time}</td>
                <td className="max-w-40 truncate text-base-content/60">{item.reason || '-'}</td>
                <td>
                  <StatusBadge map={CORRECTION_STATUS} value={item.status} />
                </td>
                <td className="text-base-content/60">{formatDateTime(item.created_at)}</td>
                <td className="text-right">{actions(item)}</td>
              </tr>
              {expandedId === item.id && renderExpanded && (
                <tr className="bg-base-200/50">
                  <td colSpan={showApplicant ? 9 : 7} className="px-6 py-3">
                    {renderExpanded(item)}
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
          {!rows.length && (
            <tr>
              <td colSpan={showApplicant ? 9 : 7} className="py-8 text-center text-base-content/50">
                暂无补卡申请
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
