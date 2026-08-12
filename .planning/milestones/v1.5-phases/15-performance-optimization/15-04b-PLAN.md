---
phase: 15-performance-optimization
plan: 04b
type: execute
wave: 3
depends_on:
  - 15-04a
files_modified:
  - xingran-react-frontend/src/lib/api/macHeatmapApi.ts
  - xingran-react-frontend/src/components/network/MACHeatmapChart.tsx
  - xingran-react-frontend/src/pages/network/mac/heatmap.tsx
  - xingran-react-frontend/src/router/routeConfigManager.ts
  - internal/core/db/migrations/15X_mac_heatmap_menu.sql
  - internal/core/db/migrations/migration_NNN_mac_heatmap_menu.go
autonomous: true
requirements:
  - PERF-04
must_haves:
  truths:
    - Frontend page xingran-react-frontend/src/pages/network/mac/heatmap.tsx exists and renders ECharts heatmap
    - Route /network/mac/heatmap registered, menu 端口使用热力图 with permission network:mac:heatmap
    - Mobile (< sm) degrades to Top-20 port list + color cards
    - MACTrajectoryChart.tsx is NOT modified
    - sys_menu SQL 包含菜单项 端口使用热力图 (parent_id = network 历史查询父菜单)
    - SQL migration 15X_mac_heatmap_menu.sql + migration_NNN_mac_heatmap_menu.go 在 AutoMigrate() 中注册 (符合项目规范: xingran-migrations-no-sql-autoloader)
    - npm run type-check 退出码 0
  artifacts:
    - path: xingran-react-frontend/src/components/network/MACHeatmapChart.tsx
      provides: ECharts heatmap chart component
      contains: echarts-for-react
    - path: xingran-react-frontend/src/pages/network/mac/heatmap.tsx
      provides: Standalone heatmap page
      contains: MACHeatmapChart
    - path: internal/core/db/migrations/15X_mac_heatmap_menu.sql
      provides: sys_menu INSERT for heatmap menu item with perm network:mac:heatmap
      contains: network:mac:heatmap
  key_links:
    - from: xingran-react-frontend/src/pages/network/mac/heatmap.tsx
      to: xingran-react-frontend/src/components/network/MACHeatmapChart.tsx
      via: React component import
      pattern: import MACHeatmapChart
    - from: xingran-react-frontend/src/router/routeConfigManager.ts
      to: xingran-react-frontend/src/pages/network/mac/heatmap.tsx
      via: Path translation
      pattern: 'heatmap'
    - from: internal/core/db/migrations/migration_NNN_mac_heatmap_menu.go
      to: internal/core/db/migrations/15X_mac_heatmap_menu.sql
      via: SQL embedded string or file read
      pattern: INSERT INTO sys_menu
---

<objective>
实现 PERF-04 前端部分 + 菜单/权限注册: ECharts heatmap 组件 + 独立页 /network/mac/heatmap + 移动端降级 + 前端 API 包装 + 路由路径翻译 + sys_menu 菜单项 (权限点 network:mac:heatmap) SQL migration。

Purpose: 暴露 MV-04 数据为前端可视化能力;按 D-18 锁定路由/菜单/权限命名;沿用 Phase 14 时间预设与 React Query 模式。

