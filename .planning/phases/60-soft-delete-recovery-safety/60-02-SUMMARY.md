---
phase: 60-软删恢复安全
plan: 02
subsystem: auth
tags: [ad, ldap, soft-delete, batch-sync, gorm, security]
requires:
  - phase: 60-01
    provides: "共享只读校验入口 checkSoftDeletedRestore 与 security.ErrSoftDeleteIdentityMismatch sentinel"
  - phase: 58-compare-and-move
    provides: "addomain.NormalizeDN 归一化比较的单一真实源（D-02 契约）"
provides:
  - "批量同步路径的软删恢复身份闸门（SAFE-01 批量侧）"
  - "BatchSyncResult.Skipped 的真实语义：策略跳过与同步失败可区分（D-06）"
  - "登录/批量同场景判定与 DB 终态一致的行为证明（SAFE-02）"
  - "Phase 60 验证映射、需求状态、路线图与项目状态回填"
affects: [60-verification, 62-hygiene]
actuals:
  tokens: 5995
  tasks: 2
  commits: 2
plan_head_before: 8cc496362ae62d9e86053533a50e32718aeb8479
plan_head_after: 065dac7521768271a705efd72290e939df4d6a1e
tech-stack:
  added: []
  patterns:
    - "批量流水线阶段 2 短路记账：策略拒绝在哈希/写库前移出工作队列并计入 Skipped"
    - "双路径一致性用例：同一 fixture 造型分别跑登录与批量，断言判定与 DB 终态双一致"
key-files:
  created: []
  modified:
    - internal/services/system/user_sync_service.go
    - internal/services/system/user_sync_service_phase60_test.go
    - .planning/phases/60-soft-delete-recovery-safety/60-VALIDATION.md
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/STATE.md
key-decisions:
  - "过滤位置固定在阶段 2 分类日志之后、阶段 3 哈希之前：被拒用户不做 60 万次 PBKDF2，也不进入 runInBatchedTx"
  - "拒绝短路不抛 error：避免策略拒绝被降级管道粉饰成 DB 故障并计入 Failed"
  - "响应中的 SyncError.Error 固定为不含 DN 的中文；DN 细节只进 path=batch 的 [SECURITY] WARN"
  - "批量用例 OuDn 置空、defaultRoleID 传空串，规避 nil mapper 与角色表依赖"
patterns-established:
  - "批量策略拒绝记账：Skipped++ 与 DN-free 明细，保证 Total == Success+Failed+Skipped"
  - "双路径一致性回归：相同软删行分别经 SyncUserFromAD 与 BatchSyncADUsers 断言终态一致"
requirements-completed: [SAFE-01, SAFE-02]
coverage:
  - id: D1
    description: "批量路径 DN 不一致时拒绝恢复，计入 Skipped 与 Errors，保持软删且 ad_dn 原值不变"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Batch_MismatchedDN_Skipped"
        status: pass
    human_judgment: false
  - id: D2
    description: "批量路径归一化 DN 一致时恢复成功，deleted_at 清空且写入 AD 当前 DN"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Batch_MatchingDN_Restores"
        status: pass
    human_judgment: false
  - id: D3
    description: "混合批次一允一拒时 Total == Success + Failed + Skipped 守恒"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Batch_MixedUsers_Conservation"
        status: pass
    human_judgment: false
  - id: D4
    description: "同一不一致场景下登录返回 sentinel 拒绝、批量计入 Skipped，两库终态一致"
    requirement: SAFE-02
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_PathConsistency_MismatchRefusedBoth"
        status: pass
    human_judgment: false
  - id: D5
    description: "同一一致场景下登录与批量均恢复，两库 ad_dn 与活跃状态一致"
    requirement: SAFE-02
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_PathConsistency_MatchingRestoredBoth"
        status: pass
    human_judgment: false
  - id: D6
    description: "静态守卫（共用入口 2 处调用、Skipped 递增 1 处、过滤早于哈希、校验未下沉）与三包全量回归"
    requirement: SAFE-02
    verification:
      - kind: other
        ref: "grep -c 's.checkSoftDeletedRestore(' == 2; grep -c 'result.Skipped++' == 1; awk filter-before-hash exit 0; awk no-sink exit 0"
        status: pass
      - kind: integration
        ref: "go build ./... && go test ./internal/services/system/ ./internal/core/security/ ./internal/api/v1/ -count=1"
        status: pass
    human_judgment: false
  - id: D7
    description: "登录路径拒绝的用户可见文案（真实登录流 + 国密请求）"
    requirement: SAFE-01
    verification: []
    human_judgment: true
    rationale: "需真实 AD 登录流程与国密加密请求，自动化仅覆盖服务层原因码映射；已保留在 60-VALIDATION.md 的 Manual-Only 表"
metrics:
  duration: 8 min
  started: "2026-10-20T07:58:44Z"
  completed: "2026-10-20T08:07:16Z"
