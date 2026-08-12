---
phase: 14-frontend-ux
plan: 01
subsystem: network-mac
tags: [feat, frontend, mac, query, timeline, virtual-scroll, react-query]
dependency_graph:
  requires: [13-04, 14-03]
  provides: [14-02, 14-04, 14-05b]
  affects: [network-mac-frontend]
tech-stack:
  added: []
  patterns:
    - "网络 API 集中收口:网络相关 API 全在 lib/api/networkApi.ts"
    - "事件类型颜色与图标元组化(EVENT_META)与 MACTrajectoryChart 严格一致"
    - "useTableQuery + AntD Table virtual + placeholderData: keepPreviousData(D-12)"
    - "useColumnConfig 8 列可隐藏/重排(D-08)"
    - "useSearchParams 注入 URL 预填表单(D-17)"
    - "Grid.useBreakpoint() 自动切换表格/卡片(D-05)"
key-files:
  created:
    - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
    - xingran-react-frontend/src/components/network/index.ts
    - xingran-react-frontend/src/pages/network/mac/history.tsx
    - xingran-react-frontend/src/pages/network/mac/history/index.tsx
  modified:
    - xingran-react-frontend/src/lib/api/networkApi.ts
decisions:
  - "端点严格按 D-01 锁定为 POST /network/history/list(plan 显式约束,即便当前后端仅注册了 /history/port 等子端点;14-04 计划明确后续在此端点上扩展 format=xlsx 分支,故采用 plan 锁定的契约)"
  - "事件颜色与图标沿用 MACTrajectoryChart.tsx:24-29 体系(绿/黄/红/蓝)以保持 Gantt 与 Timeline 视觉一致(D-10)"
  - "时间线组件对外暴露 mac + startTime + endTime + deviceId? props,内部 React Query 缓存 60s,跨页复用(列表展开行 / 轨迹页侧栏 / 设备详情页)"
  - "事件项点击跳 /network/mac/trajectory 带查询参数,后续 14-02 在轨迹页注入 query 预填查询条件"
  - "列配置 pageKey = 'mac.history.list' 与 14-04 共用,保证用户列偏好跨工具栏新增导出按钮后不丢失"
  - "空/错误兜底先用内联 AntD Empty/Alert,加 TODO 引用 14-05 替换为 EmptyStateWithAction / ErrorAlertWithRetry"
metrics:
  duration: ~50min
  completed: 2026-06-14
  tasks: 2
  files_changed: 5
  commits: 2
---

# Phase 14 Plan 01: MAC 历史查询主列表页 + 事件时间线组件 Summary

## One-liner

实现 MAC 历史查询主列表页(`/network/mac/history`),以 8 列全列 + 6 个时间预设 + 虚拟滚动 + 跨页复用垂直时间线组件为支撑,锁定端点 `POST /network/history/list` 并为 14-02/14-04/14-05b 提供统一列表范式与时间线 API。

## What Was Built

### Task 1 — 扩展 networkApi + 新增 MACEventsTimeline 组件
- **`networkApi.ts` 新增**:`queryMACHistory(params)` / `getMACEvents(mac, startTime, endTime)` + 配套 TS 类型 `MACHistoryQueryParams` / `MACHistoryRecord` / `MACHistoryPageResult`
- **`MACEventsTimeline.tsx` 新增**:跨页复用垂直时间线组件,4 种事件类型颜色与图标与 MACTrajectoryChart.tsx:24-29 严格一致(`appeared: #52c41a / PlusCircleOutlined`、`moved: #faad14 / SwapOutlined`、`disappeared: #ff4d4f / MinusCircleOutlined`、`vlan_changed: #1890ff / TagOutlined`)
- **`components/network/index.ts` 新增**:barrel 统一导出 `MACTrajectoryChart` 与 `MACEventsTimeline` 及其类型

### Task 2 — 创建 MAC 历史查询主列表页
- **`pages/network/mac/history.tsx`**:桌面表格(虚拟滚动 + 8 列全列) + 移动卡片双视图,6 个时间预设按钮 + RangePicker 互斥同步,6 个表单筛选字段,URL 参数注入(`?deviceId=&portName=&startTime=&endTime=&mac=`),操作列"查看轨迹"/"查看事件"两个 LinkButton
- **`pages/network/mac/history/index.tsx`**:路由兼容 re-export shim

## Files Changed

| Action | Path | Lines |
|--------|------|-------|
| modified | `xingran-react-frontend/src/lib/api/networkApi.ts` | +99 |
| created | `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx` | +227 |
| created | `xingran-react-frontend/src/components/network/index.ts` | +12 |
| created | `xingran-react-frontend/src/pages/network/mac/history.tsx` | +676 |
| created | `xingran-react-frontend/src/pages/network/mac/history/index.tsx` | +2 |

## Verification

