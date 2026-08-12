# Phase 30: 前端性能优化 - Context

**Gathered:** 2026-06-13
**Status:** Ready for planning

<domain>
## Phase Boundary

基于 Vercel React 最佳实践审计发现的 70+ 问题，从四大维度对 XingRan-Next 前端进行系统化性能优化：

1. **消除瀑布流** — 顺序请求改并行、预取关键数据
2. **减小包大小** — 代码分割、路由懒加载、重库按需加载
3. **修复重渲染** — 不稳定依赖、过度渲染、缺失 memo
4. **提升 JS 性能** — 防抖、虚拟滚动、计算下推

**目标可量化指标**：
- 首屏 JS ≤ 500KB（gzip 后）
- 首屏路由 LCP ≤ 2.5s
- vendor chunk 拆分：react / antd / echarts / three.js 独立
- 资产列表（43 列）启用虚拟滚动

**不包含：**
- 不修改业务功能（仅优化性能）
- 不引入 SSR / RSC（Vite SPA 不变）
- 不引入 PWA / Service Worker / CDN（基础设施改造属独立阶段）
- 不做图片优化（Image Lazy Load、CDN 加速属独立阶段）
- 不修改后端 API
</domain>

<decisions>
## Implementation Decisions

### Wave 划分与执行策略

#### D-01: 阶段结构
**决策：** 单一大型阶段（Phase 30 内 4 个 Wave）

不拆为 30A/30B/30C/30D 子阶段。本阶段审计一次性完成，但内部按 Wave 切分提交粒度。

#### D-02: Wave 1（基础设施）三件套
**决策：** 配置 + 分析器 + 样例

Wave 1 交付物：
1. 集成 `rollup-plugin-visualizer`（Vite bundle analyzer）
2. 配置代码分割基础（路由级 + vendor 分块策略）
3. 实现一个示例路由的懒加载作为模板（建议 `pages/operations/assets/index.tsx`）

**验收：** Wave 1 完成后能看到初始 bundle 大小、首屏路由加载时各 chunk 大小。

#### D-03: Wave 顺序
**决策：** 基础设施 → 重库 → 查询 → 渲染

```
Wave 1: 基础设施    (Vite 配置、analyzer、样例)
Wave 2: 重库按需    (three.js/echarts/xlsx/md-editor)
Wave 3: 查询层      (React Query 推广、字典全局化)
Wave 4: 渲染层      (memo、虚拟滚动、ESLint 规则)
```

理由：底层先稳定（Wave 1 拿到测量基线），再优化顶层体验（Wave 4 视觉与交互）。

#### D-04: 验收方式
**决策：** Lighthouse + bundle 报告对比

每个 Wave 完成后：
- 运行 Lighthouse（desktop / mobile 各一次）
- 重新生成 bundle 报告
- 对比前后的首屏 JS、LCP、FCP、TTI
- 量化指标差异作为完成依据

#### D-05: 性能预算
**决策：** 两阶段预算

| 指标 | 预算值 | 说明 |
|------|--------|------|
| 初始 JS（gzip） | ≤ 500KB | 主入口 + 首屏路由 |
| 首屏路由 LCP | ≤ 2.5s | Lighthouse mobile |
| vendor chunk | 独立拆出 | react / antd / echarts / three.js |
| 重库 chunk | 按需加载 | 三个独立 chunk + fallback |

### 代码分割与重库按需加载

#### D-06: 按需加载重库名单
**决策：** 4 库按需加载

| 库 | 触发场景 | 引入方式 |
|----|----------|----------|
| `three.js` | 3D 场景页（楼宇可视化、CAD 编辑器） | `import()` 动态加载 |
| `echarts` | 仪表盘页、监控页、统计图表 | `import()` 动态加载 |
| `xlsx` | 资产/工位/设备导入按钮点击时 | `import()` 动态加载 |
| `@uiw/react-md-editor` | 知识库编辑页 | `import()` 动态加载 |

