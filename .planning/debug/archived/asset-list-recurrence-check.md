---
slug: asset-list-recurrence-check
status: resolved
trigger: "资产列表页面只显示一列，多选框宽度太大的问题已经出现过一次，修复过一次，请查看历史"
created: 2026-06-15T17:45:00.000Z
updated: 2026-06-15T18:00:00.000Z
---

## Root Cause

**问题从未真正修复过 — 三层独立缺陷同时存在：**

1. **defaultAssetColumns 缺少 9 个 key** — `xingran-react-frontend/src/pages/operations/assets/index.tsx:60-117` 的 `defaultAssetColumns` 数组只有 43 项（line 59 注释明确写着"默认列配置（43 列）"），但 `columns` 数组（line 300-474）有 53 个数据列。`tableColumns`（line 477-492）通过 `visibleColumns.map(colConfig => allColumnsMap.get(colConfig.key))` 过滤，因此 `visibleColumns`（来自 `defaultAssetColumns`）永远只包含 43 个 key 中的子集，9 个 cd62637/历史 fix 涉及的 key（signOrgnoName / nowUserName / nowUserDeptCode / status / nbfStatus / deviceUserName / drawingDate / machineUptime / lastInventoryDate）即使 visible=true 也无从过滤。
2. **历史 fix 未提交到 git** — `.planning/debug/asset-list-only-serial-column.md` 记录的 2026-06-13 修复（补全 9 个 key + 加 hook 防御）从未出现在 main 分支的 git log 中（`git log -- xingran-react-frontend/src/pages/operations/assets/index.tsx` 最新一次对该文件的改动是 `7f13c01 feat(30-04)` 不涉及 defaultAssetColumns）。debug session 标记为 resolved 但代码未落地，bug 实际一直存在。
3. **前端 / 后端 URL 路径不匹配** — 前端 `columnConfigApi.ts:29` 调用 `/system/settings/${pageKey}`，但后端 `internal/api/router.go:131-134` 实际挂载的是 `/system/column-config/:page_key`，导致 404。后端从未实现 `/system/settings/:pageKey` 端点（`settings_router.go` 只有 `/preferences` 和 `/config/theme/*`）。
4. **多选框宽度异常大** — 由于 `scroll={{ x: 4200 }}`（line 631）但只有 1 列数据可见，table 自动扩展多选框列填充水平空间。

附带独立问题：
- AD OU 页面使用 antd 6 已弃用 API（`pages/ad-domain/ous/index.tsx:455,498` 用 `bodyStyle`，`pages/ad-domain/ous/index.tsx:486,534` 用 `Space direction="vertical"`），属于另一独立模块的兼容性警告，不影响资产列表。
- `POST /api/v1/ad-domain/groups/sync-status 500` 与本次问题无关。

## Evidence

- `xingran-react-frontend/src/pages/operations/assets/index.tsx:59` 注释 `// 默认列配置（43 列）`
- `xingran-react-frontend/src/pages/operations/assets/index.tsx:60-117` `defaultAssetColumns` 数组共 43 项，缺少 9 个 key
- `xingran-react-frontend/src/pages/operations/assets/index.tsx:336-413` `columns` 数组包含缺失的 9 个 key（`signOrgnoName`, `nowUserName`, `nowUserDeptCode`, `status`, `nbfStatus`, `deviceUserName`, `drawingDate`, `machineUptime`, `lastInventoryDate`）
- `xingran-react-frontend/src/pages/operations/assets/index.tsx:477-492` `tableColumns` 通过 `visibleColumns.map(colConfig => allColumnsMap.get(colConfig.key)).filter(...)` 过滤
- `xingran-react-frontend/src/hooks/useColumnConfig.ts:99-136` `loadConfig` 无"最少可见列"防御，catch 块直接 `setConfig(defaultColumns)` 但 localStorage 缓存优先于 API
- `git log -- xingran-react-frontend/src/pages/operations/assets/index.tsx` 最近一次相关提交是 `6e36853 fix(27-01): 扩展资产列表列定义以支持完整43列`（仅扩展 columns 数组，未改 defaultAssetColumns）
- `git log -- xingran-react-frontend/src/hooks/useColumnConfig.ts` 最近一次提交 `cd80efe` 不涉及防御逻辑
- `xingran-react-frontend/src/lib/columnConfigApi.ts:28-29` 前端调用 `get<UserColumnConfig[]>('/system/settings/${pageKey}')`
- `internal/api/router.go:131-134` 后端挂载 `authorized.Group("/column-config").Use(SetupColumnConfigRouter)` — 真实路径 `/system/column-config/:page_key`
- `internal/api/v1/system/column_config_router.go:13-15` 定义 `GET /:page_key`, `POST "", DELETE /:page_key`
- `internal/api/v1/system/settings_router.go:42-44` 实际只有 `/preferences` GET/PUT + `/config/theme/*`
- `xingran-react-frontend/src/pages/operations/assets/index.tsx:631-635` `scroll={{ x: 4200 }}` + `rowSelection` 导致 1 列可见时多选框列扩展
- `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx:455,498` `bodyStyle={{...}}` antd 6 弃用
- `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx:486,534` `<Space direction="vertical">` antd 6 弃用

