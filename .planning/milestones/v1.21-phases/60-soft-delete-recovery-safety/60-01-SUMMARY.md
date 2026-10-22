---
phase: 60-软删恢复安全
plan: 01
subsystem: auth
tags: [ad, ldap, soft-delete, gorm, gin, security]
requires:
  - phase: 58-compare-and-move
    provides: "addomain.NormalizeDN 作为 DN 归一化比较的单一真实源（D-02 契约）"
  - phase: 59-ad-login-guard-reconciliation
    provides: "结构化 WARN 风格与同步失败不崩认证的既有管道"
provides:
  - "登录路径软删恢复前的 AD DN 身份闸门（SAFE-01）"
  - "两路径共用的只读校验入口 checkSoftDeletedRestore（SAFE-02 前提）"
  - "专用 SyncErrorReason soft_delete_mismatch 与固定中文用户文案（D-04）"
  - "拒绝审计 WARN 唯一产出点（D-05）"
affects: [60-02, 62-hygiene]
actuals:
  tokens: 4563
  tasks: 2
  commits: 2
plan_head_before: 454f02ee72cc68c401ab18cf0e200b5f4afc0743
plan_head_after: 1d7b3e864a8a040924187dbb733bd104d8c00669
tech-stack:
  added: []
  patterns:
    - "sentinel error 跨包传播（system → security via %w + errors.Is）"
    - "只读共享校验入口：判定 + 审计日志集中一处，调用点只决定拒绝后的动作"
key-files:
  created:
    - internal/services/system/user_sync_service_phase60_test.go
    - internal/core/security/ad_authenticator_phase60_test.go
  modified:
    - internal/core/security/authenticator.go
    - internal/core/security/ad_authenticator.go
    - internal/api/v1/auth.go
    - internal/services/system/user_sync_service.go
    - internal/api/v1/auth_80_03_test.go
key-decisions:
  - "sentinel 放在 security 包：避免 security 反向 import system 形成环"
  - "原因码采用 soft_delete_mismatch 常量，两处引用同一常量防漂移"
  - "校验入口返回 bool 并自行写唯一 WARN；登录路径拒绝时 %w 包装 sentinel"
  - "策略拒绝日志降为 WARN 并与基础设施故障的 ERROR 区分"
patterns-established:
  - "登录路径安全闸门：分类后、哈希前短路，拒绝零写入"
requirements-completed: [SAFE-01, SAFE-02]
coverage:
  - id: D1
    description: "共享校验入口判定表：nil/空串/纯空白/归一化一致/段内空格放行，不一致拒绝"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_CheckSoftDeletedRestore_Predicate"
        status: pass
    human_judgment: false
  - id: D2
    description: "登录路径归一化 DN 一致时恢复软删用户并写入 AD 当前 DN"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Login_MatchingDN_Restores"
        status: pass
    human_judgment: false
  - id: D3
    description: "登录路径 DN 不一致时拒绝，行保持软删且存量 ad_dn 逐字不变（二次登录仍拒绝）"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Login_MismatchedDN_Refuses"
        status: pass
    human_judgment: false
  - id: D4
    description: "存量 ad_dn 为 NULL 时放行恢复（D-01）"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Login_EmptyStoredDN_Restores"
        status: pass
    human_judgment: false
  - id: D5
    description: "拒绝审计 WARN 含 userID/username/存量 DN/当前 DN/path=login 且不含用户文案"
    requirement: SAFE-01
    verification:
      - kind: unit
        ref: "internal/services/system/user_sync_service_phase60_test.go#TestPhase60_Login_RefusalAuditWarn"
        status: pass
    human_judgment: false
  - id: D6
    description: "错误到 SyncErrorReason 的纯函数映射（含多层包装）"
    requirement: SAFE-02
    verification:
      - kind: unit
        ref: "internal/core/security/ad_authenticator_phase60_test.go#TestPhase60_SyncErrorReasonFor"
        status: pass
    human_judgment: false
  - id: D7
    description: "专用原因码映射为固定中文文案（不含 DN 片段）"
    requirement: SAFE-02
    verification:
      - kind: unit
        ref: "internal/api/v1/auth_80_03_test.go#TestAuth8003_SyncErrorReasonMessage"
        status: pass
    human_judgment: false
metrics:
  duration: 17 min
  started: "2026-10-20T07:02:00Z"
  completed: "2026-10-20T07:19:37Z"
status: complete
---

