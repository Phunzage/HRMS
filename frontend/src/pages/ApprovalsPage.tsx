import { useCallback, useEffect, useState } from 'react'
import {
  Inbox,
  ClipboardList,
  CalendarClock,
  Loader2,
} from 'lucide-react'
import {
  fetchMyApprovals,
  fetchApprovalProgress,
  approveTask,
  rejectTask,
  type ApprovalTaskItem,
  type ApprovalProgress,
} from '../api/approval'
import { toast } from '../store/toast'
import { formatDateTime } from '../utils/format'
import Modal from '../components/Modal'
import Field from '../components/FormField'
import ApprovalTimeline from '../components/ApprovalTimeline'

// 业务类型 → 徽章（请假/补卡）
const BIZ_BADGE: Record<string, { label: string; badge: string }> = {
  leave: { label: '请假', badge: 'badge-info' },
  attendance_correction: { label: '补卡', badge: 'badge-warning' },
}

// 任务处理状态 → 徽章（我处理的）
const TASK_STATUS_BADGE: Record<string, { label: string; badge: string }> = {
  approved: { label: '已通过', badge: 'badge-success' },
  rejected: { label: '已驳回', badge: 'badge-error' },
}

type TabKey = 'todo' | 'done'

export default function ApprovalsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('todo')
  const [todo, setTodo] = useState<ApprovalTaskItem[]>([])
  const [done, setDone] = useState<ApprovalTaskItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  // 审批 Modal
  const [target, setTarget] = useState<ApprovalTaskItem | null>(null)
  const [progress, setProgress] = useState<ApprovalProgress | null>(null)
  const [progressLoading, setProgressLoading] = useState(false)
  const [comment, setComment] = useState('')
  const [acting, setActing] = useState(false)

  const load = useCallback(
    (tab: TabKey) => {
      setLoading(true)
      setError('')
      fetchMyApprovals({ status: tab === 'todo' ? 'pending' : 'done' })
        .then((res) => {
          const list = Array.isArray(res) ? res : []
          if (tab === 'todo') setTodo(list)
          else setDone(list)
        })
        .catch((err: unknown) => setError(err instanceof Error ? err.message : '获取审批任务失败'))
        .finally(() => setLoading(false))
    },
    [reloadKey],
  )

  useEffect(() => {
    load(activeTab)
  }, [activeTab, load])

  const reload = () => setReloadKey((k) => k + 1)

  // 打开审批 Modal：加载该业务的审批进度时间线
  const openModal = (item: ApprovalTaskItem) => {
    setTarget(item)
    setComment('')
    setProgress(null)
    setProgressLoading(true)
    fetchApprovalProgress(item.biz_type, item.biz_id)
      .then((res) => setProgress(res))
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : '加载审批进度失败'))
      .finally(() => setProgressLoading(false))
  }

  const closeModal = () => {
    setTarget(null)
    setProgress(null)
    setComment('')
  }

  const handleApprove = async () => {
    if (!target) return
    setActing(true)
    try {
      const inst = await approveTask(target.id, comment)
      if (inst.status === 'approved') {
        toast.success('审批已通过')
      } else {
        toast.success('已通过，等待 HR 终审')
      }
      closeModal()
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '审批失败')
    } finally {
      setActing(false)
    }
  }

  const handleReject = async () => {
    if (!target) return
    if (!comment.trim()) {
      toast.error('请填写驳回原因')
      return
    }
    setActing(true)
    try {
      await rejectTask(target.id, comment.trim())
      toast.success('已驳回申请')
      closeModal()
      reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '驳回失败')
    } finally {
      setActing(false)
    }
  }

  const list = activeTab === 'todo' ? todo : done
  const isTodoTab = activeTab === 'todo'

  return (
    <div className="flex flex-col gap-4">
      {/* 页面标题 + 提示 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Inbox className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold">我的审批</h2>
        </div>
        <span className="text-xs text-base-content/50">请假与补卡申请统一在此审批</span>
      </div>

      {/* Tab */}
      <div role="tablist" className="tabs tabs-boxed w-fit">
        <button
          type="button"
          role="tab"
          className={`tab ${activeTab === 'todo' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('todo')}
        >
          待我审批
          {todo.length > 0 && <span className="badge badge-primary badge-sm ml-1">{todo.length}</span>}
        </button>
        <button
          type="button"
          role="tab"
          className={`tab ${activeTab === 'done' ? 'tab-active' : ''}`}
          onClick={() => setActiveTab('done')}
        >
          我处理的
        </button>
      </div>

      {/* 列表 */}
      <div className="flex flex-col gap-3">
        {error && (
          <div className="alert alert-error">
            <span>{error}</span>
          </div>
        )}
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <span className="loading loading-spinner loading-lg text-primary" />
          </div>
        ) : !list.length ? (
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body flex flex-col items-center gap-2 py-12 text-base-content/40">
              <Inbox className="h-10 w-10" />
              <p className="text-sm">{isTodoTab ? '暂无待审批任务' : '暂无处理记录'}</p>
            </div>
          </div>
        ) : (
          list.map((item) => (
            <ApprovalCard key={item.id} item={item} isTodo={isTodoTab} onOpen={() => openModal(item)} />
          ))
        )}
      </div>

      {/* 审批 Modal：进度时间线 + 通过/驳回 + 驳回原因输入 */}
      <Modal
        open={!!target}
        title="审批详情"
        onClose={closeModal}
        footer={
          isTodoTab ? (
            <>
              <button
                type="button"
                className="btn btn-error"
                disabled={acting}
                onClick={handleReject}
              >
                {acting && <span className="loading loading-spinner loading-sm" />}
                驳回
              </button>
              <button
                type="button"
                className="btn btn-success"
                disabled={acting}
                onClick={handleApprove}
              >
                {acting && <span className="loading loading-spinner loading-sm" />}
                通过
              </button>
            </>
          ) : undefined
        }
      >
        {target && (
          <div className="flex flex-col gap-4">
            {/* 申请信息 */}
            <div className="rounded-box bg-base-200 p-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{target.applicant_name || '未知申请人'}</span>
                <span className={`badge badge-sm ${BIZ_BADGE[target.biz_type]?.badge ?? 'badge-ghost'}`}>
                  {BIZ_BADGE[target.biz_type]?.label ?? target.biz_type}
                </span>
                <span className="text-xs text-base-content/50">
                  {formatDateTime(target.created_at)} 提交
                </span>
              </div>
              <p className="mt-1 text-sm text-base-content/70">{target.summary}</p>
              {!isTodoTab && (
                <div className="mt-1">
                  <span className={`badge badge-sm ${TASK_STATUS_BADGE[target.status]?.badge ?? 'badge-ghost'}`}>
                    {TASK_STATUS_BADGE[target.status]?.label ?? target.status}
                  </span>
                </div>
              )}
            </div>

            {/* 进度时间线 */}
            <div>
              <h4 className="mb-2 text-sm font-medium text-base-content/60">审批进度</h4>
              {progressLoading ? (
                <div className="flex items-center gap-2 py-4 text-sm text-base-content/40">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  加载审批进度…
                </div>
              ) : progress ? (
                <ApprovalTimeline progress={progress} />
              ) : (
                <p className="py-2 text-sm text-base-content/40">暂无进度数据</p>
              )}
            </div>

            {/* 驳回原因输入（通过时也可作审批意见） */}
            {isTodoTab && (
              <Field label="审批意见（驳回必填）">
                <textarea
                  className="textarea textarea-bordered textarea-sm w-full"
                  rows={3}
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="通过可留空，驳回时必填原因"
                />
              </Field>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}

// 审批任务卡片：申请人/类型/摘要/提交时间 → 操作（待办=审批按钮，历史=状态徽章）
function ApprovalCard({
  item,
  isTodo,
  onOpen,
}: {
  item: ApprovalTaskItem
  isTodo: boolean
  onOpen: () => void
}) {
  const biz = BIZ_BADGE[item.biz_type]
  const status = TASK_STATUS_BADGE[item.status]
  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body flex flex-row flex-wrap items-center gap-3 p-4">
        {/* 类型图标 */}
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-box bg-base-200 text-base-content/50">
          {item.biz_type === 'leave' ? (
            <ClipboardList className="h-5 w-5" />
          ) : (
            <CalendarClock className="h-5 w-5" />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{item.applicant_name || '未知申请人'}</span>
            {biz && <span className={`badge badge-sm ${biz.badge}`}>{biz.label}</span>}
            {isTodo ? (
              <span className="badge badge-warning badge-sm">待审批</span>
            ) : (
              status && <span className={`badge badge-sm ${status.badge}`}>{status.label}</span>
            )}
          </div>
          <p className="mt-0.5 truncate text-sm text-base-content/70">{item.summary}</p>
          <p className="mt-0.5 text-xs text-base-content/40">{formatDateTime(item.created_at)} 提交</p>
        </div>
        <button type="button" className="btn btn-primary btn-sm" onClick={onOpen}>
          {isTodo ? '审批' : '进度'}
        </button>
      </div>
    </div>
  )
}