## Affected Files

- `xingran-react-frontend/src/pages/operations/assets/index.tsx:60-117` — `defaultAssetColumns` 缺 9 个 key
- `xingran-react-frontend/src/hooks/useColumnConfig.ts:99-136` — `loadConfig` 缺"最少可见列"防御
- `xingran-react-frontend/src/lib/columnConfigApi.ts:28-37` — 错误的前端路径 `/system/settings/...`
- `internal/api/router.go:131-134` — 后端路由实际在 `/system/column-config/`，与前端不匹配

## Fix Recommendation

**最小修复方案（3 处改动）：**

1. **`xingran-react-frontend/src/pages/operations/assets/index.tsx:60-117`** 在 `defaultAssetColumns` 数组末尾追加 9 项：
   ```typescript
   { key: 'signOrgnoName', label: '归属机构', visible: true, order: 44, width: 120, group: '部门与用户' },
   { key: 'nowUserName', label: '责任人', visible: true, order: 45, width: 100, group: '部门与用户' },
   { key: 'nowUserDeptCode', label: '部门编码', visible: true, order: 46, width: 120, group: '部门与用户' },
   { key: 'status', label: '状态', visible: true, order: 47, width: 80, group: '设备状态' },
   { key: 'nbfStatus', label: '拟报废', visible: true, order: 48, width: 90, group: '设备状态' },
   { key: 'deviceUserName', label: '领取人', visible: true, order: 49, width: 100, group: '部门与用户' },
   { key: 'drawingDate', label: '接收日期', visible: true, order: 50, width: 140, group: '采购信息' },
   { key: 'machineUptime', label: '最后上线', visible: true, order: 51, width: 160, group: '采购信息' },
   { key: 'lastInventoryDate', label: '盘点日期', visible: true, order: 52, width: 140, group: '采购信息' },
   ```
   同时更新 line 59 注释为 "默认列配置（52 列）"。

2. **`xingran-react-frontend/src/hooks/useColumnConfig.ts:99-136`** `loadConfig` 增加防御：缓存/API 返回后，若 `visibleColumns.length < Math.floor(defaultVisibleCount / 2)` 则忽略并使用 `defaultColumns`：
   ```typescript
   const defaultVisibleCount = defaultColumns.filter(c => c.visible).length;
   const minThreshold = Math.floor(defaultVisibleCount / 2);
   const isValidConfig = (cfg: ColumnConfig[]) => 
     cfg.filter(c => c.visible).length >= minThreshold;
   // 在每个 setConfig 之前调用 isValidConfig 校验
   ```

3. **`xingran-react-frontend/src/lib/columnConfigApi.ts:28-37`** 修正 URL 路径：
   ```typescript
   getByPageKey: (pageKey: string) => get(`/system/column-config/${pageKey}`),
   save: (data: ColumnConfigData) => post('/system/column-config', data),
   reset: (pageKey: string) => del(`/system/column-config/${pageKey}`),
   ```

**附带建议（独立于本次问题）：**
- AD OU 页面 antd 6 弃用警告（bodyStyle → styles.body，Space direction → orientation）作为单独 bug 跟踪
- ad-domain groups/sync-status 500 错误作为单独 bug 跟踪

**验证：**
- `npx tsc --noEmit` 通过
- 清空 localStorage `column_config:asset.list` 后刷新资产列表，确认默认显示 ≥ 30 列
- 浏览器 Network 面板确认 `GET /system/column-config/asset.list` 返回 200

## Specialist Hint

`react` — 主要修改集中在 React 组件（useColumnConfig hook 是 React hook + Zustand 风格的 React state 管理；资产列表 index.tsx 是 React FC；URL 修复涉及 axios 调用路径）。
