# Phase 30: 前端性能优化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-13
**Phase:** 30-js
**Areas discussed:** Wave 划分, 代码分割与重库按需, React Query 推广与字典全局化, memo/虚拟滚动/ESLint

---

## Area 1: Wave 划分（执行策略）

| Option | Description | Selected |
|--------|-------------|----------|
| 单一大型阶段（推荐） | 在 Phase 30 内按 4 个 Wave 执行（基础设施→数据获取→渲染→JS优化） | ✓ |
| 拆分为 4 个子阶段 | 把 Phase 30 拆成 30A/30B/30C/30D，边界清晰但管理成本翻倍 | |
| 拆分为 2 个子阶段 | 30A 基础设施 + 30B 应用层 | |

**User's choice:** 单一大型阶段
**Notes:** 用户偏好一次性规划 4 个 Wave，便于统一验收和回顾。

### Sub-question 1.1: Wave 1 内容

| Option | Description | Selected |
|--------|-------------|----------|
| 三件套：配置+分析器+样例（推荐） | Vite 配置 + bundle analyzer + 一个示例路由懒加载模板 | ✓ |
| 仅集成分析器 | 留待后续 wave 再做分割 | |
| 仅 Vite 配置 | 不急 | |

**User's choice:** 三件套

### Sub-question 1.2: Wave 顺序

| Option | Description | Selected |
|--------|-------------|----------|
| 基础设施→重库→查询→渲染（推荐） | 底层先稳定，再做顶层体验 | ✓ |
| 倒序：先见效大的部分 | 先做感知最强（渲染、虚拟滚动） | |
| 合并 Wave 2 和 Wave 3 | 减少 1 个 wave | |

**User's choice:** 基础设施→重库→查询→渲染

### Sub-question 1.3: 验收方式

| Option | Description | Selected |
|--------|-------------|----------|
| Lighthouse + bundle 报告对比（推荐） | 量化指标是验收标准 | ✓ |
| 手工拨测 + 现象记录 | 只手动测交互响应 | |
| 仅 bundle 报告 | 不设定量指标 | |

**User's choice:** Lighthouse + bundle 报告对比

### Sub-question 1.4: 性能预算

| Option | Description | Selected |
|--------|-------------|----------|
| 设两阶段预算（推荐） | 初始 JS ≤ 500KB（gzip）、首屏 LCP ≤ 2.5s、vendor 独立拆 | ✓ |
| 宽松目标 | < 1MB，LCP < 4s | |
| 不设硬预算 | 只求"明显改善" | |

**User's choice:** 设两阶段预算

---

## Area 2: 代码分割与重库按需加载

| Option | Description | Selected |
|--------|-------------|----------|
| 4 库按需（推荐） | three.js / echarts / xlsx / md-editor 按需加载 | ✓ |
| 范围扩大 | 加 sm-crypto、@dnd-kit | |
| 按体积大优先 | 全按需 | |

**User's choice:** 4 库按需
**Notes:** sm-crypto 因全站加密接口同步使用，保持同步；@dnd-kit 拖拽页面较少，可不按需。

### Sub-question 2.2: vendor chunk 拆分

| Option | Description | Selected |
|--------|-------------|----------|
| 仅主要重库独立拆出（推荐） | react/antd/echarts/three.js 独立 chunk | ✓ |
| 手动细粒度拆 | 每个库独立 chunk | |
| 不手动拆 | 依靠 Vite 默认 | |

**User's choice:** 仅主要重库独立拆出

### Sub-question 2.3: 懒加载粒度

| Option | Description | Selected |
|--------|-------------|----------|
| 全部业务路由懒加载（推荐） | 资产/VDI/仪表盘等都懒，首屏同步 | ✓ |
| 白名单重页面懒加载 | 只懒加载已知重的页面 | |
| 首屏之外全懒 | 全部业务路由 | |

**User's choice:** 全部业务路由懒加载

### Sub-question 2.4: Loading 体验

| Option | Description | Selected |
|--------|-------------|----------|
| AntD Spin + Skeleton（推荐） | 顶层 Spin + 页面内 Skeleton | ✓ |
| 进度条 | nprogress | |
| 空白静默 | 无提示 | |

**User's choice:** AntD Spin + Skeleton

---

## Area 3: React Query 推广与字典全局化

