import { useEffect, useState } from 'react'
import { fetchApprovalProgress, type ApprovalProgress } from '../api/approval'
import ApprovalTimeline from './ApprovalTimeline'

// 行内懒加载的审批进度展开区：首次展开时按业务类型+业务ID拉取进度并渲染时间线
export default function ProgressExpand({ bizType, bizId }: { bizType: string; bizId: number }) {
  const [progress, setProgress] = useState<ApprovalProgress | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetchApprovalProgress(bizType, bizId)
      .then((res) => {
        if (!cancelled) setProgress(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '加载审批进度失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [bizType, bizId])

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-2 text-sm text-base-content/40">
        <span className="loading loading-spinner loading-sm" />
        加载审批进度…
      </div>
    )
  }
  if (error) {
    return <div className="py-2 text-sm text-error">{error}</div>
  }
  if (!progress) return null
  return <ApprovalTimeline progress={progress} />
}
