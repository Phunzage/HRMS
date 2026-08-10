import { useEffect, useState } from 'react'
import { listAuditLogs, type AuditLog } from '../api/audit'
import type { PageResult } from '../api/client'
import { formatDateTime } from '../utils/format'
import Pagination from '../components/Pagination'

const PAGE_SIZE = 15

// 模块编码 → 中文名称（表格与筛选下拉共用）
const MODULE_LABELS: Record<string, string> = {
  auth: '认证',
  employee: '员工',
  department: '部门',
  position: '岗位',
  attendance: '考勤',
  leave: '请假',
  salary: '薪酬',
  recruitment: '招聘',
  user: '用户',
  role: '角色',
  audit: '审计',
  dashboard: '仪表盘',
}

// 模块下拉选项（固定顺序）
const MODULE_OPTIONS = [
  'auth',
  'employee',
  'department',
  'position',
  'attendance',
  'leave',
  'salary',
  'recruitment',
  'user',
  'role',
  'audit',
]

// 动作编码 → 中文名称
const ACTION_LABELS: Record<string, string> = {
  login: '登录',
  create: '创建',
  update: '修改',
  delete: '删除',
  reset_password: '重置密码',
  confirm: '确认',
  pay: '发放',
  approve: '通过',
  reject: '驳回',
  cancel: '取消',
}

// 动作下拉选项
const ACTION_OPTIONS = ['login', 'create', 'update', 'delete', 'reset_password', 'confirm', 'pay', 'approve', 'reject', 'cancel']

// 动作 → badge 颜色
const ACTION_BADGE: Record<string, string> = {
  login: 'badge-info',
  create: 'badge-success',
  update: 'badge-warning',
  delete: 'badge-error',
  reset_password: 'badge-warning',
  confirm: 'badge-info',
  pay: 'badge-success',
  approve: 'badge-success',
  reject: 'badge-error',
  cancel: 'badge-ghost',
}

// 动作徽章
function ActionBadge({ action }: { action: string }) {
  return (
    <span className={`badge badge-sm ${ACTION_BADGE[action] ?? 'badge-ghost'}`}>
      {ACTION_LABELS[action] ?? action}
    </span>
  )
}

export default function AuditLogsPage() {
  // 已生效的筛选条件（查询按钮提交）
  const [filters, setFilters] = useState({ module: '', action: '', username: '', start_date: '', end_date: '' })
  // 表单草稿
  const [draft, setDraft] = useState({ module: '', action: '', username: '', start_date: '', end_date: '' })
  const [page, setPage] = useState(1)

  const [data, setData] = useState<PageResult<AuditLog> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    listAuditLogs({
      module: filters.module || undefined,
      action: filters.action || undefined,
      username: filters.username || undefined,
      start_date: filters.start_date || undefined,
      end_date: filters.end_date || undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取审计日志失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [filters, page])

  // 查询：应用草稿条件并回到第一页
  const handleQuery = () => {
    setFilters({ ...draft })
    setPage(1)
  }

  // 清空条件
  const handleReset = () => {
    setDraft({ module: '', action: '', username: '', start_date: '', end_date: '' })
    setFilters({ module: '', action: '', username: '', start_date: '', end_date: '' })
    setPage(1)
  }

  return (
    <div className="flex flex-col gap-4">
      {/* 工具栏 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body flex flex-row flex-wrap items-end gap-3 p-4">
          <select
            className="select select-bordered select-sm w-32"
            value={draft.module}
            onChange={(e) => setDraft({ ...draft, module: e.target.value })}
          >
            <option value="">全部模块</option>
            {MODULE_OPTIONS.map((m) => (
              <option key={m} value={m}>
                {MODULE_LABELS[m]}
              </option>
            ))}
          </select>
          <select
            className="select select-bordered select-sm w-32"
            value={draft.action}
            onChange={(e) => setDraft({ ...draft, action: e.target.value })}
          >
            <option value="">全部动作</option>
            {ACTION_OPTIONS.map((a) => (
              <option key={a} value={a}>
                {ACTION_LABELS[a]}
              </option>
            ))}
          </select>
          <input
            type="text"
            className="input input-bordered input-sm w-full sm:w-40"
            placeholder="操作者用户名"
            value={draft.username}
            onChange={(e) => setDraft({ ...draft, username: e.target.value })}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleQuery()
            }}
          />
          <label className="form-control flex-1 basis-40 sm:basis-auto">
            <div className="label">
              <span className="label-text">开始日期</span>
            </div>
            <input
              type="date"
              className="input input-bordered input-sm w-full"
              value={draft.start_date}
              onChange={(e) => setDraft({ ...draft, start_date: e.target.value })}
            />
          </label>
          <label className="form-control flex-1 basis-40 sm:basis-auto">
            <div className="label">
              <span className="label-text">结束日期</span>
            </div>
            <input
              type="date"
              className="input input-bordered input-sm w-full"
              value={draft.end_date}
              onChange={(e) => setDraft({ ...draft, end_date: e.target.value })}
            />
          </label>
          <div className="ml-auto flex gap-2">
            <button type="button" className="btn btn-primary btn-sm" onClick={handleQuery}>
              查询
            </button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={handleReset}>
              清空
            </button>
          </div>
        </div>
      </div>

      {/* 审计日志表格（只读） */}
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
                    <th>时间</th>
                    <th>操作者</th>
                    <th>模块</th>
                    <th>动作</th>
                    <th>资源</th>
                    <th>详情</th>
                    <th>IP</th>
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((log) => (
                    <tr key={log.id}>
                      <td className="whitespace-nowrap text-base-content/70">{formatDateTime(log.created_at)}</td>
                      <td className="font-medium">{log.username || '-'}</td>
                      <td>{MODULE_LABELS[log.module] ?? log.module}</td>
                      <td>
                        <ActionBadge action={log.action} />
                      </td>
                      <td className="font-mono text-xs">{log.resource_id || '-'}</td>
                      <td className="max-w-80 truncate text-base-content/80">{log.detail || '-'}</td>
                      <td className="font-mono text-xs text-base-content/70">{log.ip || '-'}</td>
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={7} className="py-8 text-center text-base-content/50">
                        暂无审计日志
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
    </div>
  )
}