Output: 4 个前端文件 + 1 SQL 迁移 + 1 Go migration wrapper。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/REQUIREMENTS.md
@.planning/phases/15-performance-optimization/15-CONTEXT.md
@.planning/phases/14-frontend-ux/14-CONTEXT.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 前端 API 包装 + ECharts heatmap 组件 + 独立页 + 移动端降级</name>
  <files>xingran-react-frontend/src/lib/api/macHeatmapApi.ts, xingran-react-frontend/src/components/network/MACHeatmapChart.tsx, xingran-react-frontend/src/pages/network/mac/heatmap.tsx, xingran-react-frontend/src/router/routeConfigManager.ts</files>
  <read_first>
    - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx (颜色体系参考)
    - xingran-react-frontend/src/pages/network/mac/history.tsx (列表页 UI 模式 + RangePicker + 时间预设)
    - xingran-react-frontend/src/hooks/useTableManager.ts (列表/筛选 hook)
    - xingran-react-frontend/src/router/routeConfigManager.ts (PATH_TRANSLATIONS 表追加)
    - xingran-react-frontend/src/lib/api.ts (post/get 包装函数)
  </read_first>
  <action>
    新建 `xingran-react-frontend/src/lib/api/macHeatmapApi.ts`:

    1. import `{ post }` from `@/lib/api` + types from service 定义。
    2. 导出 `queryMACHeatmap(params: HeatmapQuery): Promise<Result<HeatmapResult>>`。
    3. 函数体: `return post('/network/history/heatmap', params)`。

    新建 `xingran-react-frontend/src/components/network/MACHeatmapChart.tsx`:

    1. 接受 props: `data: HeatmapResult` + `loading: boolean` + `isMobile: boolean`。
    2. 桌面端 (isMobile=false): 用 `echarts-for-react` 的 ReactECharts 组件, `series: [{ type: 'heatmap', data: points }]`, `xAxis: { type: 'category', data: ports }`, `yAxis: { type: 'category', data: devices }`, `visualMap: { min: 0, max: maxCount, calculable: true, inRange: { color: ['#50a3ba', '#eac736', '#d94e5d'] } }`。
    3. 移动端 (isMobile=true): 返回 Top-20 端口列表 + 颜色卡片 (`<Card>` + 背景色按 change_count 数值映射)。
    4. 组件使用 `React.memo` 包裹 (Phase 30 性能规则),按需加载 echarts: `const ReactECharts = React.lazy(() => import('echarts-for-react'))` (在桌面端分支使用)。
    5. **不要修改 MACTrajectoryChart.tsx**。

    新建 `xingran-react-frontend/src/pages/network/mac/heatmap.tsx`:

    1. 默认导出函数组件,`React.lazy` + Suspense 包裹 (Phase 30 D-08)。
    2. 用 `useTableManager` 管理时间筛选 (复用 Phase 14 D-07 时间预设:近 1h/24h/7d/30d/90d/自定义)。
    3. 用 `useQuery(['macHeatmap', params], () => queryMACHeatmap(params))` 拉数据, `placeholderData: keepPreviousData`。
    4. 移动端断点: 用 `useBreakpoint()` 或 AntD `Grid.useBreakpoint()`, `< sm` 时 `isMobile=true`。
    5. 渲染 `<MACHeatmapChart data={data} loading={isLoading} isMobile={isMobile} />`。
    6. 错误状态: 复用 Phase 14 D-20 `ErrorAlertWithRetry` 组件 (若未抽出,可用内联 Alert)。
    7. 空状态: 复用 Phase 14 D-18 `EmptyStateWithAction` 组件或内联 AntD `Empty`。

    修改 `xingran-react-frontend/src/router/routeConfigManager.ts`:

    1. 在 PATH_TRANSLATIONS 表追加: `'heatmap': '热力图'` (让菜单自动生成 热力图 中文标签)。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npm run type-check 2>&1 | head -40</automated>
  </verify>
  <done>
    - MACHeatmapChart.tsx 存在, 导出 React 组件
    - heatmap.tsx 存在, 使用 queryMACHeatmap + MACHeatmapChart
    - macHeatmapApi.ts 导出 queryMACHeatmap 函数
    - routeConfigManager.ts PATH_TRANSLATIONS 包含 heatmap 键
    - MACTrajectoryChart.tsx 未被修改 (用 git diff 确认)
    - npm run type-check 退出码 0
  </done>
</task>

