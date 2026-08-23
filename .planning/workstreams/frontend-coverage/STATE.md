---
gsd_state_version: 1.0
milestone: v1.28
milestone_name: milestone
status: executing
stopped_at: Completed 82-03-PLAN.md
last_updated: "2026-08-23T08:44:28.260Z"
last_activity: 2026-08-23
progress:
  total_phases: 7
  completed_phases: 0
  total_plans: 5
  completed_plans: 3
  percent: 0
---

# Project State (v1.28 — frontend-coverage workstream)

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-23) — v1.28 Current Milestone 段

**Core value:** 前端全量口径测试覆盖率达到优秀（≥70%），CI 防倒退 gate 对称后端 v1.26 治理，使前端覆盖率从此不可无声倒退
**Current focus:** Phase 82 — coverage-caliber-and-governance

## Current Position

Phase: 82 (coverage-caliber-and-governance) — EXECUTING
Plan: 4 of 5
Status: Ready to execute
Last activity: 2026-08-23

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 0（首个 plan 执行后开始记录）

## Accumulated Context

### Decisions

v1.28 init 锁定决策（详见 PROJECT.md v1.28 段，不可违反）:

- D-01 目标线: 语句 ≥70%（全量口径，statements 3.67% → ≥70%）
- D-02 范围: 全 src + 白名单 ≤5%（候选 cad-editor 804 + cad-elements 224 ≈ 4.5%，终版 Phase 82 登记）
- D-03 CI gate: 4 层对称后端 v1.26（全局阈值 + per-dir floor + baseline ratchet + PR diff ≥80%）
- D-04 Phase 编号: 82 起（**75-81 属并行 milestone workstream v1.27，严禁使用**）

关键对标: 后端 v1.26 模式（`.github/scripts/check-coverage.sh` / `.planning/coverage-baseline.md` ratchet tracking / 74-10 diff coverage bash+awk 自实现）

- [Phase 82]: 82-01: QUAL-02 仅交付配置侧（exclude 真相源 D-10），登记文档侧由 82-04 落地后才整体 complete
- [Phase 82]: 82-01: 两步口径断言零偏差——584/571/22602/21574/830/3.85 全部实测命中研究基点
- [Phase 82]: 82-02: pages/login floor 61.6 为真相源（62.11−0.5 一位小数），RESEARCH 表 61.7 属研究期舍入笔误（Open Question 3：gate 可复算输出优先）
- [Phase 82]: 82-02: --init GLOBAL 向下截断而非四舍五入——实测落在 (3.7,3.8) 区间时舍入会生成高于实测的阈值让 gate 自锁，82-05 CI 校准直接受益
- [Phase 82]: 82-03: diff gate merge-base 回退——git 三点 diff 对无共同祖先基线 fatal 'no merge base'（plan 空树合成基线技术假设盲点），merge-base 为空时回退两树 diff，只放宽不收紧，CI PR 主路径不变
- [Phase 82]: 82-03: D-15 span 口径 DIFF 参照基线 7376/77481/9.52%（空树全量 join）——与 statements 3.85%/istanbul-Lines 3.88% 三口径对账，9.52% 由 span 包含 + 空行/注释分布不均两机制解释，非 join bug；82-04/82-05 对照用

### Pending Todos

None yet.

### Blockers/Concerns

- 并行 workstream 纪律: 本 workstream 可写产物仅在 `.planning/workstreams/frontend-coverage/` 下；PROJECT.md / MILESTONES.md / config.json 只读
- `.github/workflows/ci.yml` 为两 workstream 共享编辑区——Phase 82 加前端 gate 步骤时严禁触碰后端 job（v1.27 同时在改后端覆盖率），建议只增不改
- roadmap 口径备注: 全局 ≥70% 由"白名单外所有目录 per-dir ≥70%"数学保证（加权平均）；design-system 194 stmts 已归入 Phase 84 / COMP-05 零散桶，不留无主面积

## Session Continuity

Last session: 2026-08-23T08:44:20.491Z
Stopped at: Completed 82-03-PLAN.md
Resume file: None

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase 82 P01 | 6min | 2 tasks | 2 files |
| Phase 82 P02 | 12min | 2 tasks | 2 files |
| Phase 82 P3 | 11min | 2 tasks | 1 files |
