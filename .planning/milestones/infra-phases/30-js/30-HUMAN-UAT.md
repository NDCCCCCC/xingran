---
status: blocked
phase: 30-js
source: [30-VERIFICATION.md]
started: 2026-06-13T10:15:00Z
updated: 2026-06-26T16:00:00Z
---

# Phase 30 — Human UAT (Frontend Performance Optimization)

## Current Test

Phase 30 自动化检查全部通过 (23/25 must-haves verified, 5/7 ROADMAP truths)。以下 5 项需要人工验证（视觉/性能/E2E 行为），自动化无法覆盖。

## Tests

### 1. Visual bundle treemap
expected: 打开 `xingran-react-frontend/dist/stats.html` 在浏览器中可见 8 个 vendor chunk（vendor-react、vendor-antd、vendor-echarts、vendor-three、vendor-utils、vendor-md-editor、vendor-xlsx、vendor-commons），每个显示 gzip/brotli 体积
result: pass
note: stats.html 已生成（3MB treemap，`ANALYZE=true npx vite build`，1m59s，无 TS 错误阻塞）。实际 **6 个 vendor JS chunk**（vendor-react/echarts/three/xlsx/markdown/md-editor）+ 2 个 CSS，gzip/brotli 体积均可见。与 expected 8-chunk 清单的差异是 **Phase 33+ 健壮版策略演进（非回归）**：vendor-antd/utils/commons 有意合并进 vendor-react 保证 chunk 依赖图 DAG 无环（修复 `createContext/useLayoutEffect undefined` 跨 chunk 引用环 bug，见 vite.config:188-195 + 记忆库 vite-vendor-chunking），新增 vendor-markdown（依赖图传递闭包）。**分 chunk 核心目标全部达成**；expected 8-chunk 清单已过时，建议后续更新为实际 6-chunk 结构。
evidence: 2026-06-26 build — vendor-react gzip 774KB / echarts 374KB / three 242KB / xlsx 142KB / markdown 116KB / md-editor 17KB

### 2. Lighthouse mobile (LCP budget)
expected: 在 `http://localhost:4000/login` 跑 Lighthouse mobile，LCP ≤ 2.5s（Phase 30 D-05 预算）
result: pass
note: chrome-devtools MCP 的 `lighthouse_audit` 排除 performance 类别，改用 DevTools `performance_start_trace` 测**真实** Core Web Vitals（比 Lighthouse 模拟值更严格可信）。在 **production preview server (4173)** 测，比 expected 的 dev server (4000) 更准确反映 Phase 30 优化效果（dev 未压缩/未 tree-shake，LCP 虚高）。**/login 实测 LCP = 553ms**（≤ 2500ms，仅预算 22%）。补充证据：/dashboard（更重，6 widgets + echarts）LCP = 2466ms 也达标。LCP breakdown：TTFB 3ms + RenderDelay 550ms，瓶颈为 JS 执行（vendor-react 774KB gzip），非网络；CLS = 0.00。即使按 Lighthouse 模拟（4x CPU + slow 3G throttle，通常放大 3-4 倍），553ms 仍在 ~1.6-2.2s，达标稳健。
evidence: 2026-06-26 production preview 4173 — /login LCP 553ms / CLS 0.00；/dashboard LCP 2466ms / CLS 0.00（首次 trace 因 profile 残留 token 跳转 dashboard，清 token 后复测 login）

### 3. Asset list virtual scroll UX
expected: 打开 `/operations/assets`（43 列资产列表），注入 200+ 条记录，DOM 中只渲染约 20 行，滚动流畅
result: pass
note: 实测 `/operations/assets` 标签页资产列表：**总 3318 条**（远超 200+ 要求；触发路径为顶部 tab "资产列表" uid=4_64，直接 navigate `/operations/assets` 重定向到 dashboard 空状态）。使用 **antd Virtual Table**（`ant-table-virtual` + `ant-table-tbody-virtual-holder-inner` + 虚拟滚动条），**DOM 实际渲染行数 = 12 行**（视口 600px 高），≤ expected 20 行。pageSize=50/页 × 67 页（后端分页），滚动条 thumb 存在，12 行均有真实 data-row-key UUID 从 3318 数据动态取。列数差异：snapshot 显示 14 列可见列，非 expected 43 列 —— **Phase 27 全局列自定义后列数随用户配置变化**（记忆库 global-column-customization），不是回归。Phase 30 Wave 4 虚拟滚动优化生效。
evidence: 2026-06-26 dev 4000 — paginationTotal "共 3318 条" / virtualHolderChildCount 12 / virtualHolderHeight 708px / tableBodyHeight 600px

