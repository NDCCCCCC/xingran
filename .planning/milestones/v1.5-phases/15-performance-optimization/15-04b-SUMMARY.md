---
phase: 15-performance-optimization
plan: 04b
type: execute
wave: 3
completed_at: 2026-06-15
commit: c16a58f
depends_on: 15-04a
---

# Plan 15-04b: 前端 ECharts 热力图 + 菜单/权限注册 — SUMMARY

## 执行结果

按计划完成 5 个前端文件 + 1 SQL 迁移 + 1 Go migration wrapper。

## 创建的文件

1. `xingran-react-frontend/src/lib/api/macHeatmapApi.ts` — `queryMACHeatmap(params)` post 包装
2. `xingran-react-frontend/src/components/network/MACHeatmapChart.tsx` — ECharts heatmap 组件 + 移动端 Top-20 降级
3. `xingran-react-frontend/src/pages/network/mac/heatmap.tsx` — 独立页 `/network/mac/heatmap`
4. `xingran-react-frontend/src/router/routeConfigManager.ts` — 追加 `'heatmap': '热力图'` 路径翻译
5. `internal/core/db/migrations/153_mac_heatmap_menu.sql` — sys_menu 菜单项 + 按钮权限
6. `internal/core/db/migrations/migration_153_mac_heatmap_menu.go` — 幂等执行 + inline 兜底

## 修改的文件

- `internal/core/db/database.go` — 注册 `Migrate153MacHeatmapMenu`

## 关键设计决策

- **桌面端 ECharts heatmap**: X=端口, Y=设备, 值=change_count, visualMap 色阶 `['#50a3ba', '#eac736', '#d94e5d']` (D-18)
- **移动端降级**: `< sm` 断点切 Top-20 端口列表 + AntD `Card` + `Tag` 颜色卡片
- **路径翻译**: `routeConfigManager.ts` PATH_TRANSLATIONS 追加 `heatmap` → `热力图` 让菜单自动显示中文
- **菜单父级兼容**: SQL 候选 `'历史查询'` 或 `'MAC地址历史'`,ORDER BY 优先 MAC地址历史
- **幂等保护**: 迁移函数先 Count `端口使用热力图` 存在性,SQL 内 NOT EXISTS 兜底
- **MACTrajectoryChart.tsx 未修改**: 通过 git diff 验证

## 验证

- `go build ./...` 退出码 0
- `npm run type-check` (tsc --noEmit) 退出码 0
- Commit: `c16a58f`

## 后续

- 15-05: EXPLAIN ANALYZE 抽样验证测试 + VERIFICATION.md