其他必要依赖保持同步加载（如 `sm-crypto` 因全站加密接口需保持同步）。

#### D-07: vendor chunk 拆分策略
**决策：** 主要重库独立拆出

`vite.config.ts` `build.rollupOptions.output.manualChunks`：
```
react / react-dom → vendor-react
antd → vendor-antd
echarts → vendor-echarts
three.js → vendor-three
其他第三方 → vendor-commons
业务代码 → 按路由自动 split
```

#### D-08: 路由懒加载粒度
**决策：** 全部业务路由懒加载

非首屏必须的路由（资产、VDI、仪表盘、设置、个人中心等）全部 `React.lazy`。首屏（Login、Layout、Home Dashboard 框架）保持同步。

**实施模式：**
```tsx
// 使用 React.lazy + Suspense 包装
const AssetList = lazy(() => import('@/pages/operations/assets'));
```

#### D-09: Loading 体验
**决策：** AntD Spin + Skeleton

- **顶层 fallback**：`Suspense fallback={<Spin size="large" />}` 包裹整个布局
- **页面内骨架**：使用 AntD `Skeleton` 组件占位（按页面结构定 skeleton rows）
- **不使用** nprogress 进度条（与 AntD 风格不匹配）
- **不使用** 静默空白（用户易误以为页面崩溃）

### 数据获取与缓存策略

#### D-10: React Query 迁移范围
**决策：** 中等范围迁移

**迁到 React Query：**
- `sys_dict` 字典查询（高频共享，全局化）
- 下拉选项数据（部门树、角色列表、字典数据）
- 列表页面（与 `useTableManager` 集成）

**保持 useState + useEffect：**
- 详情页（一次性表单）
- 简单模态框表单

#### D-11: 字典全局化缓存策略
**决策：** 5 分钟 staleTime + 手动失效

```ts
{
  staleTime: 5 * 60 * 1000,      // 5 分钟
  gcTime: 30 * 60 * 1000,         // 30 分钟
  refetchOnWindowFocus: false,    // 切窗口不重拉
}
```

**失效机制：**
- 字典管理页面（`pages/system/dict`）修改后调用 `queryClient.invalidateQueries({ queryKey: ['dict'] })`
- 用户登录后立即失效用户相关 query

**多页共享**：所有 `useQuery({ queryKey: ['dict', dictType] })` 共享同一份数据。

#### D-12: 列表分页查询策略
**决策：** useQuery 跟随参数重建

```ts
useQuery({
  queryKey: ['list', 'workstations', { page, pageSize, filters }],
  queryFn: () => workstationApi.list({ current: page, pageSize, ...filters }),
  placeholderData: keepPreviousData,  // 分页切换时保留旧数据，避免闪烁
})
```

分页、过滤、搜索都由后端承担。前端用 `keepPreviousData` 减少 loading 闪烁。

#### D-13: 迁移路径
**决策：** 逐页面推进

```
Step 1: 字典查询 (pages/system/dict/* + 全局 dict hooks)
Step 2: 下拉选项 (dept/role/menu 等公共下拉)
Step 3: 列表页面 (operations 优先，与 useTableManager 集成)
Step 4: 其他模块
```

每个 Step 完成后该页面享受 React Query 收益（自动缓存、refetch、stale 处理），逐步推开。

### 重渲染、虚拟滚动与 ESLint 防回归

#### D-14: Memo 策略
**决策：** 针对性 memo

**加 memo 的场景**（已知热点）：
- 资产列表 43 列 Table（行渲染 + 列渲染）
- VDI 虚拟机列表（每行操作按钮 + 状态变化）
- 仪表盘 Widget 渲染（多个 widget 同屏）
- 模态框表单（频繁打开/关闭）

**不加 memo 的场景**：
- 简单展示组件（一次性渲染）
- 静态文案组件
- 列表项数量 < 50 的小型列表

**不引入** `react-scan` 等 devtool（增加包体积）。