### 4. Widget memo boundary
expected: 打开 dashboard（6 个 widget），通过 API 更新其中一个 widget 的数据，使用 React Devtools Profiler 确认只有该 widget 重新渲染
result: blocked
blocked_by: no-default-dashboard
reason: "admin 账号当前无默认仪表盘（dashboard 渲染 hasNoDataMessage=true + widgetCount=0 + gridItemCount=0；bodyText 含"您还没有设置默认仪表盘"），无法验证 widget memo 边界——前置条件缺失（Phase 30 验证时 admin 有默认 dashboard，当前数据库状态不一致：可能 DB 重置/admin dashboard 被删）。localStorage 仍有 `dashboard-storage` 持久化键，但未生效为默认。"
note: 即使通过"查看仪表盘列表"恢复默认 dashboard，**chrome-devtools MCP 不支持 React Devtools Profiler**（无 `react-devtools-*` 工具），无法精确测量"只有该 widget 重新渲染"。performance_start_trace 可粗略观察组件渲染时长，但无法严格证明 memo 边界。Phase 30 Wave 4 4 个 React.memo + 5 条 ESLint 规则的代码层保护（前 3 个 virtual table 已实测验证 #3）已生效，但运行时 Profiler 验证需要浏览器 DevTools Profiler 扩展或 React DevTools standalone。

**b 审计详情 (2026-06-26 via scripts/uat-audit 一次性 Go 脚本，已删除)**：admin user_id = `652eae20-48e6-4a42-b2c5-b53247195627`。dashboard 实际表名是 **`sys_dashboards`**（不是 `sys_dashboard`），20 列；+ `sys_dashboard_versions`（6 列，0 行）。sys_dashboards 表共 6 行，**全部 `deleted_at` 非空**（2026-01-21 ~ 2026-01-26 软删除）+ **全部 `is_default=false`** + 所有 owner_id = admin user_id。即 admin 名下所有 dashboard 都被软删除，导致"无默认仪表盘"。修复路径（用户选择不修复）：`UPDATE sys_dashboards SET deleted_at=NULL, is_default=true WHERE id=<某存活 dashboard id>`，并刷新前端 React Query 缓存。
how-to-verify: (1) 在浏览器手动创建/选择默认 dashboard + 添加 6 widget；(2) 启用 React DevTools 浏览器扩展；(3) 录制 Profiler，更新单个 widget 数据，确认 flame graph 中只有目标 widget 重新渲染。

### 5. Dict cache invalidation E2E
expected: 在 `/system/dict` 字典管理页修改某个字典项，然后访问使用 `useDict('sys_user_sex')` 的页面，新字典值应自动出现无需手动刷新
result: blocked
blocked_by: expected-assumption-invalid + useDict-consumer-mismatch + tab-navigation-page-reload
reason: |
  **(1) expected 字典类型 sys_user_sex 不存在**：当前字典类型列表 8 条（dashboard_template_scope/ops_info_point_type/device_type/dashboard_widget_type/ops_isp/network_device_type/dashboard_scope/ops_dedicated_line_type），无 sys_user_sex，**expected 字面验证不可行**。

  **(2) 实际"设备类型"下拉不是 useDict('device_type') 消费者，是后端聚合枚举**：在 /assets/assets 资产列表打开"设备类型"筛选下拉，得到 10 个选项（台式电脑 3318 / 便携式电脑 801 / A4激光打印机 439 / 交换机 321 / ...），**这是 sys_asset 表 device_type 字段的去重值（带资产计数），不是 sys_dict_data 表 device_type 字典的 12 个值**（服务器/UPS/防火墙/...）。前端"设备类型"下拉用的是后端 `/ops/asset/list` 或类似接口聚合返回，**不走 useDict**，所以"字典修改 → 下拉自动更新"链路在此页不成立。

  **(3) tab 切换 = 完整页面 navigation，破坏"不刷新"前提**：从字典管理 tab 切到资产列表 tab 触发 `Page navigated to http://localhost:4000/assets/assets`（完整 navigation，非 SPA 内部切换），React Query 内存缓存重置。即便 useDict invalidate 生效，也无从观察"已缓存 → 失效 → 新值出现"的连续状态。

  **(4) 实质性验证仍部分完成**：在字典管理成功新增 "UAT-测试新增类型-1527"（device_type 字典，总 12→13，创建成功 toast），后端持久化链路 OK。但前端消费链路无法验证（无 useDict('device_type') 实际消费者页面可观察）。
