import { useCallback, useEffect, useState, type ReactNode } from 'react'
import {
  fetchLeaves,
  fetchMyLeaves,
  fetchLeaveBalance,
  submitLeave,
  approveLeave,
  rejectLeave,
  cancelLeave,
  type LeaveRequest,
  type LeaveBalance,
  type LeaveMineResult,
} from '../api/leave'
import type { PageResult } from '../api/client'
import { useAuthStore } from '../store/auth'
import { toast } from '../store/toast'
import { LEAVE_STATUS, LEAVE_TYPE } from '../utils/status'
import { formatDate, formatDays } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import Pagination from '../components/Pagination'
import StatusBadge from '../components/StatusBadge'

const PAGE_SIZE = 10

// 提交申请表单
interface SubmitForm {
  leave_type: string
  start_date: string
  end_date: string
  reason: string
}

const emptySubmit: SubmitForm = {
  leave_type: 'annual',
  start_date: '',
  end_date: '',
  reason: '',
}

type TabKey = 'mine' | 'todo' | 'all'

export default function LeavePage() {
  const user = useAuthStore((s) => s.user)
  const role = user?.role_code
  const showTodo = role === 'admin' || role === 'hr' || role === 'manager'
  const showAll = role === 'admin' || role === 'hr'

  // admin/hr 无个人员工档案，默认落在「全部申请」
  const [activeTab, setActiveTab] = useState<TabKey>(showAll ? 'all' : 'mine')

  // 余额
  const [balance, setBalance] = useState<LeaveBalance[]>([])

  // 我的申请
  const [mine, setMine] = useState<LeaveMineResult | null>(null)
  const [minePage, setMinePage] = useState(1)

  // 待我审批
  const [todo, setTodo] = useState<LeaveRequest[]>([])
  const [todoLoading, setTodoLoading] = useState(false)

  // 全部申请
  const [allData, setAllData] = useState<PageResult<LeaveRequest> | null>(null)
  const [allPage, setAllPage] = useState(1)
  const [allStatus, setAllStatus] = useState('')

  const [listLoading, setListLoading] = useState(true)
  const [listError, setListError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  // 弹窗状态
  const [submitOpen, setSubmitOpen] = useState(false)
  const [submitForm, setSubmitForm] = useState<SubmitForm>(emptySubmit)
  const [submitting, setSubmitting] = useState(false)

  const [rejectTarget, setRejectTarget] = useState<LeaveRequest | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [rejecting, setRejecting] = useState(false)

  // 加载假期余额（admin/hr 未关联员工档案时静默留空）
  useEffect(() => {
    fetchLeaveBalance()
      .then(setBalance)
      .catch(() => setBalance([]))
  }, [])

  // 我的申请
  useEffect(() => {
    if (activeTab !== 'mine') return
    let cancelled = false
    setListLoading(true)
    setListError('')
    fetchMyLeaves({ page: minePage, page_size: PAGE_SIZE })
      .then((res) => {
        if (!cancelled) setMine(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setListError(err instanceof Error ? err.message : '获取申请列表失败')
      })
      .finally(() => {
        if (!cancelled) setListLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeTab, minePage, reloadKey])

  // 待我审批
  const loadTodo = useCallback(() => {
    setTodoLoading(true)
    fetchLeaves({ status: 'pending', page: 1, page_size: 100 })
      .then((res) => {
        let list = res.list
        // 经理只能审批指派给自己的申请
        if (role === 'manager' && user?.employee_id) {
          list = list.filter((l) => l.approver_id === user.employee_id)
        }
        setTodo(list)
      })
      .catch(() => toast.error('获取待审批列表失败'))
      .finally(() => setTodoLoading(false))
  }, [role, user?.employee_id])

  useEffect(() => {
    if (activeTab === 'todo') {
      loadTodo()
    }
  }, [activeTab, loadTodo, reloadKey])

  // 全部申请（admin/hr）
  useEffect(() => {
    if (activeTab !== 'all' || !showAll) return
    let cancelled = false
    setListLoading(true)
    setListError('')
    fetchLeaves({ status: allStatus || undefined, page: allPage, page_size: PAGE_SIZE })
      .then((res) => {
        if (!cancelled) setAllData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setListError(err instanceof Error ? err.message : '获取申请列表失败')
      })
      .finally(() => {
        if (!cancelled) setListLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [activeTab, showAll, allStatus, allPage, reloadKey])

  const reload = () => setReloadKey((k) => k + 1)

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      await submitLeave({
        leave_type: submitForm.leave_type,
        start_date: submitForm.start_date,
        end_date: submitForm.end_date,
        reason: submitForm.reason,
      })
      toast.success('请假申请已提交')
      setSubmitOpen(false)
      setSubmitForm(emptySubmit)
      // 刷新余额与列表
      fetchLeaveBalance()
        .then(setBalance)
        .catch(() => undefined)
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleApprove = (leave: LeaveRequest) => {
    approveLeave(leave.id)
      .then(() => {
        toast.success('审批已通过')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '审批失败'))
  }

  const handleReject = async () => {
    if (!rejectTarget) return
    setRejecting(true)
    try {
      await rejectLeave(rejectTarget.id, rejectReason)
      toast.success('申请已驳回')
      setRejectTarget(null)
      setRejectReason('')
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '驳回失败')
    } finally {
      setRejecting(false)
    }
  }

  const handleCancel = (leave: LeaveRequest) => {
    if (!window.confirm('确认取消该请假申请？')) return
    cancelLeave(leave.id)
      .then(() => {
        toast.success('申请已取消')
        reload()
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '取消失败'))
  }

  const switchTab = (tab: TabKey) => {
    setActiveTab(tab)
    setListError('')
  }

  // 余额徽章
  const balanceBadges = (
    <div className="flex flex-wrap gap-2">
      {balance.map((b) => (
        <div key={b.leave_type} className="badge badge-lg badge-outline gap-1">
          <span>{LEAVE_TYPE[b.leave_type] ?? b.leave_type}</span>
          <span className="text-success">余 {formatDays(b.remaining)}</span>
        </div>
      ))}
    </div>
  )

  return (
    <div className="flex flex-col gap-4">
      {/* 假期余额 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4">
          <h3 className="mb-2 text-sm font-medium text-base-content/60">我的假期余额</h3>
          {balance.length ? balanceBadges : <p className="text-sm text-base-content/40">暂无余额数据</p>}
        </div>
      </div>

      {/* Tab */}
      <div role="tablist" className="tabs tabs-boxed w-fit">
        <button
          type="button"
          role="tab"
          className={`tab ${activeTab === 'mine' ? 'tab-active' : ''}`}
          onClick={() => switchTab('mine')}
        >
          我的申请
        </button>
        {showTodo && (
          <button
            type="button"
            role="tab"
            className={`tab ${activeTab === 'todo' ? 'tab-active' : ''}`}
            onClick={() => switchTab('todo')}
          >
            待我审批
          </button>
        )}
        {showAll && (
          <button
            type="button"
            role="tab"
            className={`tab ${activeTab === 'all' ? 'tab-active' : ''}`}
            onClick={() => switchTab('all')}
          >
            全部申请
          </button>
        )}
      </div>

      {/* 工具栏（我的申请：提交按钮 / 全部申请：状态筛选） */}
      {activeTab === 'mine' && (
        <div className="flex justify-end">
          <button type="button" className="btn btn-primary btn-sm" onClick={() => setSubmitOpen(true)}>
            ＋ 提交申请
          </button>
        </div>
      )}
      {activeTab === 'all' && showAll && (
        <div className="flex justify-end">
          <select
            className="select select-bordered select-sm w-36"
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
        </div>
      )}

      {/* 列表 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4">
          {listError ? (
            <div className="alert alert-error">
              <span>{listError}</span>
            </div>
          ) : activeTab === 'todo' ? (
            todoLoading ? (
              <div className="flex items-center justify-center py-16">
                <span className="loading loading-spinner loading-lg text-primary" />
              </div>
            ) : (
              <LeaveTable
                list={todo}
                columns={['申请人', '部门', '类型', '开始', '结束', '天数', '事由', '状态']}
                actions={(l) => (
                  <div className="flex justify-end gap-1">
                    <button type="button" className="btn btn-success btn-xs" onClick={() => handleApprove(l)}>
                      通过
                    </button>
                    <button
                      type="button"
                      className="btn btn-ghost btn-xs text-error"
                      onClick={() => {
                        setRejectTarget(l)
                        setRejectReason('')
                      }}
                    >
                      驳回
                    </button>
                  </div>
                )}
              />
            )
          ) : listLoading ? (
            <div className="flex items-center justify-center py-16">
              <span className="loading loading-spinner loading-lg text-primary" />
            </div>
          ) : activeTab === 'mine' ? (
            <>
              <LeaveTable
                list={mine?.list ?? []}
                columns={['类型', '开始', '结束', '天数', '事由', '状态']}
                actions={(l) =>
                  l.status === 'pending' ? (
                    <div className="flex justify-end">
                      <button type="button" className="btn btn-ghost btn-xs" onClick={() => handleCancel(l)}>
                        取消
                      </button>
                    </div>
                  ) : null
                }
              />
              {mine && (
                <div className="mt-2">
                  <Pagination page={minePage} pageSize={PAGE_SIZE} total={mine.total} onChange={setMinePage} />
                </div>
              )}
            </>
          ) : (
            <>
              <LeaveTable
                list={allData?.list ?? []}
                columns={['申请人', '部门', '类型', '开始', '结束', '天数', '事由', '状态']}
                actions={() => null}
              />
              {allData && (
                <div className="mt-2">
                  <Pagination page={allPage} pageSize={PAGE_SIZE} total={allData.total} onChange={setAllPage} />
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* 提交申请 Modal */}
      <Modal
        open={submitOpen}
        title="提交请假申请"
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
          <Field label="请假类型" required>
            <select
              className="select select-bordered select-sm"
              value={submitForm.leave_type}
              onChange={(e) => setSubmitForm({ ...submitForm, leave_type: e.target.value })}
            >
              {Object.entries(LEAVE_TYPE).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </Field>
          <Field label="请假天数">
            <input
              type="text"
              className="input input-bordered input-sm bg-base-200"
              value={
                submitForm.start_date && submitForm.end_date
                  ? formatDays(
                      (new Date(submitForm.end_date).getTime() - new Date(submitForm.start_date).getTime()) /
                        86400000 +
                        1,
                    )
                  : '-'
              }
              disabled
            />
          </Field>
          <Field label="开始日期" required>
            <input
              type="date"
              className="input input-bordered input-sm"
              value={submitForm.start_date}
              onChange={(e) => setSubmitForm({ ...submitForm, start_date: e.target.value })}
              required
            />
          </Field>
          <Field label="结束日期" required>
            <input
              type="date"
              className="input input-bordered input-sm"
              value={submitForm.end_date}
              onChange={(e) => setSubmitForm({ ...submitForm, end_date: e.target.value })}
              required
            />
          </Field>
          <Field label="请假事由" required className="col-span-2">
            <textarea
              className="textarea textarea-bordered textarea-sm"
              rows={3}
              value={submitForm.reason}
              onChange={(e) => setSubmitForm({ ...submitForm, reason: e.target.value })}
              required
            />
          </Field>
        </div>
      </Modal>

      {/* 驳回 Modal */}
      <Modal
        open={!!rejectTarget}
        title="驳回申请"
        onClose={() => setRejectTarget(null)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setRejectTarget(null)}>
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

// 请假列表通用表格
interface LeaveTableProps {
  list: LeaveRequest[]
  columns: string[]
  actions: (leave: LeaveRequest) => ReactNode
}

function LeaveTable({ list, columns, actions }: LeaveTableProps) {
  const hasApplicant = columns[0] === '申请人'
  return (
    <div className="overflow-x-auto">
      <table className="table table-sm">
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c}>{c}</th>
            ))}
            <th className="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {list.map((l) => (
            <tr key={l.id}>
              {hasApplicant && <td className="font-medium">{l.employee_name || `#${l.employee_id}`}</td>}
              {hasApplicant && <td>{l.department_name || '-'}</td>}
              <td>{LEAVE_TYPE[l.leave_type] ?? l.leave_type}</td>
              <td>{formatDate(l.start_date)}</td>
              <td>{formatDate(l.end_date)}</td>
              <td>{formatDays(l.days)}</td>
              <td className="max-w-40 truncate text-base-content/60">{l.reason || '-'}</td>
              <td>
                <StatusBadge map={LEAVE_STATUS} value={l.status} />
              </td>
              <td className="text-right">{actions(l)}</td>
            </tr>
          ))}
          {!list.length && (
            <tr>
              <td colSpan={columns.length + 1} className="py-8 text-center text-base-content/50">
                暂无数据
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
