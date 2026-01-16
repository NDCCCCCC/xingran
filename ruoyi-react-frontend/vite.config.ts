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
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // React 核心库
          if (id.includes('react') || id.includes('react-dom') || id.includes('react-router')) {
            return 'react-vendor';
          }
          // UI 组件库
          if (id.includes('antd') || id.includes('@ant-design')) {
            return 'antd';
          }
          // 3D 渲染相关
          if (id.includes('three') || id.includes('@react-three')) {
            return 'three-vendor';
          }
          // 工具库
          if (id.includes('dayjs') || id.includes('axios') || id.includes('sm-crypto')) {
            return 'utils';
          }
          // 运维管理模块
          if (id.includes('@/lib/opsApi') || id.includes('@/types/operations') || id.includes('@/utils/three')) {
            return 'ops-module';
          }
          // 其他返回 vendor
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        },
      },
    },
    // 提高警告阈值
    chunkSizeWarningLimit: 1000,
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
      '/uploads': {
        target: 'http://10.62.10.33:9000',
        changeOrigin: true,
        secure: false,
      },
    },
  },
})