status: complete
---

# Phase 60 Plan 02: 软删恢复安全 - 批量路径收敛 Summary

**批量同步在阶段 2 用与登录路径相同的 NormalizeDN 校验过滤软删恢复，被拒项计入 Skipped 并绕开降级管道，双路径判定与 DB 终态由成对用例证伪。**

## Performance

- **Duration:** 8 min
- **Started:** 2026-10-20T07:58:44Z
- **Completed:** 2026-10-20T08:07:16Z
- **Tasks:** 2
- **Files modified:** 6 (0 created, 6 modified)

## Accomplishments

- `BatchSyncADUsers` 在阶段 2 分类日志之后、阶段 3 哈希之前调用共享 `checkSoftDeletedRestore(..., "batch")`，被拒项从 `plansByCat[catSoftDeleted]` 移除，永不做密码哈希、永不进入 `runInBatchedTx`。
- `BatchSyncResult.Skipped` 首次获得真实语义：策略拒绝计入 `Skipped` 并追加不含 DN 的 `SyncError`；响应可区分「被安全策略跳过」与「同步失败」。
- 5 个新用例锁定批量允许/拒绝/混合守恒与登录↔批量双路径一致性；静态守卫证明共用入口恰 2 处调用、校验未下沉到 `applyRestoreSoftDeleted`。
- Phase 60 的验证映射补齐为 4 行并签核为 validated；SAFE-01/SAFE-02 在 REQUIREMENTS 与 ROADMAP 标记 Complete，STATE 记录执行期决策与完成位置。

## Task Commits

1. **Task 1: 批量路径阶段 2 过滤（Skipped 记账 + 绕开降级管道）** - `19fefc32` (feat)
2. **Task 2: 收口（静态守卫、回归采样与文档回填）** - `065dac75` (docs)

**Plan metadata:** 见后续 docs commit

## Files Created/Modified

- `internal/services/system/user_sync_service.go` - 阶段 2 过滤块：共享校验、`Skipped++`、DN-free 明细、移出工作队列
- `internal/services/system/user_sync_service_phase60_test.go` - 抽出 `insertPhase60SoftDeletedUser`；新增 3 个批量用例与 2 个双路径一致性用例
- `.planning/phases/60-soft-delete-recovery-safety/60-VALIDATION.md` - 4 行验证映射、Wave 0 勾选、签核 validated
- `.planning/REQUIREMENTS.md` - SAFE-01/SAFE-02 勾选并写入证据
- `.planning/ROADMAP.md` - Wave 2 计划勾选、Phase 60 进度 2/2 Complete
- `.planning/STATE.md` - Current Position 标记 COMPLETE、执行期决策、性能指标与会话信息

## Decisions Made

- **过滤位置 = 阶段 2 之后、阶段 3 之前**：同时满足 D-06（不进降级管道）与性能目标（被拒用户不做 60 万次 PBKDF2）。
- **拒绝不返回 error**：`runInBatchedTx` 的错误契约会把策略拒绝降级重试并计入 `Failed`；短路跳过是唯一正确形态。
- **DN 只进 WARN**：`SyncError.Error` 固定中文文案，满足 D-04/D-05 的信息泄露约束。
- **测试复用既有基座**：抽 helper 而非复制 fixture；批量用例 `OuDn` 置空、`defaultRoleID` 空串，规避 nil mapper 与角色表依赖。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. 三包全量回归（`internal/services/system`、`internal/core/security`、`internal/api/v1`）全绿；未触发已知的 SNMP 全量并发 flake。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SAFE-01/SAFE-02 已闭环，Phase 60 两个 plan 全部完成，可进入 `/gsd-verify-work 60`。
- `user_ou_service.go` / `GetUserDeptByADOU` / `BatchMoveUsersToNewOU` 全程未改动，Phase 62 HYGIENE-03 的对照载体完整。
- 已接受风险（RESEARCH A4）：阶段 2 分类快照与阶段 5c 写库之间存在窄 TOCTOU 窗口，本 phase 不引入事务内重读。
- Manual-Only：登录路径拒绝的用户可见文案仍需真实 AD 登录流 + 国密请求验证。

## Self-Check: PASSED

- `internal/services/system/user_sync_service.go` 与测试文件存在且任务提交为 HEAD 祖先（`19fefc32`、`065dac75`）
- `go build ./...` 通过；三个受影响包 `go test -count=1` 全绿
- Task 1 静态守卫：共用入口调用 2 处、`result.Skipped++` 1 处、过滤块早于 `hashPasswordsConcurrent`、`applyRestoreSoftDeleted` 内无校验
- Task 2 回填校验：VALIDATION 4 行 + validated 签核、SAFE-01/02 勾选与 Complete、ROADMAP 2/2 + 两个 `[x]`、STATE COMPLETE 标记全部命中

---
*Phase: 60-软删恢复安全*
*Completed: 2026-10-20*
