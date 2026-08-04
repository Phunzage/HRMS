import { useEffect, useState } from 'react'
import { fetchDashboardStats, type DashboardStats, type DeptCount } from '../api/dashboard'

// 统计卡片配置
interface StatCard {
  label: string
  value: string
  icon: string
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`
}

function StatCardView({ label, value, icon }: StatCard) {
  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body flex-row items-center gap-4 p-5">
        <div className="text-3xl">{icon}</div>
        <div className="min-w-0">
          <div className="truncate text-sm text-base-content/60">{label}</div>
          <div className="text-2xl font-bold">{value}</div>
        </div>
      </div>
    </div>
  )
}

// 部门人数分布：纯 div 宽度百分比条状图
function DeptDistribution({ data }: { data: DeptCount[] }) {
  if (!data.length) {
    return <p className="py-4 text-sm text-base-content/60">暂无部门数据</p>
  }
  const max = Math.max(...data.map((d) => d.count), 1)
  return (
    <div className="flex flex-col gap-3">
      {data.map((d) => (
        <div key={d.department_name} className="flex items-center gap-3">
          <span className="w-24 shrink-0 truncate text-sm">{d.department_name}</span>
          <div className="h-4 flex-1 overflow-hidden rounded bg-base-200">
            <div
              className="h-full rounded bg-primary"
              style={{ width: `${(d.count / max) * 100}%` }}
            />
          </div>
          <span className="w-10 shrink-0 text-right text-sm tabular-nums">{d.count}</span>
        </div>
      ))}
    </div>
  )
}

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchDashboardStats()
      .then(setStats)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : '获取统计数据失败'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        <span>{error}</span>
      </div>
    )
  }

  if (!stats) {
    return null
  }

  const cards: StatCard[] = [
    { label: '员工总数', value: String(stats.total_employees), icon: '👥' },
    { label: '在职员工', value: String(stats.active_employees), icon: '✅' },
    { label: '本月新增', value: String(stats.month_new_employees), icon: '➕' },
    { label: '本月离职', value: String(stats.month_resigned), icon: '🚪' },
    { label: '本月出勤率', value: formatRate(stats.month_attendance_rate), icon: '🕐' },
    { label: '今日请假', value: String(stats.today_leave_count), icon: '📅' },
    { label: '待审批请假', value: String(stats.pending_leave_count), icon: '📋' },
    { label: '待确认工资单', value: String(stats.pending_salary_count), icon: '💰' },
    { label: '在招职位', value: String(stats.open_job_count), icon: '📌' },
    { label: '候选人', value: String(stats.candidate_count), icon: '👤' },
  ]

  return (
    <div className="flex flex-col gap-6">
      {/* 统计卡片 */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
        {cards.map((c) => (
          <StatCardView key={c.label} {...c} />
        ))}
      </div>

      {/* 部门人数分布 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title text-base">部门人数分布</h3>
          <DeptDistribution data={stats.department_distribution} />
        </div>
      </div>
    </div>
  )
}
