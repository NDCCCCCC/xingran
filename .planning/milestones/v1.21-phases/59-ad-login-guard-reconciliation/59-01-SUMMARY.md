---
phase: 59-ad-login-guard-reconciliation
plan: 01
subsystem: backend-auth
tags: [ad-sync, outbox-lite, gorm, sqlite, guard, dept-mapping]

requires:
  - phase: 57-ad-outbox-lite
    provides: ad_push_queue 表 + ADPushQueueActivePredicate 单一事实源 + idx_ad_push_queue_entity_id
  - phase: 58-compare-and-move
    provides: update_user 队列行的排水语义与终态集合（pushed/dead）

provides:
  - AD 登录 dept_id 回写的活跃 update_user 队列守卫（pending/failed 拦截 + 查询失败保守降级 + WARN 全量诊断字段）
  - 六项自动化行为断言（pending/failed/终态/disable/无行/查询失败）

affects: [60-soft-delete-recovery, 59-02-view, 59-03-endpoint]

actuals:
  tokens: 3820
  tasks: 2
  commits: 2
  plan_head_before: 0b4082b3032ace91560d134daf79c963a4780d11
  plan_head_after: 7334e169a22dffd49a538c6df72190ffcf9ecf23

tech-stack:
  added: []
  patterns:
    - "Signal guard: query the existing outbox queue predicate before a single writeback point instead of timestamp arbitration"
    - "Fixture must prove the guarded branch is reached (positive-control cases) to prevent vacuous green tests"

key-files:
  created:
    - internal/services/system/user_sync_service_phase59_test.go
  modified:
    - internal/services/system/user_sync_service.go

key-decisions:
  - "Guard is inserted inside the existing else-if chain and wraps only the single Update call (D-04); no parallel tryUpdateDept branch (D-08)"
  - "Queue predicate is reused verbatim from models.ADPushQueueActivePredicate (D-02); no table qualification, no rewritten text"
  - "Query failure is treated as a hit: skip dept_id overwrite + WARN, never return the error to the login path (D-07)"
  - "WARN renders user.DeptID via deptValue helper to avoid printing the pointer address (Pitfall 7)"
  - "Tests assert the AD-resolved dept UUID appears in the WARN, proving the guard fired after resolveDeptFromOU succeeded rather than upstream short-circuiting"

patterns-established:
  - "Guard query helper: findActiveUserUpdateQueue(ctx, userID) returns (*models.ADPushQueue, error) using Take for an EXISTS-style signal"
  - "Positive controls (no-row/terminal/disable -> dept changes) anchor that skip-path tests are not vacuous"

requirements-completed: [GUARD-01]

coverage:
  - id: D1
    description: "pending update_user queue row blocks dept_id overwrite and emits WARN with userID/username/localDept/adDept/queueID/actionType/attempts"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "go test ./internal/services/system/ -run TestPhase59_SyncADUser_ActivePendingQueueSkipsDept -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "failed update_user queue row also blocks dept_id overwrite"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "go test ./internal/services/system/ -run TestPhase59_SyncADUser_ActiveFailedQueueSkipsDept -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "pushed/dead terminal rows and disable-action rows do not guard; no-row path updates dept to the mapped value through the same Update call"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "go test ./internal/services/system/ -run 'TestPhase59_SyncADUser_(TerminalRowsDoNotGuard|DisableActionDoesNotGuard|NoQueueUpdatesDept)' -count=1"
        status: pass
    human_judgment: false
  - id: D4
    description: "queue query failure (dropped table) skips dept overwrite, warns, and does not fail the login sync"
    requirement: GUARD-01
    verification:
      - kind: unit
        ref: "go test ./internal/services/system/ -run TestPhase59_SyncADUser_QueueQueryErrorSkipsDeptAndWarns -count=1"
        status: pass
    human_judgment: false
  - id: D5
    description: "Phase 59 flagged assumption: GUARD-01 probe is unclassified/unresolved; acceptance is taken from ROADMAP SC1/SC2 and needs human review at phase verification"
    verification: []
    human_judgment: true
    rationale: "Plan frontmatter marks this as a FLAGGED ASSUMPTION with no backstop; a human should confirm SC1/SC2 wording is the authoritative acceptance source during phase verification."