#### D-15: 虚拟滚动方案
**决策：** AntD Table virtual

| 表格 | 启用 virtual | 行数阈值 |
|------|--------------|----------|
| 资产列表 | ✓ | 43 列 + 大量行 |
| VDI 虚拟机列表 | ✓ | 行数易过千 |
| 工位列表 | ✓ | 多楼宇可达数千 |
| 部门/角色/菜单 | ✗ | 行数 < 100 |
| 字典管理 | ✗ | 行数 < 50 |

**配置：** `<Table virtual scroll={{ x: 总列宽, y: 600 }} />`

**与 Phase 27 集成**：列自定义与 virtual 不冲突，列宽变化时 AntD 内部已优化。

#### D-16: ESLint 规则
**决策：** 加 5 条性能相关规则

`eslint.config.js` 启用：

| 规则 | 级别 | 说明 |
|------|------|------|
| `react-hooks/exhaustive-deps` | error | 防 useEffect/useMemo 依赖缺失 |
| `react/jsx-no-constructed-context-values` | error | 防 Provider value 每次新建引起全树 re-render |
| `react/jsx-no-unstable-nested-components` | error | 防子组件定义在 render 中（每次新建） |
| `react/jsx-no-unnecessary-fragment` | warn | 减少无意义 Fragment |
| `react/no-array-index-key` | warn | 警告 index 作 key（按需评估） |

CI 中跑 `npm run lint`，error 级别必须为 0。

#### D-17: 重库按需加载的体验保障
**决策：** loading + suspense fallback

**加载中：**
```tsx
<Suspense fallback={<Spin size="large" tip="加载 3D 场景..." />}>
  <Scene3D />
</Suspense>
```

**加载失败：** Wave 2 中可选择性添加 ErrorBoundary（属于 Claude discretion）。

**轻代码增量**：`<Suspense>` 包裹纯声明式，运行时零开销。

### Claude's Discretion

以下方面可由实现者决定：

1. **Memo 具体边界** — 哪些组件算"已知热点"由 Profiler 测量决定
2. **Lighthouse 测试环境** — 测试网络/CPU 节流配置、基线测量时点
3. **字典 queryKey 工厂** — 命名规范（`['dict', dictType]` vs `dict:type:{dictType}`）
4. **React Query Devtools** — dev 环境是否启用 `ReactQueryDevtools` 浮动面板
5. **错误重试退避** — `retry: 1` 是否够，5xx 错误是否需要指数退避
6. **列表查询 staleTime** — 列表 staleTime（建议 30s 内）
7. **vendor chunk 阈值** — chunkSizeWarningLimit（建议 500KB）
8. **重库加载错误边界** — Wave 2 是否给 Three.js/MD 编辑器加 ErrorBoundary
9. **Antd locale 拆分** — zh_CN 单独 chunk 化
10. **预取策略** — hover 路由链接预取下一路由 chunk

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目文档
- `CLAUDE.md` — 项目概述和架构设计、前端 React 最佳实践
- `docs/项目概述和架构设计.md` — 整体架构
- `docs/开发规范.md` — Handler-Service 模式、命名规范
- `docs/API响应规范.md` — API 响应格式

### 前端技术栈参考
- `xingran-react-frontend/package.json` — 依赖列表（React 19.2 / Vite 7.2 / Ant Design 6.1 / TanStack Query v5.90）
- `xingran-react-frontend/vite.config.ts` — Vite 构建配置（需修改）
- `xingran-react-frontend/src/App.tsx` — 应用根组件（Provider 嵌套、QueryClient 初始化）
- `xingran-react-frontend/src/main.tsx` — 应用入口

