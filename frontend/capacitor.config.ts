import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.hrms.app',
  appName: 'HRMS人力资源管理系统',
  webDir: 'dist',
  server: {
    androidScheme: 'https',
    // API 走 http 明文（暂无证书），需允许 WebView 从 https 页面请求 http 资源
    allowMixedContent: true,
  },
}

export default config
