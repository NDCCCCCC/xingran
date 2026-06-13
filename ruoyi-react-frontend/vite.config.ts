import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import removeConsole from 'vite-plugin-remove-console'
import { visualizer } from 'rollup-plugin-visualizer'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    // 生产构建时移除 console 语句
    removeConsole(),
    // Bundle analyzer (opt-in via ANALYZE=true env var)
    // 输出 dist/stats.html，包含 gzip/brotli 大小
    ...(process.env.ANALYZE === 'true'
      ? [
          visualizer({
            filename: 'dist/stats.html',
            gzipSize: true,
            brotliSize: true,
            template: 'treemap',
          }),
        ]
      : []),
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
        // D-07 vendor chunk 拆分策略
        // 顺序很重要：three-family 必须先于 react-core 检查，避免 @react-three 被错误归入 vendor-react
        manualChunks(id) {
          // 业务代码保持按路由自动 split（不强制归入 vendor chunk）
          if (!id.includes('node_modules')) {
            return undefined
          }
          // 3D 渲染相关（three 家族）
          if (
            id.includes('three') ||
            id.includes('@react-three') ||
            id.includes('@react-spring/three') ||
            id.includes('maath')
          ) {
            return 'vendor-three'
          }
          // ECharts 图表
          if (id.includes('echarts') || id.includes('zrender')) {
            return 'vendor-echarts'
          }
          // Ant Design UI 库（含 rc-* 工具包）
          if (id.includes('antd') || id.includes('@ant-design') || id.includes('/rc-')) {
            return 'vendor-antd'
          }
          // React 核心 + 路由
          // 注意：必须排除 @react-three 等以避免被此规则捕获
          if (
            id.includes('/react-dom/') ||
            id.includes('/react-router') ||
            id.includes('/scheduler/') ||
            (id.includes('/react/') && !id.includes('@react-'))
          ) {
            return 'vendor-react'
          }
          // 工具库（D-06：sm-crypto 保持同步，全站加密接口需要）
          if (id.includes('dayjs') || id.includes('axios') || id.includes('sm-crypto')) {
            return 'vendor-utils'
          }
          // 其他第三方依赖
          return 'vendor-commons'
        },
      },
    },
    // D-05 / D-07：500KB gzip 预算
    chunkSizeWarningLimit: 500,
  },
  server: {
    port: 4000,
    host: true,
    proxy: {
      // 将所有 /api 请求代理到后端服务器
      '/api': {
        target: 'http://localhost:9000',
        changeOrigin: true,
        // 不需要重写路径，因为后端就是用 /api/v1 前缀
      },
      // 将 /uploads 请求代理到后端服务器（静态文件）
      '/uploads': {
        target: 'http://localhost:9000',
        changeOrigin: true,
      },
    },
    // Vite 默认支持 SPA fallback，会自动将所有请求返回 index.html
    // 这样直接访问 /dashboard 等子路由时也能正常工作
  },
})