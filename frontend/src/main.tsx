import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import ErrorBoundary from './components/ErrorBoundary'
import { applyTheme } from './store/theme'

// 启动时应用持久化主题（默认 light），避免暗色模式刷新后丢失
try {
  const raw = localStorage.getItem('hrms_theme')
  const parsed = raw ? (JSON.parse(raw) as { state?: { theme?: string } }) : null
  applyTheme(parsed?.state?.theme === 'dark' ? 'dark' : 'light')
} catch {
  applyTheme('light')
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {/* 全局错误边界：渲染异常时展示可操作错误页，防止黑屏 */}
    <ErrorBoundary>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ErrorBoundary>
  </StrictMode>,
)