# Phase 60 Plan 01: 软删恢复安全 - 登录路径身份闸门 Summary

**登录路径软删恢复在哈希与写库前增加 NormalizeDN 身份校验，拒绝经 sentinel → 专用原因码 → 固定中文文案传播，且拒绝零写入、DN 仅进 WARN。**

## Performance

- **Duration:** 17 min
- **Started:** 2026-10-20T07:02:00Z
- **Completed:** 2026-10-20T07:19:37Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- 新增 `security.ErrSoftDeleteIdentityMismatch` sentinel 与 `SyncReasonSoftDeleteMismatch` 原因码常量，登录失败原因链可区分「安全拦截」与「AD 基础设施故障」。
- `checkSoftDeletedRestore` 成为两路径共用的**只读**校验入口：空存量 DN 放行（D-01），`NormalizeDN` 归一化比较（D-02），不一致时写唯一一条结构化 `[SECURITY]` WARN（D-05）。
- `restoreSoftDeletedUser` 在 `HashPassword` 与任何写库之前短路；拒绝后行保持软删、`ad_dn` 逐字不变（D-03），二次登录无法靠空 DN 绕过。
- `ad_authenticator` 新增纯函数 `syncErrorReasonFor`，策略拒绝日志降为 WARN；`auth.go` 映射为固定中文文案，且不含任何 DN 片段。
- 7 个自动化用例锁定判定表、恢复/拒绝/空 DN/审计字段/原因码映射/文案。

## Task Commits

1. **Task 1: 端到端登录路径接线** - `466acf04` (feat)
2. **Task 2: 行为与原因链测试** - `1d7b3e86` (test)

**Plan metadata:** 见后续 docs commit

## Files Created/Modified

- `internal/core/security/authenticator.go` - 新增 sentinel 与专用原因码常量，补充原因码注释
- `internal/core/security/ad_authenticator.go` - 新增 `syncErrorReasonFor` 纯函数；错误分支按 sentinel 分级日志并使用专用原因码
- `internal/api/v1/auth.go` - `syncErrorReasonMessage` 新增专用原因码 → 固定中文文案
- `internal/services/system/user_sync_service.go` - 新增共享只读校验入口并在 `restoreSoftDeletedUser` 开头短路；`SyncADUser` 错误日志分级
- `internal/services/system/user_sync_service_phase60_test.go` - 判定表 + 登录恢复/拒绝/空 DN/审计用例
- `internal/core/security/ad_authenticator_phase60_test.go` - 原因码映射纯函数用例
- `internal/api/v1/auth_80_03_test.go` - 原因码表驱动新增一行

## Decisions Made

- **sentinel 放在 `security` 包**：`system` 已 import `security`，反向声明会在 `security` 中 import `system` 形成循环；sentinel 是唯一无环方向。
- **原因码字符串只以常量出现**：`ad_authenticator` 与 `auth.go` 均引用 `security.SyncReasonSoftDeleteMismatch`，避免两处字面量漂移（Pitfall 7）。
- **校验入口返回 bool 且日志内聚**：拒绝后的动作由调用点决定（登录返回 sentinel、60-02 批量计入 Skipped），审计日志只在校验入口产出一次（D-05/D-07）。
- **日志级别按策略/故障分级**：策略拒绝是预期行为（WARN + `[SECURITY]`），基础设施故障保持 ERROR（D-04 的运维可区分目标）。

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 登录路径闸门与原因链已闭环，`checkSoftDeletedRestore` 已可供 60-02 批量路径直接复用。
- 60-02 需要：阶段 2 分类后调用同一入口、被拒项计入 `result.Skipped` 且不进入 `runInBatchedTx`，并补两路径一致性用例。
- 本 phase 未触碰 `user_ou_service.go` / `GetUserDeptByADOU` / `BatchMoveUsersToNewOU`（Phase 62 对照载体）。

## Self-Check: PASSED

- Created files exist on disk: 2 个测试文件均 FOUND
- Task commits are ancestors of HEAD: `466acf04` / `1d7b3e86` 均 FOUND
- Plan-level verification: `go build ./...` 通过；三个包 `go test -count=1` 全绿
- 需求 ID `SAFE-01` / `SAFE-02` 由 60-02 共享声明，`requirements.ready-ids` 返回 blocked，未标记 complete（由最后一个 plan 回填）

---
*Phase: 60-软删恢复安全*
*Completed: 2026-10-20*