duration: 12min
completed: 2026-10-19
status: complete
---

# Phase 59 Plan 01: 登录回写守卫（active update_user 队列信号）Summary

AD 登录回写 `dept_id` 前增加 outbox 活跃队列守卫：pending/failed 的 `update_user` 行或守卫查询失败时跳过覆盖并 WARN；无活跃行时仍走原唯一 `Update` 调用。

## Performance

- **Duration:** 12min
- **Started:** 2026-10-19T12:57:08Z
- **Completed:** 2026-10-19T13:08:20Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- `SyncADUser` 的唯一 `dept_id` 回写点现在受活跃队列信号保护：pending/failed 的 `update_user` 行命中即跳过；查询失败按 D-07 保守跳过；两种情况均写 WARN 且不阻断登录。
- WARN 含 `userID` / `username` / 本地 `dept_id` / AD 解析 `deptID`；命中分支另含队列行 `id` / `action_type` / `attempts`。
- 新增 6 个自动化用例（pending / failed / pushed+dead / disable / 无行 / 查询失败），全部用真实 `addomain.DeptOUmapper` 和 `ad_username` fixture 驱动，正向用例同时证明守卫分支确实被执行。

## Task Commits

1. **Task 1: 端到端 pending 队列守卫 + WARN** — `3ba3b124` (feat)
2. **Task 2: failed / 终态 / disable / 无行 / 查询失败分支** — `7334e169` (test)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `internal/services/system/user_sync_service.go` — 新增 `findActiveUserUpdateQueue`、`deptValue`；`SyncADUser` 的 else-if 链改为三分支 switch，唯一 `Update("dept_id")` 调用保留。
- `internal/services/system/user_sync_service_phase59_test.go` — 共用 fixture `setupPhase59GuardFixture` + 6 个 `TestPhase59_SyncADUser_*` 用例。

## Decisions Made

- 守卫严格复用 `models.ADPushQueueActivePredicate`，未复制/改写谓词文本，未加表名限定（D-02）。
- 查询失败与命中同等处理：跳过覆盖并 WARN，错误不上抛（D-07）。
- 无活跃行时保持 Phase 58 的完全相同代码路径，只多一次 `LIMIT 1` 查询（D-08）。
- 测试 fixture 的关键约束：`ad_username` 非空、`OUDN` 与 `mapping.ou_dn` 逐字符相等、真实 mapper；否则用例会空跑（已由正向用例 `_NoQueueUpdatesDept` 锚定）。

## Deviations from Plan

### Harness note (not a code deviation)

- **Protected-branch guard vs `branching_strategy: none`**: 该仓库 `.planning/config.json` 配置为 `branching_strategy: "none"`，历史与本次 phase 的计划提交均在 `main`；GSD 的 `git.base-branch --is-protected main` 返回 `true`，而其自身的 `execute-phase/steps/protected-branch.md` 对 `branching_strategy: none` 规定「warn and continue on the current branch」。按 orchestrator 的 `<sequential_execution>` 指示，本次在两个任务提交时继续在配置分支 `main` 上提交（未修改任何配置、未使用 `--no-verify`）。如仓库策略需要改为分支提交，请由人工决定。

### Auto-fixed Issues

None.

**Total deviations:** 0 auto-fixed. **Impact on plan:** 无代码层面偏差。

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- GUARD-01 的实现与自动化证据已就位；59-02（视图 migration_217）可继续执行，不依赖本计划新增的任何表或配置。
- 真环境仍建议在 phase verification 时人工复核 ROADMAP SC1/SC2 的验收口径（plan frontmatter 的 FLAGGED ASSUMPTION）。

## Self-Check: PASSED

- FOUND: internal/services/system/user_sync_service.go
- FOUND: internal/services/system/user_sync_service_phase59_test.go
- FOUND: 3ba3b124 (task 1)
- FOUND: 7334e169 (task 2)
- `go build ./...` exit 0; `go test ./internal/services/system/ -count=1` pass; 6 个 `TestPhase59_SyncADUser_*` 全部 PASS.
- Source counts: `Model(user).Update("dept_id"` = 1; `models.ADPushQueueActivePredicate` = 1; `SyncADUser` = 1.

---
*Phase: 59-ad-login-guard-reconciliation*
*Completed: 2026-10-19*
