import { Check, X, Circle } from 'lucide-react'
import type { ApprovalProgress, ApprovalStep } from '../api/approval'
import { formatDateTime } from '../utils/format'

// 步骤状态 → 图标 + 配色
function stepIcon(step: ApprovalStep) {
  switch (step.status) {
    case 'approved':
      return (
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-success text-white">
          <Check className="h-3.5 w-3.5" />
        </span>
      )
    case 'rejected':
      return (
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-error text-white">
          <X className="h-3.5 w-3.5" />
        </span>
      )
    default:
      return (
        <span className="flex h-6 w-6 items-center justify-center rounded-full border-2 border-base-300 text-base-content/40">
          <Circle className="h-3 w-3" />
        </span>
      )
  }
}

// 步骤状态 → 徽章
function stepBadge(step: ApprovalStep) {
  switch (step.status) {
    case 'approved':
      return <span className="badge badge-success badge-sm">已通过</span>
    case 'rejected':
      return <span className="badge badge-error badge-sm">已驳回</span>
    default:
      return <span className="badge badge-ghost badge-sm">待审批</span>
  }
}

// 审批进度时间线：步骤图标 + 步骤名 + 审批人 + 时间 + 意见
// 用于「我的申请/补卡记录」行内展开，与统一待办页审批 Modal 复用
export default function ApprovalTimeline({ progress }: { progress: ApprovalProgress }) {
  const tasks = progress.tasks ?? []
  if (!tasks.length) {
    return <p className="py-2 text-center text-sm text-base-content/40">暂无审批记录</p>
  }
  return (
    <ol className="space-y-0">
      {tasks.map((step, i) => {
        const isLast = i === tasks.length - 1
        return (
          <li key={step.id} className="relative flex gap-3 pb-5">
            {/* 连接线 */}
            {!isLast && <span className="absolute left-3 top-6 bottom-0 w-px bg-base-300" />}
            <div className="shrink-0">{stepIcon(step)}</div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium">
                  {step.step_name || `步骤 ${step.step_order}`}
                </span>
                {stepBadge(step)}
              </div>
              <div className="mt-0.5 text-xs text-base-content/60">
                {step.approver_name
                  ? `审批人：${step.approver_name}`
                  : step.status === 'pending'
                    ? '等待审批人处理'
                    : '审批人：-'}
                {step.acted_at && <span className="ml-2">{formatDateTime(step.acted_at)}</span>}
              </div>
              {step.comment && (
                <div className="mt-1 rounded-box bg-base-200 px-2 py-1 text-xs text-base-content/70">
                  意见：{step.comment}
                </div>
              )}
            </div>
          </li>
        )
      })}
    </ol>
  )
}