<task type="auto">
  <name>Task 2: sys_menu 菜单项 + 权限点注册 (SQL + Go migration)</name>
  <files>internal/core/db/migrations/15X_mac_heatmap_menu.sql, internal/core/db/migrations/migration_NNN_mac_heatmap_menu.go</files>
  <read_first>
    - internal/core/db/migrations/*.sql (Phase 13 13-XX-menu-registration*.sql 风格参考)
    - internal/core/db/migrations/migration_*.go (迁移调用模式 + AutoMigrate 注册)
    - .planning/phases/13-query-layer-trajectory/13-CONTEXT.md (D-13 菜单注册风格)
    - .planning/phases/14-frontend-ux/14-CONTEXT.md (D-14-D-04 权限点命名约定)
  </read_first>
  <action>
    **项目规范**: xingran 项目 .sql 迁移文件不会被自动加载, 必须用 migration_NNN_*.go 函数显式调用并加入 AutoMigrate()。

    新建 `internal/core/db/migrations/15X_mac_heatmap_menu.sql`:

    1. SQL 风格沿用 Phase 13 13-XX-menu-registration*.sql (INSERT INTO sys_menu ...)。
    2. 菜单项元数据:
       - menu_name = '端口使用热力图'
       - parent_id = (SELECT id FROM sys_menu WHERE menu_name = '历史查询' OR menu_name = 'MAC地址历史' LIMIT 1) (沿用 Phase 13 父菜单)
       - path = 'heatmap'
       - component = 'network/mac/heatmap'
       - query = NULL
       - is_frame = '1'
       - is_cache = '0'
       - menu_type = 'C' (菜单类型: 菜单)
       - visible = '0' (可见)
       - status = '0' (正常, 沿用 0=正常 约定)
       - perms = 'network:mac:heatmap' (D-18 锁定权限点)
       - icon = 'heat-map' 或 'fund' (Ant Design icon, 选其一)
       - order_num = (parent 下 +1)
       - remark = '端口使用热力图 (Phase 15 PERF-04)'
    3. 使用 ON CONFLICT (menu_name) DO NOTHING 或 WHERE NOT EXISTS 防重复插入 (项目 SQL 迁移惯例)。
    4. 同时 INSERT 按钮权限 (menu_type = 'F') perms = 'network:mac:heatmap:query', menu_name = '热力图查询', parent_id = (上述菜单的 id)。

    新建 `internal/core/db/migrations/migration_NNN_mac_heatmap_menu.go` (NNN 选当前未占用的最大号 + 1, 例如 153):

    1. 包名 `migrations`。
    2. 函数名 `RegisterMacHeatmapMenuMigration(db *gorm.DB) error`。
    3. 函数体: `db.Exec(SQL_STRING)` 其中 SQL_STRING 是上面 SQL 文件的内容 (内嵌 const) 或通过 `embed.FS` 读取。
    4. 若使用 embed: `import _ "embed"` + `//go:embed 15X_mac_heatmap_menu.sql` + `var menuSQL string`。
    5. 在 `internal/core/db/database.go` 的 AutoMigrate() 调用链中注册 (参考 migration_mac_history.go 注册位置, 不删除现有注册, 追加新行)。
    6. 函数名前缀: `migration_NNN_` (符合项目命名规范)。

    注意: 项目内存 (xingran-migrations-no-sql-autoloader) 警告 .sql 文件不会被自动加载, 必须显式 Go 包装。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./... 2>&1 | head -20 && grep -l "15X_mac_heatmap_menu" internal/core/db/migrations/*.go</automated>
  </verify>
  <done>
    - 15X_mac_heatmap_menu.sql 存在, 包含 INSERT INTO sys_menu 含 perms='network:mac:heatmap'
    - migration_NNN_mac_heatmap_menu.go 存在, 调用 db.Exec(SQL)
    - database.go AutoMigrate 调用链包含 RegisterMacHeatmapMenuMigration
    - go build ./... 退出码 0
  </done>
</task>

</tasks>

<verification>
- npm run type-check 退出码 0
- go build ./... 退出码 0
- 前端 heatmap 页 /network/mac/heatmap 渲染 ECharts heatmap
- 移动端降级为 Top-20 端口列表
- MACTrajectoryChart.tsx 未被修改
- sys_menu 中存在 端口使用热力图 菜单项 + 权限点 network:mac:heatmap
</verification>

<success_criteria>
- 前端 ECharts heatmap 渲染
- 移动端降级正常
- 权限点 network:mac:heatmap 在 sys_menu 注册
- 菜单显示 端口使用热力图 (parent = 历史查询 或 MAC地址历史)
- npm run type-check 退出码 0
- go build ./... 退出码 0
</success_criteria>

<output>
Create `.planning/phases/15-performance-optimization/15-04b-SUMMARY.md` when done
</output>