### 现有 Hooks / Stores
- `xingran-react-frontend/src/hooks/useTableManager.ts` — 列表分页管理（Wave 3 集成目标）
- `xingran-react-frontend/src/hooks/useColumnConfig.ts` — 列自定义（Phase 27）
- `xingran-react-frontend/src/hooks/useWidgetData.ts` — 仪表盘 Widget 数据
- `xingran-react-frontend/src/hooks/useWidgetPolling.ts` — Widget 轮询
- `xingran-react-frontend/src/hooks/useWebSocket.ts` — WebSocket 实时数据
- `xingran-react-frontend/src/store/` — 9 个 Zustand store

### API 封装
- `xingran-react-frontend/src/lib/api.ts` — 核心 API（含加密、token 刷新）
- `xingran-react-frontend/src/lib/opsApi.ts` — Operations CRUD 工厂

### 已知性能问题区域
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 资产列表（43 列，已确认加 virtual）
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — 工位列表（行数大）
- `xingran-react-frontend/src/pages/operations/vdi/*` — VDI 虚拟机列表
- `xingran-react-frontend/src/components/three/*` — Three.js 场景组件（按需加载）
- `xingran-react-frontend/src/pages/dashboard-system/*` — 可配置仪表盘

### 前序阶段上下文
- `.planning/phases/27-column-customization/27-CONTEXT.md` — 列自定义（Wave 4 virtual 不冲突）
- `.planning/phases/28-workstation-device-association/28-CONTEXT.md` — 工位设备关联（瀑布流子表格）
- `.planning/phases/29-sys-dict/29-CONTEXT.md` — sys_dict（Wave 3 字典全局化基础）

### Codebase 分析
- `.planning/codebase/STACK.md` — 技术栈详情
- `.planning/codebase/STRUCTURE.md` — 目录结构
- `.planning/codebase/CONCERNS.md` — 已知技术债（已记录 useEffect 稳定性问题）

### Vercel React 最佳实践（审计源）
- 来源：Vercel Engineering React/Next.js 性能指南
- 通过 `superpowers:vercel-react-best-practices` skill 可访问
- 重点：消除瀑布流、并行请求、Suspense、useTransition、Server Components（本项目不适用）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **TanStack React Query v5.90** — 已安装但仅在 App.tsx 初始化 `QueryClient`，尚未在业务中广泛使用（Wave 3 主要工作）
- **AntD Table virtual** — Ant Design 6.1 已支持 `virtual` 属性（Wave 4 直接启用）
- **AntD Spin / Skeleton** — 已使用，Loading 体验可直接复用
- **Zustand stores** — 9 个 store 已就绪，可承担客户端状态
- **opsApi / api 封装** — 已统一封装，React Query 改造时直接接 `queryFn`
- **Vite 7.2** — 支持 `manualChunks`、`dynamicImport`、tree-shaking

### Established Patterns
- **Handler-Service 模式** — 后端 API 标准模式（与前端性能优化无关）
- **包前缀 `xingran:`** — Redis key 约定（性能优化不涉及）
- **ESLint + Prettier** — 已有规则基础（Wave 4 添加 5 条性能规则）
- **TypeScript strict** — 已有类型检查

### Integration Points
- **App.tsx** — 顶层 Provider（QueryClient、AntD ConfigProvider）— Wave 3 需扩展 QueryClient 默认值
- **Vite 配置** — `vite.config.ts` 需新增 `rollup-plugin-visualizer`、`manualChunks`
- **路由配置** — `src/router/DynamicRoutes.tsx` — 需支持 lazy 加载
- **ESLint 配置** — `eslint.config.js` — 需添加 5 条性能规则
- **包大小阈值** — Vite `build.chunkSizeWarningLimit` 需调整

### Known Constraints
- Vite SPA 不支持 SSR/RSC（Vercel RSC 建议在本项目不适用）
- AntD ConfigProvider 包裹全局，locale 切换需重新加载
- 部分重库（如 sm-crypto）被加密接口广泛使用，不适合按需
- TanStack Query v5 与 React 19 严格模式下需注意 staleTime 语义
- 506 个 TS/TSX 文件批量改造需谨慎，建议分 Wave 推进

</code_context>

<specifics>
## Specific Ideas