| Option | Description | Selected |
|--------|-------------|----------|
| 中等范围（推荐） | 字典、下拉选项、列表页迁；详情/表单保留 | ✓ |
| 全面推广 | 所有 useState+useEffect 都迁 | |
| 仅字典/下拉 | 范围最小 | |

**User's choice:** 中等范围

### Sub-question 3.2: 字典缓存策略

| Option | Description | Selected |
|--------|-------------|----------|
| 5 分钟 staleTime + 手动失效（推荐） | staleTime 5min、gcTime 30min、字典管理页手动失效 | ✓ |
| 10 分钟 + 定时刷新 | 加 WebSocket 推送 | |
| 静态化 | staleTime Infinity | |

**User's choice:** 5 分钟 staleTime + 手动失效

### Sub-question 3.3: 列表分页策略

| Option | Description | Selected |
|--------|-------------|----------|
| useQuery 跟随参数重建（推荐） | useTableManager 集成 + keepPreviousData | ✓ |
| useInfiniteQuery | 无限滚动 | |
| 仍用 useState + fetch | 不迁 | |

**User's choice:** useQuery 跟随参数重建

### Sub-question 3.4: 迁移路径

| Option | Description | Selected |
|--------|-------------|----------|
| 逐页面推进（推荐） | 字典→下拉→列表→其他 | ✓ |
| 按模块全量迁移 | 一个模块一次全迁 | |
| 仅创建基础设施不迁 | 只装不迁 | |

**User's choice:** 逐页面推进

---

## Area 4: memo/虚拟滚动/ESLint

| Option | Description | Selected |
|--------|-------------|----------|
| 针对性 memo（推荐） | 已知热点（资产/VDI/仪表盘/模态框） | ✓ |
| 全面 memo 化 | 所有组件/回调/派生值 | |
| 仅修依赖 | 不主动加 memo | |

**User's choice:** 针对性 memo

### Sub-question 4.2: 虚拟滚动方案

| Option | Description | Selected |
|--------|-------------|----------|
| AntD Table virtual（推荐） | 资产/VDI/工位启用 | ✓ |
| react-window/virtuoso | 重写 Table | |
| 仅 AntD virtual（资产） | 范围最小 | |

**User's choice:** AntD Table virtual

### Sub-question 4.3: ESLint 规则

| Option | Description | Selected |
|--------|-------------|----------|
| 加 5 条性能相关规则（推荐） | exhaustive-deps、no-constructed-context-values、no-unstable-nested-components、no-unnecessary-fragment、no-array-index-key | ✓ |
| 仅补 3 条主要规则 | 关键 3 条 | |
| 不调整 ESLint | 不加新规则 | |

**User's choice:** 加 5 条性能相关规则

### Sub-question 4.4: 重库按需加载体验

| Option | Description | Selected |
|--------|-------------|----------|
| loading + suspense fallback（推荐） | AntD Spin + Skeleton | ✓ |
| 错误边界 + fallback | ErrorBoundary 友好错误 | |
| 双方案组合 | 错误边界 + Spin | |

**User's choice:** loading + suspense fallback

---

## Claude's Discretion

实现者可自主决定的方面（10 项）：

1. Memo 具体边界（Profiler 测量决定）
2. Lighthouse 测试环境与基线测量
3. 字典 queryKey 工厂命名规范
4. React Query Devtools 启用与否
5. 错误重试退避策略
6. 列表查询 staleTime
7. vendor chunk 阈值
8. 重库加载错误边界
9. Antd locale 拆分
10. 预取策略（hover 预取下一路由）

---

## Deferred Ideas

不在本阶段范围：

- **PWA / Service Worker** — 离线缓存、后台同步属独立基础设施阶段
- **CDN 加速** — 静态资源 CDN 属运维
- **Image Optimization** — 图片懒加载、CDN 图床、WebP 转换
- **HTTP/3 / Brotli 压缩** — 服务端配置
- **预渲染 (Prerender)** — 不适用于后台管理系统
- **SSR / RSC** — Vite SPA 迁移到 Next.js 改造巨大
- **微前端 (Module Federation)** — 单体应用足够
- **PWA 推送通知** — 通知属独立功能
- **离线表单编辑** — Service Worker 配套
- **Web Vitals 上报** — 监控告警平台
- **路由级骨架屏细化** — 当前 Skeleton 已够用
- **Service Worker 后台数据同步** — 配套 PWA
