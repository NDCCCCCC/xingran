---
phase: 120
plan: 120-02
title: BUNDLE-02 — iconUtils 假动态导入删除 + 回归测试
status: complete
wave: 2
requirements_completed: [BUNDLE-02]
commit: 7d970eb
files_modified: 2
---

## What Shipped

### BUNDLE-02 — iconUtils 假动态导入删除
- `iconUtils.tsx:546-550` 假动态导入（bare specifier Vite 不可分析、运行时必失败被 catch 静默吞掉）删除
- 所有图标走 `STATIC_ICON_REGISTRY` + `fullIconNameMap` 静态注册表
- **行为变更**：菜单图标渲染现在 100% 可用（修复原"动态导入失败 → 图标消失"的功能 bug）
- **回归测试**：锁定图标注册表加载完整性 + 已知菜单项图标存在性

## Verification

- 静态注册表 100% 可分析（Vite 可正确打包）
- 菜单图标渲染正常（手动验证 + 回归测试断言）
- bundle 不引入运行时失败路径
- npm run lint / type-check 绿

## Deviations

无 — 按计划执行