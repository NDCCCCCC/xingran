---
gsd_state_version: 1.0
milestone: v1.33
milestone_name: 前端性能治理 (Frontend Performance Remediation)
status: executing
stopped_at: Completed 114-02-PLAN.md
last_updated: "2026-09-11T18:20:22.620Z"
last_activity: 2026-09-11
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 3
  completed_plans: 2
  percent: 0
---

# Project State (v1.33 — Frontend Performance Remediation)

## Project Reference

See: `.planning/PROJECT.md` — v1.33 Current Milestone 段

**Core value:** 端到端运维可观测与可审计
**Current focus:** Phase 114 — map3d-clustering（地图聚类 O(n²) 消除）

## Current Position

Phase: 114 (map3d-clustering（地图聚类 O(n²) 消除）) — EXECUTING
Plan: 3 of 3
Status: Ready to execute
Last activity: 2026-09-11

Progress: [░░░░░░░░░░] 0%

## v1.33 范围摘要

**Goal**: 修复 2026-09-11 前端全量性能审计（`.planning/reviews/20260911-frontend-perf-audit.md`）全部 33 项 findings（2 HIGH + 8 MEDIUM + 3 次级 + 12 LOW）+ 8 项死代码/依赖清理。用户确认全量范围不分批 defer。

**修复域**: H-2 地图聚类 O(n²) / H-1 dashboard N² 级联 / selector 收尾 26 处 / BUNDLE（ExcelImportLazy 推广 + iconUtils 假动态导入）/ RENDER（columns 工厂 + Table virtual）/ DATA（VDI 去重 + 菜单门控）/ 正确性 ×3 / 死代码与僵尸依赖 ×8

**前序背景**: quick task `260911-m76`（PR #19，同日合并）修了 6 批次相邻问题；本审计为 merge 后全面扫描，33+8 项经抽查确认全部残留。

## Milestone Reference

- Roadmap: `.planning/ROADMAP.md`（2026-09-12 生成：Phases 114-120）
- Requirements: `.planning/REQUIREMENTS.md` v1.33 段（38 项，进度追踪表已填充）
- Audit input: `.planning/reviews/20260911-frontend-perf-audit.md`

## Accumulated Context (carried forward from v1.32)

### Decisions preserved

- 七 gate 基线: go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage
- operlog 25 OperType 常量 + 11 敏感关键词 AST 锁值不回归
- status 0/1 普适规则（Menu visible 例外）
- Phase 92 `base.CacheProvider` / `base.GetOrSetJSON[T]` single authority
- `src/lib/apiFactory.ts` + apiFactory.invariants.test.ts dual-guard
- captcha-background 1=启用语义锁定；agent 裸 c.JSON 有意设计；operlog exclude_paths 挂账

### Blockers

- 无

## Code State

- origin/main HEAD: `727e370` (PR #19 quick/260911-m76-frontend-perf-audit merge)
- Branch Protection: direct push to main allowed (CI gates active)，PR reviews disabled（单人项目）

## Session Continuity

Last session: 2026-09-11T18:20:22.614Z
Stopped at: Completed 114-02-PLAN.md

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 114 P01 | 11min | 2 tasks | 2 files |
| Phase 114 P02 | 12min | 2 tasks | 2 files |

## Decisions

- [Phase ?]: 114-01: 聚类共享函数独立 cluster.ts 落位（桶=threshold、3×3 邻域、下标升序、保留 sqrt 严格小于），与旧 O(n²) 逐位一致由参考实现对照测试锁定
- [Phase ?]: 114-01: cluster.ts 零 SDK 依赖（仅 BuildingItem 类型 + utils 两函数），像素投影剥离到调用方单遍预计算 Map<id,pixel>
- [Phase 114]: 114-02: 两组件聚类改单遍预计算 Map<id,pixel> + clusterBuildings 消费，n² 地图 API 调用归零；有坐标过滤保留 effect 内保层级+坐标输入集语义
- [Phase 114]: 114-02: HubeiMap 渲染体 5 道 filter 收敛 useMemo [buildings]，两组件 zoomend/tiltend 提升 effect 作用域具名 handler 并补同引用 cleanup
