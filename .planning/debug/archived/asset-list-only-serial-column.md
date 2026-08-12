---
gsd_state_version: 1.0
slug: asset-list-only-serial-column
status: resolved
trigger: 资产列表页面只有一列 序列号，请修复
created: 2026-06-13T08:00:00.000Z
updated: 2026-06-13T09:00:00.000Z
---

# Debug Session: 资产列表页面只有序列号一列

## Symptoms

**用户报告:** 资产列表页面只有一个表头 "序列号"，下面是序列号数据。

**收集到的症状:**

1. **实际行为**: 只有"序列号"一个表头，下面是序列号数据（页面上只有一个表格列，除了复选框/操作列）
2. **时间线**: 之前正常，刚发现坏了 — 之前页面包含至少 5+ 列（部门编码、归属机构、接收日期、最后上线、盘点日期等，由 Quick Task 260609-columns 在 2026-06-09 添加，commit cd62637）
3. **错误信息**: 没有报错 — 控制台干净，无 500/404
4. **复现路径**: 通过菜单进入（资产管理 → 资产列表）
5. **数据**: 数据能正常渲染（不是后端 API 完全失败）

## Context

**相关历史:**
- Quick Task `260609-columns` (2026-06-09, commit cd62637): 添加资产列表 5 个显示字段：部门编码、归属机构、接收日期、最后上线、盘点日期
- Quick Task `260609-009` (2026-06-09, commit e3983df): 资产列表页面添加类型筛选组件
- Phase 26: 资产管理模块
- Phase 27: 全局列自定义显示功能

## Current Focus

- hypothesis: 两个独立缺陷叠加：
  1. commit cd62637 在 `columns` 数组中添加了 5 个新列（`signOrgnoName` / `nowUserDeptCode` / `drawingDate` / `machineUptime` / `lastInventoryDate`）以及原有的 `nowUserName` / `status` / `nbfStatus` / `deviceUserName` 等 key，但**忘记同步添加到 `defaultAssetColumns`**。`tableColumns` 通过 `visibleColumns`（只包含 `defaultAssetColumns` 的 keys）过滤，因此这 9 个 key 永远不会被勾选 / 显示 / 保存。
  2. `useColumnConfig` 加载阶段没有"最少可见列"防御：如果用户曾通过"列设置"弹窗手动取消所有勾选只保留"序列号"并保存到后端 / localStorage，`visibleColumns` 会只剩 1 项 `sequenceNo`，表格就只显示"序列号"一列。
- test: 验证 `defaultAssetColumns` 包含所有 53 个 columns key，且 `useColumnConfig` 加载时若可见列少于默认一半则回退到默认。
- expecting: 修复后页面默认显示 5+ 列（含 cd62637 新增的 5 列），即使曾被误配置为单列也会自动恢复。
- next_action: 验证修复已完成 — type-check 通过，build 仅剩与本次修改无关的预先存在的错误。
- reasoning_checkpoint: |
  hypothesis: "cd62637 把 5 个新列（部门编码/归属机构/接收日期/最后上线/盘点日期）写入 `columns` 数组，但未写入 `defaultAssetColumns`；`tableColumns` 由 `visibleColumns` 过滤而 `visibleColumns` 来自 `defaultAssetColumns`，所以这 5 列（以及其他未同步的 key）永远不会被勾选/显示。同时 `useColumnConfig` 加载阶段无防御，错误保存的单列配置会持久化生效。"
  confirming_evidence:
    - "columns 数组中 cd62637 新增的 5 个 key (`signOrgnoName`, `nowUserDeptCode`, `drawingDate`, `machineUptime`, `lastInventoryDate`) 在 defaultAssetColumns 中确实缺失"
    - "line 424 注释 `// 额外的列定义（对应 defaultAssetColumns 中缺失的列）` 明确说明开发者当时已意识到该缺口"
    - "defaultAssetColumns 仅 43 项，columns 数组共 53 项（45 数据列 + 操作列 + 复选框）"
    - "useColumnConfig.loadConfig 在缓存/服务端返回的 config 异常时直接 `setConfig(...)` 不做健全性检查"
  falsification_test: "若 defaultAssetColumns 已包含全部 columns key，但页面仍只显示 1 列，则假设 1 不成立，根因在 hook；若 hook 防御也修了但仍 1 列，则根因在后端/缓存持久化状态，需要清空 localStorage"
  fix_rationale: "根因有两层：(a) cd62637 同步缺失 → 补全 9 个 key 到 defaultAssetColumns 并设 visible:true (包含 cd62637 的 5 列 + 配套的 nowUserName/status/nbfStatus/deviceUserName) (b) hook 缺乏防御 → 增加 `可见列数 < floor(defaultVisible/2)` 时的回退逻辑，自动从坏配置恢复"
  blind_spots: "未运行时验证（需后端服务运行）；未验证列设置弹窗中拖拽排序是否仍然正确（新增项后序号 +9）；未验证保存按钮是否正确发送新增 key 到后端"

## Evidence

