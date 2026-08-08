import { Routes, Route, Navigate } from 'react-router-dom'
import RequireAuth, { GuestOnly } from './router/guard'
import MainLayout from './layouts/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import EmployeeList from './pages/EmployeeList'
import DepartmentTree from './pages/DepartmentTree'
import PositionPage from './pages/PositionPage'
import AttendancePage from './pages/AttendancePage'
import LeavePage from './pages/LeavePage'
import SalaryPage from './pages/SalaryPage'
import AnnouncementsPage from './pages/Announcements'
import PerformancePage from './pages/PerformancePage'
import RecruitmentPage from './pages/RecruitmentPage'
import ProfilePage from './pages/ProfilePage'
import UsersPage from './pages/Users'
import RolesPage from './pages/Roles'
import AuditLogsPage from './pages/AuditLogs'
import { useAuthStore } from './store/auth'

// 首页：admin/hr 看统计仪表盘，其他角色跳转到考勤（其具备权限的页面）
function Home() {
  const role = useAuthStore((s) => s.user?.role_code)
  if (role === 'admin' || role === 'hr') {
    return <Dashboard />
  }
  return <Navigate to="/attendance" replace />
}

function App() {
  return (
    <Routes>
      {/* 登录页（已登录则重定向首页） */}
      <Route
        path="/login"
        element={
          <GuestOnly>
            <Login />
          </GuestOnly>
        }
      />

      {/* 需登录的主布局 */}
      <Route
        element={
          <RequireAuth>
            <MainLayout />
          </RequireAuth>
        }
      >
        <Route path="/" element={<Home />} />
        <Route path="/employees" element={<EmployeeList />} />
        <Route path="/departments" element={<DepartmentTree />} />
        <Route path="/positions" element={<PositionPage />} />
        <Route path="/attendance" element={<AttendancePage />} />
        <Route path="/leaves" element={<LeavePage />} />
        <Route path="/salaries" element={<SalaryPage />} />
        <Route path="/announcements" element={<AnnouncementsPage />} />
        <Route path="/performance" element={<PerformancePage />} />
        <Route path="/recruitment" element={<RecruitmentPage />} />
        <Route path="/users" element={<UsersPage />} />
        <Route path="/roles" element={<RolesPage />} />
        <Route path="/audit-logs" element={<AuditLogsPage />} />
        <Route path="/profile" element={<ProfilePage />} />
      </Route>

      {/* 未匹配路由 → 首页（守卫会兜底跳登录） */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
