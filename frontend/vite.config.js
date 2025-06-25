import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // 将所有 /api 开头的请求代理到后端服务
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true, // 需要更改源
        secure: false,      // 不需要 https
      },
    },
  },
}) 