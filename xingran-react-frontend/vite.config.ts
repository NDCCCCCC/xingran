import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import removeConsole from 'vite-plugin-remove-console'
import { visualizer } from 'rollup-plugin-visualizer'
import path from 'path'
import fs from 'fs'

// ============================================================================
// 预计算三组「包族」集合，让 vendor chunk 切分遵循真实依赖图，避免跨 chunk 引用环
// ----------------------------------------------------------------------------
// 背景：手动按包名切 chunk 会与真实依赖图冲突——Rollup 把共享模块 hoist 到某个 chunk，
// 若该 chunk 与 React/three 形成双向引用，模块求值时 React 绑定未就绪 →
// "Cannot read properties of undefined (reading 'createContext' / 'useLayoutEffect')"。
// 做法：配置加载时扫描 node_modules 的 package.json（dependencies + peerDependencies），
// 计算三组传递闭包，按依赖关系而非包名分组，确保每个 vendor chunk 只单向依赖 vendor-react。
//   • THREE_FAMILY：所有（直接/间接）依赖 three 的包 → vendor-three（含 three-stdlib、
//     troika-*、camera-controls、meshline 等扩展，自动覆盖新增）。
//   • REACT_FAMILY：所有依赖 react/react-dom 的包 → vendor-react（默认兜底）。
//   • MARKDOWN_FAMILY：markdown 解析生态（unified/remark/rehype/micromark/mdast/hast 等）
//     及其**向下**纯工具依赖（parse5、hastscript、nth-check、github-slugger 等长尾），
//     排除 REACT/THREE family → vendor-markdown（自包含叶子，不引用 vendor-react）。
//     react-markdown 本身依赖 React，属 REACT_FAMILY，留在 vendor-react。
// ============================================================================
function computePackageFamilies() {
  const nm = 'node_modules'
  const pkgDeps: Record<string, string[]> = {}
  const empty = { three: new Set<string>(['three']), react: new Set<string>(), markdown: new Set<string>() }
  if (!fs.existsSync(nm)) return empty
  const readDeps = (pj: string): string[] | null => {
    try {
      const j = JSON.parse(fs.readFileSync(pj, 'utf8'))
      return Object.keys({ ...(j.dependencies || {}), ...(j.peerDependencies || {}) })
    } catch {
      return null
    }
  }
  try {
    for (const entry of fs.readdirSync(nm, { withFileTypes: true })) {
      if (!entry.isDirectory() || entry.name.startsWith('.')) continue
      if (entry.name.startsWith('@')) {
        const scope = path.join(nm, entry.name)
        for (const sub of fs.readdirSync(scope, { withFileTypes: true })) {
          if (!sub.isDirectory()) continue
          const deps = readDeps(path.join(scope, sub.name, 'package.json'))
          if (deps) pkgDeps[`${entry.name}/${sub.name}`] = deps
        }
      } else {
        const deps = readDeps(path.join(nm, entry.name, 'package.json'))
        if (deps) pkgDeps[entry.name] = deps
      }
    }
  } catch {
    return empty
  }
  // 向上闭包：roots 本身 + 所有直接/间接依赖 roots 的包
  const closureUp = (roots: string[]): Set<string> => {
    const fam = new Set<string>(roots)
    let changed = true
    let guard = 0
    while (changed && guard++ < 50) {
      changed = false
      for (const [pkg, deps] of Object.entries(pkgDeps)) {
        if (fam.has(pkg)) continue
        if (deps.some((d) => fam.has(d))) {
          fam.add(pkg)
          changed = true
        }
      }
    }
    return fam
  }
  const three = closureUp(['three'])
  const react = closureUp(['react', 'react-dom'])
  // 向下闭包：markdown 种子的传递依赖，排除 REACT/THREE family（保证纯逻辑、不引 React）
  const seedRe = /^(unified|remark-[\w-]+|rehype-[\w-]+|micromark[\w-]*|mdast-[\w-]+|unist-[\w-]+|hast-util-[\w-]+|hastscript|vfile[\w-]*|property-information|html-url-attributes|react-markdown)$/
  const markdown = new Set<string>()
  const stack: string[] = Object.keys(pkgDeps).filter((p) => seedRe.test(p))
  while (stack.length) {
    const p = stack.pop() as string
    if (markdown.has(p) || react.has(p) || three.has(p)) continue
    markdown.add(p)
    for (const d of pkgDeps[p] || []) stack.push(d)
  }
  return { three, react, markdown }
}
const { three: THREE_FAMILY, markdown: MARKDOWN_FAMILY } = computePackageFamilies()

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
        // ============================================================
        // vendor chunk 拆分策略（健壮版）
        // ------------------------------------------------------------
        // 核心原则：**未知包默认归入 vendor-react**。
        //
        // createContext/useLayoutEffect undefined 这类错误的根因，是「使用 React
        // 的包」与「React 本身」被拆到不同 chunk，形成跨 chunk 引用环，导致模块
        // 求值时 React 绑定尚未就绪。解决方法是让整个 React 生态保持原子性——
        // 只把「确认不依赖 React」的库拆出去（单向依赖，不会成环）：
        //   - Markdown 解析生态 → vendor-markdown
        //   - 纯工具库（dayjs/axios 等）→ vendor-utils
        //   - 大型独立库（echarts/three/xlsx）→ 各自 chunk
        // 其余一律 vendor-react，任何新增的 React 相关包自动安全归位。
        // ============================================================
        manualChunks(id) {
          // 业务代码保持按路由自动 split（不强制归入 vendor chunk）
          if (!id.includes('node_modules')) {
            return undefined
          }

          // 统一提取顶层包名（如 @react-three/drei、react-markdown、dayjs），用于族集合判断
          const pm = id.match(/node_modules[/\\]((?:@[\w-]+[/\\])?[\w-]+)/)
          const pkgName = pm ? pm[1].replace(/[/\\]/g, '/') : ''

          // Markdown 编辑器（按需加载，仅在通知公告表单打开时加载）
          if (pkgName === '@uiw/react-md-editor' || id.includes('@uiw_react-md-editor')) {
            return 'vendor-md-editor'
          }

          // @uiw/react-markdown-preview 是 React 组件（内部用 react-markdown），必须与 React
          // 同 chunk（vendor-react），否则会被 Rollup 拆到 vendor-markdown，形成与 vendor-react
          // 的双向引用环。其余 @uiw/*（copy-to-clipboard 等）同理归入 vendor-react。
          if (pkgName.startsWith('@uiw/')) {
            return 'vendor-react'
          }

          // 3D 渲染：所有传递依赖 three 的包（THREE_FAMILY 在配置加载时计算）→ vendor-three
          // @react-three/* 及 three 扩展库依赖 React，单向 import vendor-react（只进不出），
          // 从而避免 vendor-react 反向引用 vendor-three 的 three 核心而形成双向环。
          if (pkgName && THREE_FAMILY.has(pkgName)) {
            return 'vendor-three'
          }

          // Markdown 解析生态及其纯工具依赖（MARKDOWN_FAMILY 向下闭包，自动覆盖 parse5、
          // hastscript、nth-check、github-slugger 等长尾）。该集合已排除 REACT/THREE family，
          // 故 vendor-markdown 是自包含叶子，不引用 vendor-react，保证无环。
          if (pkgName && MARKDOWN_FAMILY.has(pkgName)) {
            return 'vendor-markdown'
          }

          // echarts-for-react 依赖 React，必须在 echarts 规则之前 → 归入 vendor-react
          // 否则它会被分到 vendor-echarts，导致 React 在两个 chunk 中，引发 createContext undefined
          if (id.includes('echarts-for-react')) {
            return 'vendor-react'
          }

          // ECharts 图表核心（不依赖 React）
          if (pkgName === 'echarts' || pkgName === 'zrender') {
            return 'vendor-echarts'
          }

          // xlsx 表格处理（按需加载，不依赖 React）
          if (pkgName === 'xlsx') {
            return 'vendor-xlsx'
          }

          // 兜底：React 生态整体保持原子性
          // 注意：dayjs/axios/sm-crypto 等纯工具库不再单独拆 vendor-utils——实测 Rollup 会
          // 把与 React 生态共享的辅助模块 hoist 进 vendor-react，单独拆 utils 反而形成双向
          // 引用环。统一留在 vendor-react 保证 chunk 依赖图为 DAG（无环）。
          // react / react-dom / scheduler / antd / @ant-design / rc-* / react-router /
          // @tanstack/react-query / zustand / @dnd-kit/* / react-grid-layout /
          // react-markdown / @uiw/react-baidu-map / dayjs / axios / 通用工具(debug/ms) 等全部在此
          return 'vendor-react'
        },
      },
    },
    // vendor-react 因包含 React 19 + antd 6 整套生态，约 2.5-3MB 属预期（gzip 后约 900KB）。
    // 其他 chunk 超过该阈值才告警。
    chunkSizeWarningLimit: 3000,
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
