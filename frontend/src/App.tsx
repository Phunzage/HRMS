import { Routes, Route, Navigate } from 'react-router-dom'
import RequireAuth, { GuestOnly } from './router/guard'
import MainLayout from './layouts/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import EmployeeList from './pages/EmployeeList'
import DepartmentTree from './pages/DepartmentTree'
import AttendancePage from './pages/AttendancePage'
import LeavePage from './pages/LeavePage'
import SalaryPage from './pages/SalaryPage'
import RecruitmentPage from './pages/RecruitmentPage'

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
        <Route path="/" element={<Dashboard />} />
        <Route path="/employees" element={<EmployeeList />} />
        <Route path="/departments" element={<DepartmentTree />} />
        <Route path="/attendance" element={<AttendancePage />} />
        <Route path="/leaves" element={<LeavePage />} />
        <Route path="/salaries" element={<SalaryPage />} />
        <Route path="/recruitment" element={<RecruitmentPage />} />
      </Route>

      {/* 未匹配路由 → 首页（守卫会兜底跳登录） */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
