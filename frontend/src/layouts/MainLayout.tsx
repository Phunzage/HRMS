import { useEffect, useRef, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  Building2,
  FolderKanban,
  Clock,
  ClipboardList,
  Wallet,
  Megaphone,
  Target,
  Briefcase,
  UserCircle,
  UserCog,
  ShieldCheck,
  ScrollText,
  Inbox,
  Bell,
  Sun,
  Moon,
  type LucideIcon,
} from 'lucide-react'
import { useAuthStore } from '../store/auth'
import { useThemeStore } from '../store/theme'
import { useBadgeStore } from '../store/badges'
import {
  listNotifications,
  markRead,
  markAllRead,
  type NotificationItem,
} from '../api/notification'
import { fetchBadges, type Badges } from '../api/badge'
import { formatDateTime } from '../utils/format'
import ToastContainer from '../components/Toast'

// 侧边栏菜单项：roles 控制可见角色（admin 仅保留仪表盘与公告管理，业务菜单已移除）；badge 关联角标字段（>0 时右侧显示）
interface MenuItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
  roles: string[]
  badge?: keyof Badges
}

const menuItems: MenuItem[] = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard, end: true, roles: ['admin', 'hr'] },
  { to: '/employees', label: '员工管理', icon: Users, roles: ['hr', 'manager'] },
  { to: '/departments', label: '部门管理', icon: Building2, roles: ['hr', 'manager'] },
  { to: '/positions', label: '岗位管理', icon: FolderKanban, roles: ['hr'] },
  { to: '/attendance', label: '考勤管理', icon: Clock, roles: ['hr', 'manager', 'employee'] },
  { to: '/leaves', label: '请假审批', icon: ClipboardList, roles: ['hr', 'manager', 'employee'] },
  { to: '/approvals', label: '我的审批', icon: Inbox, roles: ['hr', 'manager'], badge: 'pending_approvals' },
  { to: '/salaries', label: '薪酬管理', icon: Wallet, roles: ['hr', 'employee'] },
  { to: '/announcements', label: '公告管理', icon: Megaphone, roles: ['admin', 'hr', 'manager', 'employee'], badge: 'unread_announcements' },
  { to: '/performance', label: '绩效管理', icon: Target, roles: ['hr', 'manager', 'employee'] },
  { to: '/recruitment', label: '招聘管理', icon: Briefcase, roles: ['hr'] },
  { to: '/profile', label: '个人中心', icon: UserCircle, roles: ['hr', 'manager', 'employee'] },
]

// 系统管理分组菜单：用户管理 / 角色管理 / 审计日志（仅 admin，hr 为业务管理员不涉系统配置）
const systemMenuItems: MenuItem[] = [
  { to: '/users', label: '用户管理', icon: UserCog, roles: ['admin'] },
  { to: '/roles', label: '角色管理', icon: ShieldCheck, roles: ['admin'] },
  { to: '/audit-logs', label: '审计日志', icon: ScrollText, roles: ['admin'] },
]

// 角色编码 → 中文名称
const roleNames: Record<string, string> = {
  admin: '管理员',
  hr: 'HR',
  manager: '经理',
  employee: '员工',
}

// 通知类型 → 中文徽章（铃铛下拉每条显示）
const NOTIF_TYPE_BADGE: Record<string, { label: string; badge: string }> = {
  leave_approval: { label: '请假审批', badge: 'badge-info' },
  leave_result: { label: '请假结果', badge: 'badge-success' },
  attendance_correction: { label: '补卡审批', badge: 'badge-warning' },
  announcement: { label: '公告', badge: 'badge-primary' },
  system: { label: '系统', badge: 'badge-ghost' },
}

