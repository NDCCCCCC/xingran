import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import removeConsole from 'vite-plugin-remove-console'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // 生产构建时移除 console 语句
    removeConsole(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  // 预构建优化配置
  optimizeDeps: {
    // 包含 sm-crypto 和 react-router-dom 进行预构建，使其与 ES 模块兼容
    include: ['sm-crypto', 'react', 'react-dom', 'react-router-dom'],
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
      // 将所有 /api 请求代理到后端服务器
      '/api': {
        target: 'http://10.62.10.33:9000',
        changeOrigin: true,
        // 不需要重写路径，因为后端就是用 /api/v1 前缀
      },
    },
    // Vite 默认支持 SPA fallback，会自动将所有请求返回 index.html
    // 这样直接访问 /dashboard 等子路由时也能正常工作
  },
})

