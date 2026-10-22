---
phase: 61-shadow-table-integrity
plan: 03
subsystem: backend
tags: [gorm, transaction, ad-sync, shadow-table, sqlite-test, tdd]

# Dependency graph
requires:
  - phase: 61-02
    provides: 四类 upsert 软删行复活路径（existing Unscoped 化 + OnConflict 补 deleted_at）——本 plan 的事务化与其共存于同一写路径
provides:
  - syncGroupMembers Delete+Create 单事务化（每组一个事务，三调用方零改动受益）
  - fixture 表级 CHECK (user_dn != 'FORCED_FAIL') 失败注入点（可复用的回滚证伪基建）
  - TestSyncGroupMembers_CreateFailRollsBackDelete / TestSyncGroupMembers_ReplayIdempotent 行为锁定
affects: [Phase 62 (HYGIENE 收尾), ad-domain 组同步运维]

# Actuals (#2632)
actuals:
  tokens: 28000
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "fixture 表级 CHECK 注入：建表 DDL 加 CHECK (col != '哨兵值')，用一次 INSERT 整句被拒制造 Create 半途失败，证伪回滚语义（sqlite/PG 双方言可用）"

key-files:
  created: []
  modified:
    - internal/services/addomain/sync.go
    - internal/services/addomain/sync_78_05_test.go
    - .planning/phases/61-shadow-table-integrity/61-VALIDATION.md
    - .planning/REQUIREMENTS.md
    - .planning/STATE.md
    - .planning/ROADMAP.md

key-decisions:
  - "每组一个事务（不合并为大事务）：三个调用方（syncGroups 全量 / SyncSingleGroup 单组 / syncGroupEntries 增量）零改动自动受益，单组失败不拖垮整轮"
  - "Delete 的 Unscoped() 硬删语义保留：sys_ad_group_member 是派生数据每轮重放，软删化会行数爆炸 + GetGroupSyncStatus 计数虚高（plan prohibition 保持）"
  - "空成员集语义不变：Delete 随事务提交、Create 跳过，与改造前行为一致"
  - "SHADOW-02 三调用方覆盖口径：SyncSingleGroup 无直接单测（既有缺口非本 phase 引入，仅 ad_domain_handler.go:782 调用），由源断言零改动 + 全包/全套件零回归承担"

patterns-established:
  - "CHECK 注入证伪回滚：见 tech-stack.patterns"

