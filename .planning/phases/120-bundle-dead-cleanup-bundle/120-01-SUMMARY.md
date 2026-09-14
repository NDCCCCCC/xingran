---
phase: 120
plan: 120-01
title: BUNDLE-01/03/04 — ExcelImport 懒加载统一 + 路由 glob 排除 + EChartsWrapper 注释修正
status: complete
wave: 1
requirements_completed: [BUNDLE-01, BUNDLE-03, BUNDLE-04]
commit: 09ac29f
files_modified: 11
---

## What Shipped

### BUNDLE-01 — ExcelImport 9 个静态调用点统一
- 9 个静态 `ExcelImport` 调用点（assets/buildings/server-rooms/room-devices/info-points/floors/dedicated-lines/dept/user）全部迁移至 `ExcelImportLazy`
- 保留 Lazy 作为唯一入口（方向：迁 ExcelImportLazy，反向 Lazy 删除未采纳）
- 删除非 Lazy 直接导入，确保 entry chunk 不引入 ExcelImport 静态依赖

### BUNDLE-03 — 路由 glob phantom chunk 清零
- `componentLoader.tsx` glob 模式增加 `**/modals/**`、`**/components/**`、`**/hooks/**` 排除
- knowledge/articles/modals 等不再产出独立 chunk（phantom chunk 清零验证：grep dist/assets/*.js 无 modals 独立 chunk）

### BUNDLE-04 — EChartsWrapper 注释修正
- 修正误导性注释（不是真 lazy load 的注释误导已清）
- 保持 ECharts 单 vendor chunk（vendor-echarts）合理拆分

## Verification

- `npm run build` ✓ built in 34.58s
- entry gzip 显著下降（vs Phase 115 基线）
- 全量 gzip < 2.5MB（vendor chunks 正常拆分）
- size-limit 门禁保持绿

## Deviations

无