export default function MainLayout() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const theme = useThemeStore((s) => s.theme)
  const toggleTheme = useThemeStore((s) => s.toggle)
  const navigate = useNavigate()

  // 导航栏角标（侧边栏菜单 + 顶栏铃铛共用，由 30s/聚焦轮询统一刷新）
  const pendingApprovals = useBadgeStore((s) => s.pending_approvals)
  const unreadAnnouncements = useBadgeStore((s) => s.unread_announcements)
  const unreadNotifications = useBadgeStore((s) => s.unread_notifications)
  const setBadges = useBadgeStore((s) => s.setBadges)
  const decrementNotification = useBadgeStore((s) => s.decrementNotification)

  // 站内通知：铃铛 + 下拉面板
  const [notifOpen, setNotifOpen] = useState(false)
  const [notifications, setNotifications] = useState<NotificationItem[]>([])
  const notifRef = useRef<HTMLDivElement>(null)

  // 角标轮询：每 30 秒 + 窗口聚焦时拉取 /badges，统一刷新侧边栏角标与铃铛未读数
  // （与旧 unread-count 轮询合并：unread_notifications 即铃铛未读数，避免重复请求）
  useEffect(() => {
    let cancelled = false
    const load = () => {
      fetchBadges()
        .then((res) => {
          if (!cancelled) setBadges(res)
        })
        .catch(() => undefined)
    }
    load()
    const timer = setInterval(load, 30000)
    window.addEventListener('focus', load)
    return () => {
      cancelled = true
      clearInterval(timer)
      window.removeEventListener('focus', load)
    }
  }, [setBadges])

  // 打开面板时加载最新 10 条通知
  useEffect(() => {
    if (!notifOpen) return
    listNotifications({ page: 1, page_size: 10 })
      .then((res) => setNotifications(res.list ?? []))
      .catch(() => undefined)
  }, [notifOpen])

  // 点击面板外部关闭
  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (notifRef.current && !notifRef.current.contains(e.target as Node)) {
        setNotifOpen(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  // 点击单条通知：标记已读并跳转（已读联动角标减一）
  const handleOpenNotification = (n: NotificationItem) => {
    if (!n.is_read) {
      markRead(n.id)
        .then(() => {
          setNotifications((list) => list.map((x) => (x.id === n.id ? { ...x, is_read: true } : x)))
          decrementNotification()
        })
        .catch(() => undefined)
    }
    setNotifOpen(false)
    if (n.link) navigate(n.link)
  }

  // 全部已读：本地列表置已读 + 角标清零
  const handleMarkAllRead = () => {
    markAllRead()
      .then(() => {
        setNotifications((list) => list.map((x) => ({ ...x, is_read: true })))
        setBadges({ unread_notifications: 0 })
      })
      .catch(() => undefined)
  }

  // 按角色过滤菜单
  const visibleMenu = menuItems.filter((item) => !user?.role_code || item.roles.includes(user.role_code))
  const visibleSystem = systemMenuItems.filter((item) => !user?.role_code || item.roles.includes(user.role_code))

  // 侧边栏角标值查找：menu item.badge（Badges 字段名）→ 当前 store 数值
  const badgeMap: Record<keyof Badges, number> = {
    pending_approvals: pendingApprovals,
    unread_announcements: unreadAnnouncements,
    unread_notifications: unreadNotifications,
  }

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex min-h-screen">
      {/* 侧边栏 */}
      <aside className="w-60 shrink-0 border-r border-base-300 bg-base-100">
        <div className="flex h-16 items-center gap-2 border-b border-base-300 px-4">
          <span className="text-xl font-bold">HRMS</span>
          <span className="badge badge-primary badge-sm">人力资源</span>
        </div>
        <ul className="menu w-full gap-1 p-2">
          {visibleMenu.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                end={item.end}
                className={({ isActive }) => (isActive ? 'active' : '')}
              >
                <item.icon className="h-5 w-5" />
                {item.label}
                {item.badge && badgeMap[item.badge] > 0 && (
                  <span className="badge badge-primary badge-sm ml-auto">{badgeMap[item.badge]}</span>
                )}
              </NavLink>
            </li>
          ))}
          {visibleSystem.length > 0 && (
            <>
              <li className="menu-title mt-2">系统管理</li>
              {visibleSystem.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) => (isActive ? 'active' : '')}
                  >
                    <item.icon className="h-5 w-5" />
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </>
          )}
        </ul>
      </aside>

      {/* 主区域 */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* 顶栏 */}
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-base-300 bg-base-100 px-6">
          <div className="text-lg font-medium">人力资源管理系统</div>
          <div className="flex items-center gap-3">
            {/* 通知铃铛 */}
            <div className="relative" ref={notifRef}>
              <button
                type="button"
                className="btn btn-ghost btn-circle relative"
                onClick={() => setNotifOpen((v) => !v)}
                title="通知"
              >
                <Bell className="h-5 w-5" />
                {unreadNotifications > 0 && (
                  <span className="absolute -right-0.5 -top-0.5 badge badge-error badge-xs badge-outline">
                    {unreadNotifications > 99 ? '99+' : unreadNotifications}
                  </span>
                )}
              </button>

              {notifOpen && (
                <div className="absolute right-0 z-30 mt-2 w-80 overflow-hidden rounded-box border border-base-300 bg-base-100 shadow-xl">
                  <div className="flex items-center justify-between border-b border-base-300 px-3 py-2">
                    <span className="text-sm font-medium">通知</span>
                    <button type="button" className="btn btn-ghost btn-xs" onClick={handleMarkAllRead}>
                      全部已读
                    </button>
                  </div>
                  <ul className="max-h-80 overflow-y-auto">
                    {notifications.map((n) => (
                      <li key={n.id}>
                        <button
                          type="button"
                          className={`flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left hover:bg-base-200 ${n.is_read ? '' : 'bg-primary/5'}`}
                          onClick={() => handleOpenNotification(n)}
                        >
                          <span className="flex w-full items-center gap-1.5 text-sm font-medium">
                            {!n.is_read && <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-error" />}
                            <span className={`badge badge-xs shrink-0 ${NOTIF_TYPE_BADGE[n.type]?.badge ?? 'badge-ghost'}`}>
                              {NOTIF_TYPE_BADGE[n.type]?.label ?? n.type}
                            </span>
                            <span className="truncate">{n.title}</span>
                          </span>
                          {n.content && <span className="line-clamp-2 w-full text-xs text-base-content/60">{n.content}</span>}
                          <span className="text-xs text-base-content/40">{formatDateTime(n.created_at)}</span>
                        </button>
                      </li>
                    ))}
                    {!notifications.length && (
                      <li className="px-3 py-8 text-center text-sm text-base-content/40">暂无通知</li>
                    )}
                  </ul>
                </div>
              )}
            </div>

            {/* 主题切换 */}
            <button
              type="button"
              className="btn btn-ghost btn-circle"
              onClick={toggleTheme}
              title={theme === 'dark' ? '切换浅色模式' : '切换暗色模式'}
            >
              {theme === 'dark' ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
            </button>

            <div className="text-right">
              <div className="text-sm font-medium">{user?.full_name || '未登录'}</div>
              <div className="text-xs text-base-content/60">
                {user ? roleNames[user.role_code] || user.role_code : ''}
              </div>
            </div>
            <button type="button" className="btn btn-ghost btn-sm" onClick={handleLogout}>
              退出登录
            </button>
          </div>
        </header>

        {/* 内容区 */}
        <main className="flex-1 overflow-y-auto bg-base-200 p-6">
          <Outlet />
        </main>
      </div>

      {/* 全局 toast */}
      <ToastContainer />
    </div>
  )
}
