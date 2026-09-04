---
gsd_state_version: 1.0
milestone: v1.29
milestone_name: 技术债治理 (Tech Debt Governance)
status: planning
stopped_at: ""
last_updated: "2026-09-04T00:00:00.000Z"
last_activity: 2026-09-04
progress:
  total_phases: 6
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State (v1.29 — milestone workstream)

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-04) — v1.29 Current Milestone 段

**Core value:** 按 2026-09-03 综合审计报告发现的优先级，逐批治理 7 项技术债行动；后端常量/CRUD/缓存/配置备份 + 前端 API 工厂化 + Phase 88 收口，使代码质量基线从此不可无声倒退。

**Current focus:** Phase 89 (常量集中化) 规划中 — `pkg/constants/pagination.go` + `pkg/constants/timeouts.go` 双基础包

## Current Position

Phase: 89 (待规划)
Plan: —
Status: Defining requirements
Last activity: 2026-09-04 — Milestone v1.29 started (技术债治理)

## Accumulated Context (carried from v1.28)

### Decisions to preserve
- D-26-01..05 (v1.26 后端覆盖率 gate): 4 层 CI 防倒退（加权阈值 + per-dir floor + ratchet + PR diff ≥80%）继续生效
- D-27-01..04 (v1.27 后端覆盖率优秀 II): 测试基建 (miniredis/httpmock/ScrapliWrapper/LDAPClientIface/TestHelperProcess/AST守护) 沿用
- D-28-01..04 (v1.28 前端覆盖率): 阶段性收口 45.13%，不再 push 到 70% 目标
- operlog regression_test.go: 11 强制敏感关键词、25 OperType 常量全程保持绿
- status_constants_test.go: 状态 0/1 命名常量全程 AST 锁值

### Blockers (active)
- 无新增 blocker（v1.29 主要是技术债治理，业务风险低）

### Pending Todos (carry forward, not in v1.29 scope)
- `.planning/todos/pending/operlog-exclude-paths.md` — operlog 白名单配置驱动（RPA heartbeat 日志污染），独立 deferred 到后续 milestone（不在 v1.29 7 项治理范围内）

### Audit Baseline (2026-09-03)
| 维度 | 基线 | 数据 |
|------|------|------|
| 后端覆盖率 | 78.12% | v1.27 SHIPPED |
| 前端覆盖率 | 45.13% | v1.28 SHIPPED (45.87% at batch47, final = 阶段性收口) |
| 测试通过 | 1688/1688 | 45/45 dirs Gate PASS |
| 真实 TODO | 18 项 | 后端 13 + 前端 7 |
| 中高度硬编码 | ~20 处 | 分页 12+ / 超时 6 / URL/端口 2 |
| CRUD 重复 | 8 services | 60-70% 重复；base.Repository[T] 已存在未用 |
| 缓存层重复 | 3 路径 | legacy root + system/ + operations/ 同一模式 |
| 配置文件 TODO | 3 函数 | config_backup_service.go:158/206/543 |
| 前端 API 文件 | ~15 | 散落在 src/lib/ |

### Phase 88 收口状态
- Phase 88 已完成 18+ batches (R24-R41, R42-R47)
- GLOBAL: 45.13% → 45.87% (batch47 末态)
- 1688 tests passing, Gate 45/45 dirs PASS
- Blocker: 边效益递减，每批仅 ~0.27pp，距 70% 目标差 24.13pp
- Decision: 接受 45.13% 阶段性收口（用户 2026-09-04 确认）

## Next Step

`/gsd:plan-phase 89` (待执行) — Phase 89: 常量集中化 (pkg/constants/pagination.go + timeouts.go)

或 `/gsd:discuss-phase 89` 先讨论实施方案