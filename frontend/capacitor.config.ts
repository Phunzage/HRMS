import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.hrms.app',
  appName: 'HRMS人力资源管理系统',
  webDir: 'dist',
  server: {
    // 本地页面用 http://localhost（与线上 http://115.29.197.130 API 同协议，避免 https 页面请求 http API 的混合内容拦截）
    androidScheme: 'http',
    // API 走 http 明文（暂无证书）
    allowMixedContent: true,
  },
}

export default config