- `npx tsc --noEmit -p .` exit code 0(0 errors,0 warnings)
- 全部 8 个 Task 1 acceptance criteria 通过(grep 命中):
  - `queryMACHistory` / `getMACEvents` 两个 export 函数存在
  - `/network/history/list` 端点出现 ≥ 1 次
  - 4 个 hex 颜色全部命中
  - 4 个图标常量(PlusCircleOutlined / SwapOutlined / MinusCircleOutlined / TagOutlined)全部命中
  - `navigate('/network/mac/trajectory?mac=...')` 命中
  - `components/network/index.ts` 含 `MACEventsTimeline` export
- 全部 8 个 Task 2 acceptance criteria 通过(grep 命中):
  - `history.tsx` 与 `history/index.tsx` 文件存在
  - `useColumnConfig.pageKey = 'mac.history.list'`
  - 8 个 `key: 'time' ... 'action'` 全部命中
  - 5 个预设按钮(`近 1h` / `近 24h` / `近 7d` / `近 30d` / `近 90d`)+ `自定义` 全部命中
  - `Table virtual` 属性命中
  - `useTableQuery('mac.history.list', ...)` 命中
  - `MACEventsTimeline` 命中 ≥ 1 行
  - `useSearchParams` 命中 ≥ 1 行

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Worktree branch was 2 commits behind main**
- **Found during:** Pre-Task 1 setup(网络 API 文件不存在)
- **Issue:** Worktree `worktree-agent-a5c6c1d3e1ecbd9b8` 起始于 `2bf9b5e`(Phase 31),落后当前 main `d99318e`(Phase 14-05a)约 35 commits,导致 `networkApi.ts` / `MACTrajectoryChart.tsx` / `EmptyStateWithAction.tsx` 等 Phase 13/14-05a 资产缺失
- **Fix:** 执行 `git merge main --ff-only` 把 worktree 分支快进到 `d99318e`(在自身命名空间内,不动其他 worktree / 不动 main 指针);随后用 `mklink /J` 创建 Windows 目录联接让 `node_modules` 指向主 checkout,恢复 `npx tsc` 能力
- **Files modified:** 无文件改动(仅 .git/index 与 worktree filesystem 状态)
- **Commit:** 无单独 commit(属于 worktree 自身修复)

### Plan-exact

除上述 worktree 同步外,其余严格按照 plan 执行:
- 端点严格 `/network/history/list`(D-01 锁定)
- 颜色与图标严格沿用 MACTrajectoryChart.tsx:24-29(D-10 锁定)
- 8 列 `key` 命名与 plan 严格一致(time / mac / device / port / eventType / vlan / status / action)
- 列配置 `pageKey = 'mac.history.list'` 与 plan 严格一致

## Known Stubs (待 14-05 / 14-05b 接管)

| Stub | Location | Reason | Hand-off Plan |
|------|----------|--------|---------------|
| 内联 `<Alert type="error">` 错误展示 | history.tsx:551-562 | ErrorAlertWithRetry 完整接入由 14-05 提供 | 14-05 |
| 内联 `<Empty>` 空数据展示 | history.tsx:498-506 | EmptyStateWithAction 完整接入由 14-05 提供 | 14-05 |
| 列配置抽屉入口占位(`{false && ...}`) | history.tsx:611-614 | ColumnConfigModal 挂载由 14-05 提供 | 14-05 |
| 移动端卡片最终样式 | history.tsx:373-471 | 14-05b 统一收口 | 14-05b |

## Threat Flags

无新增安全相关攻击面 — 全部为前端纯展示/查询,无新增网络端点、auth 路径、文件访问或 schema 变更。

## Commits

| Hash | Message |
|------|---------|
| `9ec2555` | feat(14-01): add queryMACHistory/getMACEvents API + MACEventsTimeline component |
| `3cf18df` | feat(14-01): add MAC history main list page (desktop + mobile card + URL param injection) |

## Hand-off

- **14-02 轨迹可视化页 UX 增强**:可直接 import `{ MACEventsTimeline }` from `@/components/network` 嵌入右侧栏,事件项点击跳 `/network/mac/trajectory?mac=...` 的契约已就绪
- **14-04 Excel 导出与批量操作**:可在 history.tsx 工具栏右侧 `extra` 槽位追加两个互斥按钮,`networkApi.ts` 已预留 `getMACEvents` 模式可复用到 `exportMACHistory`
- **14-05 / 14-05b 移动端 + 三态打磨**:history.tsx 中已加 `TODO(14-05)` 注释,Empty/Alert 替换位置已留好,可直接替换
- **14-03 菜单注册**:`/network/mac/history` 路由对应菜单项 `mac-history` 已在 Phase 13-06 + 14-03 SQL 中注册,本 plan 只需前端就位

## Self-Check: PASSED

```
[FOUND] xingran-react-frontend/src/lib/api/networkApi.ts
[FOUND] xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
[FOUND] xingran-react-frontend/src/components/network/index.ts
[FOUND] xingran-react-frontend/src/pages/network/mac/history.tsx
[FOUND] xingran-react-frontend/src/pages/network/mac/history/index.tsx
[FOUND] 9ec2555 (Task 1 commit)
[FOUND] 3cf18df (Task 2 commit)
```
