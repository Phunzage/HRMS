import { useEffect, useState } from 'react'
import { getReport, type AttendanceReport as Report } from '../api/attendance'
import { currentMonth } from '../utils/format'

export default function AttendanceReportPage() {
  const [month, setMonth] = useState(currentMonth())
  const [data, setData] = useState<Report | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    getReport({ month })
      .then((res) => {
        if (!cancelled) setData(res)
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : '获取考勤月报失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [month])

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
              onChange={(e) => setMonth(e.target.value)}
            />
          </label>
          <div className="ml-auto text-sm text-base-content/60">
            {data?.month ? `${data.month} 考勤月报` : '考勤月报'}
          </div>
        </div>
      </div>

      {/* 汇总表 */}
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
                    <th className="text-right">出勤天数</th>
                    <th className="text-right">正常</th>
                    <th className="text-right">迟到</th>
                    <th className="text-right">早退</th>
                    <th className="text-right">旷工</th>
                    <th className="text-right">请假</th>
                  </tr>
                </thead>
                <tbody>
                  {(data?.list ?? []).map((row) => (
                    <tr key={row.employee_id}>
                      <td className="font-medium">{row.employee_name || `#${row.employee_id}`}</td>
                      <td>{row.department_name || '-'}</td>
                      <td className="text-right font-semibold tabular-nums">{row.work_days}</td>
                      <td className="text-right tabular-nums text-success">{row.normal_days}</td>
                      <td className="text-right tabular-nums text-warning">{row.late_count}</td>
                      <td className="text-right tabular-nums text-warning">{row.early_leave_count}</td>
                      <td className="text-right tabular-nums text-error">{row.absent_days}</td>
                      <td className="text-right tabular-nums">{row.leave_days}</td>
                    </tr>
                  ))}
                  {!data?.list.length && (
                    <tr>
                      <td colSpan={8} className="py-8 text-center text-base-content/50">
                        该月份暂无考勤汇总数据
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
