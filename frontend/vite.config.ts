import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  // 使用相对路径产物：桌面端 Electron 以 file:// 加载 index.html，
  // 绝对路径 /assets/... 会解析到文件系统根目录导致白屏
  base: './',
  server: {
    // 开发环境代理：/api 与 /uploads 转发到后端（Gin，监听 8080）
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
