import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  optimizeDeps: {
    // 包含 sm-crypto 进行预构建，使其与 ES 模块兼容
    include: ['sm-crypto'],
  },
  server: {
    port: 4000,
    host: true,
    proxy: {
      '/api': {
        target: 'http://10.62.10.33:9000',
        changeOrigin: true,
        secure: false,
      },
    },
  },
})