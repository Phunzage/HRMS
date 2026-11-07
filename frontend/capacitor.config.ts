import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.hrms.app',
  appName: 'HRMS人力资源管理系统',
  webDir: 'dist',
  server: {
    // 远程加载模式：WebView 直接加载线上前端（与 Windows 桌面端 loadURL 一致，永远最新）。
    // 缺点：需联网才能打开应用（离线不可用）；数据本身始终实时（同一后端）。
    url: 'https://hrms.phunzage.fun',
    // 允许导航到 API 域（同源，实际无跨域导航，保留兜底）
    allowNavigation: ['hrms.phunzage.fun', '*.phunzage.fun'],
    androidScheme: 'https',
    allowMixedContent: false,
  },
}

export default config