- [2026-06-13T08:10:00.000Z] 定位到资产列表页面文件 `xingran-react-frontend/src/pages/operations/assets/index.tsx` (line 1)
- [2026-06-13T08:11:00.000Z] 文件共 666 行（修复前），使用 Ant Design `Table` 组件
- [2026-06-13T08:12:00.000Z] `columns` 数组（line 300-474）共 53 项：16 显式 + 36 扩展 + 1 action
- [2026-06-13T08:13:00.000Z] `defaultAssetColumns`（line 60-117）仅 43 项，缺少 9 个 key：
  - signOrgnoName（cd62637 归属机构）
  - nowUserName（责任人）
  - nowUserDeptCode（cd62637 部门编码）
  - status（状态）
  - nbfStatus（拟报废）
  - deviceUserName（领取人）
  - drawingDate（cd62637 接收日期）
  - machineUptime（cd62637 最后上线）
  - lastInventoryDate（cd62637 盘点日期）
- [2026-06-13T08:14:00.000Z] `tableColumns`（line 477-492）通过 `visibleColumns.map(colConfig => allColumnsMap.get(colConfig.key))` 过滤，缺失的 9 个 key 永远不会被包含
- [2026-06-13T08:15:00.000Z] line 424 注释 `// 额外的列定义（对应 defaultAssetColumns 中缺失的列）` 是开发者当时已意识到缺口但未补全的物证
- [2026-06-13T08:16:00.000Z] commit cd62637 diff 显示只改了 columns 数组 22 行增 / 2 行删，未触及 defaultAssetColumns
- [2026-06-13T08:17:00.000Z] `useColumnConfig.loadConfig` (hook line 100-136) 缺乏"最少可见列"防御，坏配置会原样生效
- [2026-06-13T08:18:00.000Z] git log 显示 useColumnConfig 经历过多次重构（commit 2644679, cd80efe, 1d1e528）
- [2026-06-13T08:30:00.000Z] 修复后 `defaultAssetColumns` 共 52 项，columns 数组中所有 52 个数据列 key 均已同步
- [2026-06-13T08:31:00.000Z] 修复后 `useColumnConfig` 在加载阶段对缓存和服务端返回的 config 都加了"可见列 < floor(defaultVisible/2) 即回退" 防御
- [2026-06-13T08:32:00.000Z] `npx tsc --noEmit` 在完整项目上 EXIT=0 — 全部 19 个 build 错误均为本次修改前已存在的预先错误（涉及 EChartsWrapper、MACTrajectoryChart、WorkstationDeviceTable、VDIRow、BuildingScene、networkApi、vdiApi、VirtualMachine*、types/index.ts 等），与本次修改无关

## Eliminated

- [2026-06-13T08:00:00.000Z] hypothesis: 后端 API 完全失败 → 控制台无报错且数据能渲染（序列号可见），API 工作正常
- [2026-06-13T08:20:00.000Z] hypothesis: columns 硬编码数组被删空 → 文件 line 300-474 完整保留 53 项，columns 数组非空
- [2026-06-13T08:21:00.000Z] hypothesis: Ant Design Table 组件 bug → 项目中其他表格（建筑/楼层/工位）正常渲染
- [2026-06-13T08:22:00.000Z] hypothesis: useTableManager hook 错误裁剪了 columns → useTableManager 只管理 data/pagination/selection，不涉及 columns 过滤
- [2026-06-13T08:35:00.000Z] hypothesis: 仅 hook 防御即可解决 → 仅修 hook 不能让 cd62637 新增的 5 列默认显示（defaultAssetColumns 仍缺这 5 个 key，visibleColumns 永远不会包含它们），需要双层修复

## Resolution

- root_cause: 双层缺陷：
  1. **数据缺失**：`defaultAssetColumns` 缺少 9 个 key（其中 5 个是 commit cd62637 新增的"部门编码/归属机构/接收日期/最后上线/盘点日期"），导致 `tableColumns` 过滤时这 9 列永远不会被勾选/显示/保存。
  2. **缺乏防御**：`useColumnConfig.loadConfig` 没有"最少可见列"健全性检查 — 如果用户曾手动取消所有列只保留"序列号"并保存到后端，页面就会只显示 1 列且无法自动恢复（必须用户手动点击"重置"）。
- fix:
  1. 在 `xingran-react-frontend/src/pages/operations/assets/index.tsx` 的 `defaultAssetColumns` 数组末尾补全 9 项：signOrgnoName / nowUserName / nowUserDeptCode / status / nbfStatus / deviceUserName / drawingDate / machineUptime / lastInventoryDate，全部 `visible: true`。
  2. 在 `xingran-react-frontend/src/hooks/useColumnConfig.ts` 的 `loadConfig` 中，对 localStorage 缓存和服务端返回的 config 都加上"可见列 < floor(defaultVisible/2) 时回退到默认配置"的防御。
- verification:
  - `npx tsc --noEmit` EXIT=0（完整项目类型检查通过）
  - `npm run build` 仅剩 19 个与本次修改无关的预先存在的 TypeScript 错误（在 stash 验证前后数量相同）
  - 修改后 `defaultAssetColumns` 共 52 项，与 `columns` 数组的 52 个数据列 key 集合完全一致
- files_changed:
  - xingran-react-frontend/src/pages/operations/assets/index.tsx
  - xingran-react-frontend/src/hooks/useColumnConfig.ts
