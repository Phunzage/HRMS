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
  UserCircle,
} from 'lucide-react'
import { useAuthStore } from '../store/auth'
import ToastContainer from '../components/Toast'

// 侧边栏菜单项：roles 控制可见角色（admin 拥有全部）
const menuItems = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard, end: true, roles: ['admin', 'hr'] },
  { to: '/employees', label: '员工管理', icon: Users, roles: ['admin', 'hr', 'manager'] },
  { to: '/departments', label: '部门管理', icon: Building2, roles: ['admin', 'hr', 'manager'] },
  { to: '/positions', label: '岗位管理', icon: FolderKanban, roles: ['admin', 'hr'] },
  { to: '/attendance', label: '考勤管理', icon: Clock, roles: ['admin', 'hr', 'manager', 'employee'] },
  { to: '/leaves', label: '请假审批', icon: ClipboardList, roles: ['admin', 'hr', 'manager', 'employee'] },
  { to: '/salaries', label: '薪酬管理', icon: Wallet, roles: ['admin', 'hr', 'employee'] },
  { to: '/recruitment', label: '招聘管理', icon: Megaphone, roles: ['admin', 'hr'] },
  { to: '/profile', label: '个人中心', icon: UserCircle, roles: ['admin', 'hr', 'manager', 'employee'] },
]

// 角色编码 → 中文名称
const roleNames: Record<string, string> = {
  admin: '管理员',
  hr: 'HR',
  manager: '经理',
  employee: '员工',
}

export default function MainLayout() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  // 按角色过滤菜单
  const visibleMenu = menuItems.filter((item) => !user?.role_code || item.roles.includes(user.role_code))

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
              </NavLink>
            </li>
          ))}
        </ul>
      </aside>

      {/* 主区域 */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* 顶栏 */}
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-base-300 bg-base-100 px-6">
          <div className="text-lg font-medium">人力资源管理系统</div>
          <div className="flex items-center gap-4">
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
