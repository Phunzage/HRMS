import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
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
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
