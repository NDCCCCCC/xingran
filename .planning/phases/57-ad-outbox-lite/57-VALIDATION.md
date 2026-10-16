---
phase: 57
slug: ad-outbox-lite
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-10-16
revised: 2026-10-16
---

# Phase 57 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `.planning/phases/57-ad-outbox-lite/57-RESEARCH.md` (Validation Architecture section)
> 本文件已按 57-01 / 57-02 / 57-03 三个 plan 的**实际测试名**重写（原版本的 `TestPhase57_MigrationSetup` / `TestPhase57_UpdateEnqueue_TxAtomic` / `TestPhase57_DeleteEnqueue` / `TestPhase57_Precondition` 在任何 plan 中都不存在，已全部替换）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/require` + `github.com/glebarez/sqlite` (sqlite in-memory) |
| **Config file** | None — sqlite tests use `gorm.Open(sqlite.Open(":memory:"))` (precedent `internal/services/system/user_service_test.go:49`) |
| **Test file** | `internal/services/system/ad_push_queue_test.go`（Phase 57 全部新测试的唯一落地文件，三个 plan 共用追加） |
| **Quick run command** | `go test ./internal/services/system/ -run TestPhase57 -v` |
| **Second env sentinel** | `go test ./internal/api/v1/system/ -run "TestUserHandler_Delete\|TestUserHandler_BatchDelete" -v`（第二套 sqlite 环境） |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds (test suite + build) |

---

## Sampling Rate

- **After every task commit:** `go build ./...` + `go test ./internal/services/system/ -run TestPhase57 -v`
- **After 57-03 Task 1:** 额外跑第二套环境哨兵（见上表 Second env sentinel）
- **After every plan wave:** `go test ./...` + `go vet ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~15 seconds (single package) / ~60 seconds (full suite)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Test Name (as written in plan) | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------------------------|-------------|--------|
| 57-01-01 | 01 | 1 | HYGIENE-06 (sqlite 迁移建表) / QUEUE-02 (唯一索引语义) | T-57-04, T-57-05 | migration_215 双方言 DDL 在 sqlite 内 in-memory 可建；部分唯一索引夺锁生效；模型无 DeletedAt / 无 uniqueIndex tag | unit | `go test ./internal/services/system/ -run "TestPhase57_Migration215" -v` | `TestPhase57_Migration215_SQLiteDDL`、`TestPhase57_Migration215_PartialUniqueIndex`、`TestPhase57_ADPushQueue_ModelRoundTrip` | ❌ W0 | ⬜ pending |
| 57-01-02 | 01 | 1 | QUEUE-02 (DB 层合并基础) + HYGIENE-06 (无软删) | T-57-05 | `SELECT count(*) FROM pragma_table_info('ad_push_queue') WHERE name='deleted_at'` == 0；裸 INSERT 显式传 `uuid.NewString()`（sqlite 非 INTEGER 主键可 NULL，防 fixture 静默退化） | unit | `go test ./internal/services/system/ -run "TestPhase57_Migration215_PartialUniqueIndex" -v` | 同上 + 裸 INSERT 不带 id 的行为记录断言 | ❌ W0 | ⬜ pending |
| 57-01-03 | 01 | 1 | HYGIENE-06 (模型与迁移 schema 一致) | T-57-05 | 模型 tag 与 DDL 逐列对齐；`grep -v '^[[:space:]]*//' ... \| grep -c "DeletedAt"` == 0 且 `grep -c 'gorm:"[^"]*uniqueIndex'` == 0（排除注释行，因任务注释本身要写明"不含 DeletedAt"） | unit + source-assert | `go test ./internal/services/system/ -run "TestPhase57_ADPushQueue_ModelRoundTrip" -v` | `TestPhase57_ADPushQueue_ModelRoundTrip` | ❌ W0 | ⬜ pending |
| 57-02-01 | 02 | 2 | QUEUE-01 (四字段 diff 入队) + D-02 第一层 (无 AD 配置跳过) | T-57-02 | diff 纯函数 nil/空串/TrimSpace 边界；无启用行 → 不入队 + return nil（业务放行） | unit | `go test ./internal/services/system/ -run "TestPhase57_HasADFieldChanged_Boundaries\|TestPhase57_EnqueueUpdate_EnqueuesOnlyOnRealChange\|TestPhase57_EnqueueUpdate_NoADConfig_SkipsEnqueue" -v` | `TestPhase57_HasADFieldChanged_Boundaries`、`TestPhase57_EnqueueUpdate_EnqueuesOnlyOnRealChange`、`TestPhase57_EnqueueUpdate_NoADConfig_SkipsEnqueue` | ❌ W0 | ⬜ pending |
| 57-02-02 | 02 | 2 | QUEUE-02 (同实体合并 / failed 行打回 pending) | T-57-06 | ON CONFLICT **TargetWhere** 谓词文本级命中部分索引；连续两次入队恒 1 行；failed 行打回 pending 并清空 attempts/last_error/failed_at/next_retry_at；pushed 终态行不被覆盖 | unit | `go test ./internal/services/system/ -run "TestPhase57_EnqueueUpdate_MergeOnSecondEdit\|TestPhase57_EnqueueUpdate_SyncsWithTerminalRow" -v` | `TestPhase57_EnqueueUpdate_MergeOnSecondEdit`、`TestPhase57_EnqueueUpdate_SyncsWithTerminalRow` | ❌ W0 | ⬜ pending |
| 57-02-03 | 02 | 2 | QUEUE-06 (disable 入队) + D-02 第二层 (错误上抛) | T-57-01, T-57-03 | disable 单条/批量/去重/重复合并/无 AD 配置；DROP 表后入队返回**非 nil** error（错误未被吞掉） | unit | `go test ./internal/services/system/ -run "TestPhase57_EnqueueDisable\|TestPhase57_Enqueue_PropagatesInsertFailure" -v` | `TestPhase57_EnqueueDisable_SingleAndBatch`、`TestPhase57_EnqueueDisable_DedupesDuplicateIDs`、`TestPhase57_EnqueueDisable_MergeOnRepeatCall`、`TestPhase57_EnqueueDisable_NoADConfig_SkipsEnqueue`、`TestPhase57_Enqueue_PropagatesInsertFailure` | ❌ W0 | ⬜ pending |
| 57-03-01 | 03 | 3 | HYGIENE-06 (**两套** sqlite 环境补表) | T-57-10 | `setupUserServiceTestDB`（环境①）与 `setupUserTestDB`（`internal/api/v1/system/user_handler_test.go:110`，环境②）都建 `sys_ad_config`（含 `deleted_at`）+ 直调 `migrations.Migrate215CreateADPushQueue`；TC9/TC12 的 `require 200` 断言保持绿 | unit | `go test ./internal/api/v1/system/ -run "TestUserHandler_Delete_Success\|TestUserHandler_BatchDelete_Success" -v` | `TestUserHandler_Delete_Success`（既有 TC9）、`TestUserHandler_BatchDelete_Success`（既有 TC12）+ 新增 6 例 `TestPhase57_UserService_*` | ❌ W0 | ⬜ pending |
| 57-03-02 | 03 | 3 | QUEUE-01 (事务内入队 + 失败整体回滚) | T-57-03, T-57-07 | 入队点在 `Transaction` 闭包内 `return nil` 之前；队列表写失败 → `return apperrors.DatabaseError(err)` → `sys_user.nickname` 保持 NULL | unit | `go test ./internal/services/system/ -run "TestPhase57_UserService_Update" -v` | `TestPhase57_UserService_Update_EnqueuesInSameTransaction`、`TestPhase57_UserService_Update_NoADFieldChange_NoEnqueue`、`TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails` | ❌ W0 | ⬜ pending |
| 57-03-03 | 03 | 3 | QUEUE-06 (Delete / BatchDelete disable 入队) | T-57-01, T-57-08 | Delete/BatchDelete 事务内产生 `action_type='disable'` 行；停用配置下业务照常、队列为空；`go test ./...` 全绿 | unit + full suite | `go test ./internal/services/system/ -run "TestPhase57_UserService_Delete\|TestPhase57_UserService_BatchDelete" -v` && `go test ./...` | `TestPhase57_UserService_Delete_EnqueuesDisable`、`TestPhase57_UserService_BatchDelete_EnqueuesDisable`、`TestPhase57_UserService_Delete_NoADConfig_NoEnqueue` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/system/ad_push_queue_test.go` — Phase 57 全部新测试的**唯一**落地文件，由 57-01 Task 1 创建（RED），57-02 Task 2/Task 3 与 57-03 Task 1 追加。覆盖：
  - QUEUE-01（diff + 事务内入队 + INSERT 失败回滚）
  - QUEUE-02（同实体合并 + failed 行打回 pending + 终态行并存）
  - QUEUE-06（Delete / BatchDelete disable 入队 + 批量去重）
  - D-02 第一层（无 AD 配置跳过）/ 第二层（入队错误上抛）
  - HYGIENE-06（sqlite 迁移 DDL + 部分唯一索引 + 模型往返）
- [x] ~~`internal/core/db/migrations/migration_215_create_ad_push_queue_test.go`~~ — **本 phase 不创建**（仿 `migration_212` 模式，migrations 包无 `_test.go` 文件）。双方言 DDL 的 smoke 已由环境①（`TestPhase57_Migration215_SQLiteDDL`）覆盖，PG 分支由 UAT 覆盖。此项从 Wave 0 清单移除，不再作为 pending 项。
- [x] Framework install: 既有（无新增依赖）

*Total Wave 0 file: 1 (`internal/services/system/ad_push_queue_test.go`)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 进程重启不丢失已入队行 | QUEUE-01 | 需真实部署二进制 + 启停 | 在 PG 部署后 UPDATE sys_user → 入队 → kill -9 进程 → 重启 → `SELECT * FROM ad_push_queue WHERE status='pending'` 验证行存在 |
| ON CONFLICT 谓词字段正确（TargetWhere 而非 Where） | QUEUE-02 | PG 特定错误码 42P10，sqlite 不完全复现 PG 的推断路径 | PG 部署后连续两次编辑同一用户：若失败报 `42P10 there is no unique or exclusion constraint matching the ON CONFLICT specification` → 谓词写进了 `Where` 而非 `TargetWhere`；若成功且队列表仅 1 行 → 正确。sqlite 侧 `TestPhase57_EnqueueUpdate_MergeOnSecondEdit` 已提供等价但更快的回归保护 |
| D-04 复活链路守门 | QUEUE-06 | 涉及 AD 真实 bind + disable UAC 514 | AD 端真实操作：sys 删除用户 → AD bind 期望失败（UAC 533） |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 9/9 tasks have `<automated>`
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — max run is 2 tasks between verifies
- [x] Wave 0 covers all MISSING references — single Wave 0 file (`ad_push_queue_test.go`) declared and claimed by 57-01 Task 1; the previously-listed migrations-package test file was never claimed by any plan and has been removed from this list
- [x] No watch-mode flags
- [x] Feedback latency < 60s — single-package run ~15s, full suite ~60s
- [x] `nyquist_compliant: true` set in frontmatter
- [x] `wave_0_complete: true` set in frontmatter — Wave 0 scope is fully enumerated above (1 file, 3 helper/test entry points); execution status of the tests themselves is tracked in the Status column above, not here

**Approval:** ready for execution (pending actual green runs)
