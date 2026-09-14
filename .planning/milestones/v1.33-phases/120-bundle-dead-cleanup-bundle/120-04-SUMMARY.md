---
phase: 120
plan: 120-04
title: DEAD-02 — 4 僵尸依赖移除
status: complete
wave: 4
requirements_completed: [DEAD-02]
commit: f1e4596
files_modified: 1
---

## What Shipped

### DEAD-02 — 4 僵尸依赖移除
- **`cron-parser`** — 全库零 import 验证后从 package.json deps 移除（依赖 Phase 115 之前的 cron 重构已删除使用方）
- **`@react-spring/three`** — Three.js 弹簧动画库，BuildingMarkers/CityMarkers 删除后无消费者（Phase 114 MAP3D-06 先行）
- **`maath`** — Three.js 数学工具，3D 重建（Phase 115 SELECTOR-04）后已无引用
- **`@uiw/react-baidu-map`** — BuildingMarkers/CityMarkers 删除后无消费者

## Verification

- `grep -rn "cron-parser|@react-spring/three|maath|@uiw/react-baidu-map" package.json` → 0 结果
- `npm install` 后 node_modules 体积下降（4 依赖 + 传递依赖）
- `npm run build` ✓ 34.58s（success，无构建错误）
- `npm run type-check` ✓ exit 0
- `npm run lint` 0 errors
- size-limit 门禁保持绿（entry gzip 不推高）

## Deviations

无