requirements-completed: [SHADOW-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "syncGroupMembers 的 Delete+Create 在单事务闭包内完成（一律用 tx）；Create 失败时 Delete 回滚、旧成员原样"
    requirement: SHADOW-02
    verification:
      - kind: unit
        ref: "internal/services/addomain/sync_78_05_test.go#TestSyncGroupMembers_CreateFailRollsBackDelete"
        status: pass
    human_judgment: false
  - id: D2
    description: "同一组同一成员集重放两次，成员行集收敛一致（Delete-all + Create-all 幂等）"
    requirement: SHADOW-02
    verification:
      - kind: unit
        ref: "internal/services/addomain/sync_78_05_test.go#TestSyncGroupMembers_ReplayIdempotent"
        status: pass
    human_judgment: false
  - id: D3
    description: "Unscoped 硬删语义与空成员集语义保持；三个调用方零改动且行为不回归"
    requirement: SHADOW-02
    verification:
      - kind: unit
        ref: "go test ./internal/services/addomain/ -count=1（全包 ok，含既有 TestSync78_SyncGroupMembers_FullDiff 计数断言）"
        status: pass
      - kind: integration
        ref: "go test ./... -count=1（78 包 ok / 0 FAIL，2026-10-21 orchestrator 独立复跑）"
        status: pass
    human_judgment: false
  - id: D4
    description: "真并发下空成员可见窗口消除的端到端观察（第二会话查 sys_ad_group_member）"
    verification: []
    human_judgment: true
    rationale: "事务隔离下的并发可见性需真 PG 双会话观察，sqlite 单测测不出——归 61-VALIDATION.md Manual-Only"

# Metrics
duration: 45min（executor 01:33–02:19 被 API quota 429 中断；Task 2 收尾由 orchestrator 依盘上产物 reconcile 完成）
completed: 2026-10-21
status: complete
---

# Phase 61 Plan 03: syncGroupMembers 事务化 Summary

**syncGroupMembers 的 Delete+Create 包进 GORM `Transaction` 闭包（每组一个事务），以 fixture 表级 CHECK 注入证伪「Create 半途失败永久丢成员」，重放幂等锁定，三调用方零改动**

## Performance

- **Duration:** ~45 min（executor 中断于 Task 2 收尾，orchestrator reconcile 完成）
- **Started:** 2026-10-21T01:33 (dispatch)
- **Completed:** 2026-10-21T02:19 (生产提交) / 收尾 commit 本日
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- `syncGroupMembers`（sync.go:672）Delete+Create 全部移入 `s.db.WithContext(ctx).Transaction` 闭包，闭包体内一律用 `tx`，消除空成员可见窗口与「Create 半途失败永久丢成员」
- fixture `setupSync78DB` 的 `sys_ad_group_member` 建表 DDL 追加 `CHECK (user_dn != 'FORCED_FAIL')` 失败注入点；`TestSyncGroupMembers_CreateFailRollsBackDelete` 以之证伪回滚语义（先 RED 后 GREEN，Unscoped 查询旧成员原样）
- `TestSyncGroupMembers_ReplayIdempotent` 锁定重放幂等
- `Unscoped()` 硬删语义保留（plan prohibition 源断言通过）；空成员集语义不变
- 三个调用方零改动：`sync.go syncGroups` / `group_sync_service.go SyncSingleGroup` / `syncGroupEntries`
- phase 收口回填：61-VALIDATION.md（Per-Task Verification Map 按实际编号回填 + status: validated）、REQUIREMENTS.md（SHADOW-02 → Complete）、STATE.md、ROADMAP.md（3/3 plans）

## Task Commits

1. **Task 1: syncGroupMembers 事务化 + CHECK 失败注入证伪 + 重放幂等锁定** — `508b9b9e` (test, RED) → `cb1382c3` (feat, GREEN)
2. **Task 2: 三调用方回归采样 + phase 收口回填** — 文档改动与 SUMMARY 合并入收尾 docs commit（见下）

## Files Created/Modified

- `internal/services/addomain/sync.go` — syncGroupMembers 事务化（cb1382c3）
- `internal/services/addomain/sync_78_05_test.go` — CHECK 注入 fixture + 2 新用例（508b9b9e / cb1382c3）
- `.planning/phases/61-shadow-table-integrity/61-VALIDATION.md` — 验证矩阵回填 + status: validated
- `.planning/REQUIREMENTS.md` — SHADOW-02 → Complete（附证据）
- `.planning/STATE.md` — Phase 61 收口状态
- `.planning/ROADMAP.md` — 61-03 勾选 + Phase 61 进度表 3/3

## Decisions Made

见 key-decisions（每组单事务不合并、Unscoped 保留、SyncSingleGroup 无直接单测的覆盖口径披露）。

## Deviations from Plan

None - plan executed exactly as written（生产与测试产物与 plan must_haves 逐项对上；文档回填含计划内的占位行收敛与覆盖口径披露，见 61-VALIDATION.md 回填注记）。

## Issues Encountered

- **executor 于 Task 2 收尾被 API 429 quota 中断**（5 小时用量上限，04:52 重置）：Task 1 生产提交（`508b9b9e` + `cb1382c3`）与 Task 2 四个文档改动均已落盘但未提交、SUMMARY 缺失。orchestrator 按 completion-reconciliation（#4217）以盘上产物 reconcile：独立复跑 `go build ./...`（EXIT=0）、addomain 全包（ok）、全套件 `go test ./...`（78 包 ok / 0 FAIL）后补写本 SUMMARY 并修复 STATE.md 中 executor 误写的零填充假 `state_head`（改为真实完整 SHA）。中断不影响任何生产代码正确性。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 61 三 plan 全部完成（SHADOW-01/02/03 全部 Complete）；剩余 Manual-Only 三项（真并发空窗观察等）归 61-VALIDATION.md，随 verify-work 走人工项
- Phase 62（HYGIENE 收尾：F-22 旧路径删除 + 嵌套组标记 + 同名部门 OU DN 冲突告警）就绪，其前置 SAFE-01（Phase 60）已满足

## Self-Check: PASSED

- `[ -f ]` 两个改动源文件在盘：sync.go / sync_78_05_test.go ✓
- `gsd_run check evaluation-scope --plan "61-03" --commits-only` 返回 2 个提交（508b9b9e test / cb1382c3 feat，本分支可达）✓
- `go build ./...` EXIT=0（orchestrator 独立复跑）✓
- `go test ./internal/services/addomain/ -count=1` ok ✓
- `go test ./... -count=1` 78 包 ok / 0 FAIL（orchestrator 独立复跑）✓
- 无 `## Self-Check: FAILED` 标记 ✓

---
*Phase: 61-shadow-table-integrity*
*Completed: 2026-10-21*
