---
type: quick
slug: merge-mac-trajectory-to-list-drawer
created: 2026-06-30
status: in-progress
---

## Goal

Delete the standalone MAC trajectory page (`/network/mac/trajectory`), embed the MAC events timeline as a Drawer on the MAC address list page so the trajectory view opens when the user clicks a MAC address cell. Consolidates low-density single-purpose page into the high-traffic list page.

## Scope

### Delete
- `xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx` (独立页面)
- `xingran-react-frontend/src/pages/network/mac/trajectory/index.tsx` (re-export)
- `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` (ECharts Gantt,仅 TrajectoryPage 使用)
- `xingran-react-frontend/src/components/network/index.ts` 中的 `MACTrajectoryChart` 导出 + 类型重导出

### Modify
- `pages/network/mac/index.tsx` (MAC 列表页):
  - 新增 `drawerOpen` + `drawerMac` state
  - 新增 Drawer(右侧抽屉, width=480),内容是 `MACEventsTimeline`
  - MAC 列改为可点击 link(Button type="link"),点击打开抽屉
  - 默认时间范围 7d(同旧 trajectory 页 PRESETS[7d])
- `components/network/MACEventsTimeline.tsx`:
  - 移除"事件项点击跳 trajectory 页"的逻辑(因为独立页已删)
  - 注释更新:不再是"跨页复用",而是"列表页抽屉内"使用
- `pages/network/mac/history/MACHistoryPage.tsx`:
  - 移除 2 处"查看轨迹"按钮(跳 trajectory 页),只保留"查看事件"(展开行内嵌 timeline)
- `constants/routes.ts`:
  - 移除 `NETWORK_MAC_TRAJECTORY` 常量
- `lib/api/networkApi.ts`:
  - 清理指向 TrajectoryPage 的注释
- `components/network/macEventMeta.ts`:
  - 移除 `pages/network/mac/trajectory/TrajectoryPage.tsx` 的引用注释

### Out of Scope (留待用户手动处理)
- DB `sys_menu` 表中 `path='network/mac/trajectory'` 的菜单项需要手动删除或重定向(此 quick 任务不动 DB,后端无菜单迁移接口)。
  - 提供清理 SQL 见 SUMMARY.md 备注。

## Verification

1. `npm run type-check` 通过
2. `npm run lint` 通过(可选)
3. 前端构建:`npm run build` 通过(可选)
4. UAT:列表页点击 MAC 列 → 右侧抽屉弹出,显示该 MAC 7d 范围内的事件时间线;从 MAC 历史页进入不会跳到 trajectory 页(/network/mac/trajectory 路由 404,因为文件已删)

## Atomic Commits

1. `feat(mac): merge trajectory page into list page drawer` — 列表页加 Drawer + MAC 列点击
2. `refactor(mac): drop standalone trajectory page + dead components` — 删 page/chart/routes 常量/历史页跳转
3. `docs(quick): 20260630 merge mac trajectory to list drawer` — quick 任务记录

## Files Changed

- `pages/network/mac/index.tsx` (modify)
- `pages/network/mac/history/MACHistoryPage.tsx` (modify)
- `components/network/MACEventsTimeline.tsx` (modify)
- `components/network/index.ts` (modify)
- `components/network/macEventMeta.ts` (modify)
- `constants/routes.ts` (modify)
- `lib/api/networkApi.ts` (modify)
- `pages/network/mac/trajectory/TrajectoryPage.tsx` (delete)
- `pages/network/mac/trajectory/index.tsx` (delete)
- `components/network/MACTrajectoryChart.tsx` (delete)