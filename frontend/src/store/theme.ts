import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// 主题状态：light / dark（daisyUI 内置主题，dark 由 data-theme="dark" 生效）
type Theme = 'light' | 'dark'

interface ThemeState {
  theme: Theme
  toggle: () => void
  setTheme: (theme: Theme) => void
}

// 应用主题到 <html data-theme>，供初始化与 toggle 共用
export function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme)
}

// 全局主题状态，持久化到 localStorage（key: hrms_theme）
export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'light',
      setTheme: (theme) => {
        set({ theme })
        applyTheme(theme)
      },
      toggle: () => {
        const next: Theme = get().theme === 'light' ? 'dark' : 'light'
        get().setTheme(next)
      },
    }),
    {
      name: 'hrms_theme',
    },
  ),
)
