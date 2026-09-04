---
gsd_state_version: 1.0
milestone: v1.29
milestone_name: 技术债治理
status: Phase 89 + 90 SHIPPED，Phase 91-95 待推进
last_updated: "2026-09-04T07:50:35.298Z"
last_activity: 2026-09-04 — Phase 90 SHIPPED (TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化，4 plans，零行为变更)
progress:
  total_phases: 7
  completed_phases: 2
  total_plans: 7
  completed_plans: 7
  percent: 29
---

# Project State (v1.29 — milestone workstream)

> 本文件自 2026-09-04 起追踪 v1.29。v1.27 workstream 状态历史见 `.planning/milestones/v1.27-ROADMAP.md`。

## Project Reference

Config: "mode": "yolo"

**Core value:** 按 2026-09-03 综合审计报告发现的优先级，逐批治理 7 项技术债行动；后端常量/CRUD/缓存/配置备份 + 前端 API 工厂化 + Phase 88 收口，使代码质量基线从此不可无声倒退。

**Current focus:** Phase 91 (CRUD 复用 base.Repository[T]) 待 discuss/plan — 下一个 P1 行动

## Current Position

Phase: 91 (待 discuss)
Plan: —
Status: Phase 89 + 90 SHIPPED，Phase 91-95 待推进
Last activity: 2026-09-04 — Phase 90 SHIPPED (TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化，4 plans，零行为变更)
Resume file: .planning/workstreams/milestone/phases/91-crud-base-repository-t-p1/91-CONTEXT.md
Next action: `/gsd:discuss-phase 91`

## Completed Phases (v1.29)

### Phase 89: PAGINATION 常量集中化 — SHIPPED 2026-09-04

- 3 plans (89-01/02/03)，commits 238283c..3559626
- `pkg/constants/pagination.go` 3 常量 + `pkg/query.NormalizePagination` 纯函数唯一入口
- 8 文件 12+ 处硬编码迁移；6 处业务行为变更用户已接受
- CLAUDE.md 新增 Pagination Constants Convention 段
- 决策 D-01..D-19 见 89-CONTEXT.md

### Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 — SHIPPED 2026-09-04

- 4 plans (90-01/02/03/04)，commits b51f44c..3a2efe5
- 4 个 leaf const pkg（timeouts 6 Duration + ports + protocol + concurrency）共 10 常量
- 8 调用点迁移（network handlers + ad_ldap + ws_notice + scheduler/cron 扩展审计 D-07）
- AST 锁值 10 tests；零业务行为变更（D-09）
- CLAUDE.md 新增 Timeout/Port/Protocol Constants Convention 段
- 决策 D-01..D-11 见 90-CONTEXT.md；VERIFICATION gaps 唯一项（REQUIREMENTS 命名漂移）已修复 3a2efe5

## Accumulated Context (carried forward)

### Decisions to preserve

- D-PRINCIPLE (v1.29 全局): 行业最佳实践为唯一依据,允许任何形式重构;去除硬编码/处理 todo/消除重复/合理抽象
- leaf const pkg + AST 锁值(Stability+Count 双锁)模式：Phase 89/90 先例，后续 phase 沿用
- 常量命名不加 Default 前缀（D-05，89/90 一致）
- D-26-01..05 (v1.26 后端覆盖率 gate): 4 层 CI 防倒退继续生效
- D-27-01..04 (v1.27 测试基建): miniredis/httpmock/ScrapliWrapper/LDAPClientIface/TestHelperProcess/AST守护 沿用
- operlog regression_test.go: 11 强制敏感关键词、25 OperType 常量全程保持绿
- status_constants_test.go: 状态 0/1 命名常量全程 AST 锁值

### Workstream 同步修复 (2026-09-04, 本 commit)

- `.planning/workstreams/milestone/ROADMAP.md` 由 stale v1.27 内容重写为 v1.29 追踪格式
- 根因：workstream roadmap 未随 v1.29 启动同步，phase-complete 在 Phase 90 后误报 is_last_phase=true
- 91-95 现已在 Progress 表注册，next_phase 可正确解析为 91

### Blockers (active)

- 无

### Pending Todos (carry forward, not in v1.29 scope)

- `.planning/todos/pending/operlog-exclude-paths.md` — operlog 白名单配置驱动（RPA heartbeat 日志污染），独立 deferred

## Next Step

`/gsd:discuss-phase 91 --chain` — CRUD 复用 base.Repository[T]（P1，4 plans 预估）

或并行推进: 93 (config_backup，与 91 互不依赖) / 94 (前端 API 工厂化，完全独立)