### Wave 1 基础设施示例
```ts
// vite.config.ts 新增
import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-antd': ['antd'],
          'vendor-echarts': ['echarts', 'echarts-for-react'],
          'vendor-three': ['three', '@react-three/fiber', '@react-three/drei'],
        },
        chunkSizeWarningLimit: 500,  // KB
      },
    },
  },
  plugins: [react(), visualizer({ filename: 'dist/stats.html', gzipSize: true })],
});
```

### Wave 2 路由懒加载示例
```tsx
// src/router/DynamicRoutes.tsx
import { lazy, Suspense } from 'react';
import { Spin } from 'antd';

const AssetList = lazy(() => import('@/pages/operations/assets'));
const WorkstationList = lazy(() => import('@/pages/operations/workstations'));
const DashboardSystem = lazy(() => import('@/pages/dashboard-system'));

// Suspense fallback
<Suspense fallback={<Spin size="large" />}>
  <AssetList />
</Suspense>
```

### Wave 2 重库按需加载示例
```tsx
// 3D 场景页
const Scene3D = lazy(() => import('@/components/three/BuildingScene'));

function BuildingPage() {
  return (
    <Suspense fallback={<Spin tip="加载 3D 场景..." />}>
      <Scene3D />
    </Suspense>
  );
}

// ECharts 图表组件（按需）
const ReactECharts = lazy(() => import('echarts-for-react'));
```

### Wave 3 React Query 字典全局化示例
```ts
// src/hooks/useDict.ts
import { useQuery } from '@tanstack/react-query';
import { post } from '@/lib/api';

export function useDict(dictType: string) {
  return useQuery({
    queryKey: ['dict', dictType],
    queryFn: async () => {
      const result = await post('/system/dicts/data/list', {
        dictType,
        current: 1,
        pageSize: 100,
      });
      return result.data?.list ?? [];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
  });
}

// 失效（字典管理页）
import { useQueryClient } from '@tanstack/react-query';
const qc = useQueryClient();
qc.invalidateQueries({ queryKey: ['dict'] });
```

### Wave 4 虚拟表示例
```tsx
// 资产列表（43 列）
<Table
  virtual
  scroll={{ x: 4200, y: 600 }}
  columns={columns}  // Phase 27 列自定义
  dataSource={data}
  rowKey="id"
/>
```

### Wave 4 ESLint 配置
```js
// eslint.config.js
{
  rules: {
    'react-hooks/exhaustive-deps': 'error',
    'react/jsx-no-constructed-context-values': 'error',
    'react/jsx-no-unstable-nested-components': 'error',
    'react/jsx-no-unnecessary-fragment': 'warn',
    'react/no-array-index-key': 'warn',
  },
}
```

</specifics>

<deferred>
## Deferred Ideas

以下想法不在本阶段范围：

### 基础设施类
- **PWA / Service Worker** — 离线缓存、后台同步属于独立基础设施阶段
- **CDN 加速** — 静态资源 CDN 部署属运维层面
- **Image Optimization** — 图片懒加载、CDN 图床、WebP 转换属独立阶段
- **HTTP/3 / Brotli 压缩** — 服务端配置层面
- **预渲染 (Prerender)** — 适合营销页，本项目是后台管理系统不适用

### 架构改造
- **SSR / RSC** — 从 Vite SPA 迁移到 Next.js/Remix 改造巨大，超出性能优化范围
- **微前端 (Module Federation)** — 拆分多 bundle 部署，本项目单体应用足够

### 功能扩展
- **PWA 推送通知** — 通知属独立功能
- **离线表单编辑** — Service Worker 配套功能
- **路由级骨架屏细化** — 当前 Skeleton 已够用，过度细化 ROI 低
- **Service Worker 后台数据同步** — 配套 PWA
- **Web Vitals 上报** — 监控告警平台属独立阶段

</deferred>

---

*Phase: 30-js*
*Context gathered: 2026-06-13*
