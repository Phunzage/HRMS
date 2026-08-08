import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Users,
  BadgeCheck,
  UserPlus,
  UserMinus,
  CalendarCheck,
  CalendarDays,
  Inbox,
  FileText,
  Briefcase,
  UserRound,
  Pin,
  ArrowRight,
  RefreshCw,
  type LucideIcon,
} from 'lucide-react'
import { fetchDashboardStats, type DashboardStats, type DeptCount } from '../api/dashboard'
import { fetchAnnouncements, type Announcement } from '../api/announcement'
import { formatDateTime } from '../utils/format'
import BarChart, { formatCompactAmount } from '../components/BarChart'

// 统计卡片配置
interface StatCard {
  label: string
  value: string
  icon: LucideIcon
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`
}

function StatCardView({ label, value, icon }: StatCard) {
  const Icon = icon
  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body flex-row items-center gap-4 p-5">
        <div className="text-3xl">
          <Icon className="h-6 w-6" />
        </div>
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
  const navigate = useNavigate()
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [announcements, setAnnouncements] = useState<Announcement[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  // 加载统计数据
  const loadStats = useCallback(async () => {
    const data = await fetchDashboardStats()
    setStats(data)
  }, [])

  // 首次挂载加载
  useEffect(() => {
    loadStats()
      .then(() => undefined)
      .catch(() => undefined)
      .finally(() => setLoading(false))
  }, [loadStats])

  // 窗口聚焦时自动刷新（其他页面/标签页操作后回到仪表盘能拿到最新数据）
  useEffect(() => {
    const onFocus = () => {
      setRefreshing(true)
      loadStats()
        .then(() => undefined)
        .catch(() => undefined)
        .finally(() => setRefreshing(false))
    }
    window.addEventListener('focus', onFocus)
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') onFocus()
    })
    return () => {
      window.removeEventListener('focus', onFocus)
      document.removeEventListener('visibilitychange', onFocus)
    }
  }, [loadStats])

  // 最新 3 条公告
  useEffect(() => {
    fetchAnnouncements({ page: 1, page_size: 3 })
      .then((res) => setAnnouncements(res.list ?? []))
      .catch(() => setAnnouncements([]))
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    )
  }

  if (!stats) {
    return null
  }

  const cards: StatCard[] = [
    { label: '员工总数', value: String(stats.total_employees), icon: Users },
    { label: '在职员工', value: String(stats.active_employees), icon: BadgeCheck },
    { label: '本月新增', value: String(stats.month_new_employees), icon: UserPlus },
    { label: '本月离职', value: String(stats.month_resigned), icon: UserMinus },
    { label: '本月出勤率', value: formatRate(stats.month_attendance_rate), icon: CalendarCheck },
    { label: '今日请假', value: String(stats.today_leave_count), icon: CalendarDays },
    { label: '待审批请假', value: String(stats.pending_leave_count), icon: Inbox },
    { label: '待确认工资单', value: String(stats.pending_salary_count), icon: FileText },
    { label: '在招职位', value: String(stats.open_job_count), icon: Briefcase },
    { label: '候选人', value: String(stats.candidate_count), icon: UserRound },
  ]

  return (
    <div className="flex flex-col gap-6">
      {/* 工具栏：标题 + 刷新 */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold">数据概览</h2>
        <button
          type="button"
          className="btn btn-ghost btn-sm gap-1"
          onClick={() => {
            setRefreshing(true)
            loadStats()
              .then(() => undefined)
              .catch(() => undefined)
              .finally(() => setRefreshing(false))
          }}
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

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
          <DeptDistribution data={stats.department_distribution ?? []} />
        </div>
      </div>

      {/* 近 6 个月工资总额与招聘趋势 */}
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h3 className="card-title text-base">近 6 个月工资总额</h3>
            <BarChart
              items={(stats.salary_trend ?? []).map((s) => ({ label: s.month, value: s.total_net }))}
              color="bg-primary"
              formatValue={formatCompactAmount}
            />
          </div>
        </div>
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h3 className="card-title text-base">近 6 个月招聘趋势</h3>
            <BarChart
              items={(stats.recruit_trend ?? []).map((r) => ({ label: r.month, value: r.count }))}
              color="bg-secondary"
            />
          </div>
        </div>
      </div>

      {/* 最新公告 */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-between">
            <h3 className="card-title text-base">最新公告</h3>
            <button
              type="button"
              className="btn btn-ghost btn-xs"
              onClick={() => navigate('/announcements')}
            >
              查看全部
              <ArrowRight className="h-3.5 w-3.5" />
            </button>
          </div>
          {announcements.length ? (
            <ul className="flex flex-col divide-y divide-base-200">
              {announcements.map((a) => (
                <li key={a.id}>
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 py-2 text-left hover:bg-base-200"
                    onClick={() => navigate('/announcements')}
                  >
                    {a.is_pinned && (
                      <span className="badge badge-error badge-sm gap-0.5">
                        <Pin className="h-3 w-3" />
                        置顶
                      </span>
                    )}
                    <span className="flex-1 truncate text-sm font-medium">{a.title}</span>
                    <span className="shrink-0 text-xs text-base-content/50">{formatDateTime(a.published_at)}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="py-4 text-sm text-base-content/60">暂无公告</p>
          )}
        </div>
      </div>
    </div>
  )
}