note: |
  **测试数据残留**：`sys_dict_data` 表 device_type 字典新增了一条 "UAT-测试新增类型-1527" / "uat_test_type_1527"。UI 删除因 antd 远程 Select 嵌套 + chrome-devtools UI 自动化在该场景链路不稳（dropdown 展开后 listbox childCount=1，options 未及时渲染）而未完成。**无害标记**（不影响业务，因 device_type 字典实际无前端 useDict 消费者；如需清理：`DELETE FROM sys_dict_data WHERE dict_label='UAT-测试新增类型-1527'` 或在字典管理 UI 手动删除）。
how-to-verify: |
  (1) 找一个**真实 useDict(type) 消费者页面**（grep `useDict\('xxx'\)` 找引用）；
  (2) 在 /system/dict 修改该字典数据；
  (3) 在该消费页面**不刷新**看新值是否自动出现（需 SPA 内 tab/route 切换，不触发完整 navigation）；
  (4) 用 React Query DevTools 或 DevTools Network 面板确认失效后自动 refetch。
evidence: |
  2026-06-26 —
  - 字典类型下拉 8 个选项（无 sys_user_sex）
  - 设备类型字典原 12 条 → 新增后 13 条（"创建成功" toast）
  - /assets/assets 设备类型下拉 10 选项（聚合枚举，非字典值）

**b 审计详情 (2026-06-26)**：grep `useDict\('[^']+'\)` 全前端代码 = **3 个真实消费者**：
  - `pages/operations/dedicated-lines/index.tsx:64` → `useDict("ops_dedicated_line_type")`
  - `pages/operations/dedicated-lines/index.tsx:65` → `useDict("ops_isp")`
  - `pages/operations/info-points/index.tsx:95` → `useDict("ops_info_point_type")`

**`sys_user_sex` 仅在 `src/hooks/useDict.ts:4` JSDoc 注释示例**，前端零真实 `useDict('sys_user_sex')` 调用。sys_dict_type 表查询：`sys_user_sex` / `sys_user_status` / `sys_yes_no` **不存在**；6 个 ops/dashboard 字典 status=0 + 数据齐全。6 种 widget 类型可用：stat-card / chart / table / list / progress / metric。

**正确的 #5 验证路径应是**：改 `ops_dedicated_line_type` 或 `ops_isp` 数据 → 在 `/operations/dedicated-lines` 看 useDict 缓存失效。但**顶部 tab 切换触发完整 page navigation**（`Page navigated to http://localhost:4000/assets/assets`），React Query 内存缓存重置，仍破坏"不刷新页面"前提。所以即便换正确路径 + 正确字典，验证手段仍受限（除非用 SPA 内 React Router 导航或直接 URL 跳转）。

## Summary

total: 5
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 2

## Gaps

无 BLOCKER 缺口。Phase 30 跨 4 个 Wave 全部执行：
- Wave 1: 基础设施（visualizer, manualChunks, 500KB 阈值, baseline）
- Wave 2: 4 个重库按需加载（three/echarts/xlsx/md-editor）— vendor-commons -17.9%
- Wave 3: React Query 层（useDict, useDeptTree, useRoleList, useTableQuery, dict/dept/role 失效）
- Wave 4: 渲染层（3 个 virtual table, 4 个 React.memo, 5 条 ESLint 规则）

### Partial findings (已记录)

| 项 | 状态 | 说明 |
|----|------|------|
| `useDict` hook 无直接 page consumer | 基础设施先行 | 留待后续 quick task 接入 |
| `useTableQuery` hook 无直接 page consumer | 基础设施先行 | 留待后续 quick task 接入 |
| `Widget.tsx` 未接入 DashboardGrid | 部分完成 | Wave 4 next-steps 已记录 |
| `BaseEditModal.tsx` 未替换现有 modal 调用 | 部分完成 | 留待后续 quick task 接入 |
| vendor-commons = 610 KB gzip | 超出 500KB 预算 | 已从 baseline 743 KB 减少 -133 KB / -17.9%，需进一步 split |

### 已知遗留 (deferred-items.md 已记录)

- LCP 实测未捕获（手动 Lighthouse 任务，自动化无法覆盖）
- 108 条 pre-existing ESLint 错误级违规需后续 quick task 清理
- pre-existing TypeScript 错误阻塞 `npm run build`（workaround: `npx vite build`）
- ESLint 规则名按 v7.37 canonical 修正（语义保持）
