---
phase: 57-ad-outbox-lite
plan: 03
subsystem: services
tags: [outbox-lite, ad-sync, transaction-wiring, delete-disable, sqlite-fixture, hygiene]

requires:
  - phase: 57-ad-outbox-lite
    plan: 01
    provides: "models.ADPushQueue + migrations.Migrate215CreateADPushQueue（直调建表的测试路径）"
  - phase: 57-ad-outbox-lite
    plan: 02
    provides: "addomain.NewADPushEnqueuer().EnqueueUpdateIfChanged / EnqueueDisable + HasADFieldChanged + IsADSyncEnabled"

provides:
  - "user_service.Update 同事务 diff 入队接线（oldUserSnapshot 值拷贝 + EnqueueUpdateIfChanged(tx,...)）"
  - "user_service.Delete / BatchDelete 同事务 disable 入队接线（EnqueueDisable(tx, ...)）"
  - "三套 sqlite 测试环境均具备 sys_ad_config + ad_push_queue（HYGIENE-06 全覆盖，含计划外的第三套 setupUserCacheTestDB）"
  - "Wave 3 测试 6 例（TestPhase57_UserService_*），TestPhase57_ 累计 20 例全绿"

affects: [58-drainer, 59-login-guard, 60-safe-restore]

tech-stack:
  added: []
  patterns:
    - "入队调用点固定形态：事务闭包内、return nil 之前、传 tx、失败 return apperrors.DatabaseError(err)"
    - "sqlite fixture 建表顺序：业务表 → sys_ad_config（带 deleted_at）→ 直调 migrations.Migrate215CreateADPushQueue"
    - "Update 类 fixture 的 sys_user 必须补齐 models.User 全部列（tx.Save 写满所有字段，缺列即 SQL logic error 1）"

key-files:
  created: []
  modified:
    - internal/services/system/user_service.go
    - internal/services/system/user_service_test.go
    - internal/services/system/ad_push_queue_test.go
    - internal/services/system/user_cache_impl_test.go
    - internal/api/v1/system/user_handler_test.go

key-decisions:
  - "oldUserSnapshot 在 First 之后、事务闭包之前做值拷贝（user 是指针，闭包内重新赋值会毁掉 diff 旧值）"
  - "第三套 sqlite 环境（setupUserCacheTestDB，userCacheService 内嵌 userService）同样补表 —— 计划只列了两套，实际有三套复用同一写路径"

requirements-completed: [QUEUE-01, QUEUE-06, HYGIENE-06]

# Metrics
duration: 22min
completed: 2026-10-16
---

# Phase 57 Plan 03: user_service 三处事务内入队接线 + sqlite 测试环境补表 Summary

**Update 的四字段 diff 入队与 Delete/BatchDelete 的 disable 入队全部落在业务事务闭包内、失败即回滚，队列表写不可用时 sys_user 无任何残留；HYGIENE-06 实际覆盖到三套 sqlite 环境（含计划外的 userCacheService 那一套）**

## Performance

- **Duration:** 22 min
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- `Update`：import `addomain`；`First` 旧值后新增 `oldUserSnapshot := user` 值拷贝；事务闭包末尾（岗位关联块之后、`return nil` 之前）调 `EnqueueUpdateIfChanged(tx, &oldUserSnapshot, req)`，失败 `return apperrors.DatabaseError(err)` 使 Commit 跳过
- `Delete`：`tx.Delete(&user)` 之后调 `EnqueueDisable(tx, []string{id})`；`BatchDelete`：`tx.Where("id IN ?", ids).Delete(...)` 之后调 `EnqueueDisable(tx, ids)`（重复 id 由 Plan 02 的 map 去重吸收）
- 文件内 `go func` 命中数 0、`UpdateStatus` 零改动、`internal/api/v1/system/user_handler.go` 零改动（D-03 红线保住，旧推送路径 `adSyncInFlight` 保留至 Phase 58）
- 三套 sqlite 环境补齐 `sys_ad_config`（含 `deleted_at`）+ 直调 `Migrate215CreateADPushQueue`
- Wave 3 新增 6 例（Update 同事务入队 / 非四字段不入队 / 队列表不可写整体回滚 / Delete disable / BatchDelete disable / 配置停用放行），`TestPhase57_` 累计 20 例全绿，`go test ./...` 连续两轮 exit 0

## Task Commits

1. **Task 1: sqlite 测试环境补表 + service 层接线测试（先红）** - `ae345d46` (test)
2. **Task 2: Update 事务内 diff 入队接线（QUEUE-01）** - `a69ecd9c` (feat)
3. **Task 3: Delete / BatchDelete 事务内 disable 入队接线（QUEUE-06）+ 全套测试** - `b14e1fad` (feat)

## 三处接线点（行号 @ b14e1fad）

| 位置 | 插入内容 |
|------|----------|
| `user_service.go:11` | import `"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"` |
| `user_service.go:157` | `oldUserSnapshot := user` —— Phase 57 D-01 注释紧随其后 |
| `user_service.go:227-231`（Update 事务闭包尾） | `EnqueueUpdateIfChanged(tx, &oldUserSnapshot, req)` → `apperrors.DatabaseError(err)` |
| `user_service.go:267-272`（Delete 事务闭包尾） | `EnqueueDisable(tx, []string{id})` |
| `user_service.go:600-604`（BatchDelete 事务闭包尾） | `EnqueueDisable(tx, ids)` |

`git diff 8246173a..HEAD -- internal/api/v1/system/user_handler.go` 输出为空 —— D-03 红线守住。

## 快照捕获位置（接线中最易错的一点）

`Update` 中 `user` 是 `*models.User` 的解引用局部变量，事务闭包内 `user.Nickname = req.Nickname` 等只是改写局部变量指向的 struct。若直接把 `&user` 传入入队器，diff 的"旧值"就是新值，`HasADFieldChanged` 恒 false。故在 `First` 成功、`s.db.WithContext(ctx).Transaction(` 之前做值拷贝：

```go
oldUserSnapshot := user   // 值拷贝，闭包内的重新赋值不影响它
```

`models.User` 只含指针/值字段且无 slice 字段，值拷贝足够（不共享可变底层数组）。

## 测试环境改动（HYGIENE-06 实测三套）

| 环境 | helper | 改动 | 守护的既有测试 |
|------|--------|------|------------------|
| ① service 层 | `user_service_test.go:47 setupUserServiceTestDB` | sys_ad_config DDL + `Migrate215CreateADPushQueue` + sys_user 补全列 | `TestUserService_*` 全套 |
| ② handler 层 | `internal/api/v1/system/user_handler_test.go:110 setupUserTestDB` | 同构补表（仅 fixture） | TC9 `TestUserHandler_Delete_Success`、TC12 `TestUserHandler_BatchDelete_Success` 的 `require 200` |
| ③ cache service 层 | `user_cache_impl_test.go:88 setupUserCacheTestDB` | 同构补表（计划外，见偏差 3） | `TestUserCache_Delete_TriggersInvalidate`、`TestUserCache_BatchDelete_TriggersInvalidate` |

环境 ③ 的原因：`userCacheService` 内嵌 `userService`（`NewUserServiceWithCache`），复用同一套 `Delete` / `BatchDelete` / `Update` 事务体。缺表时它同样在 `IsADSyncEnabled` 处报表不存在并整体转红。

## 测试运行输出

Wave 3 RED（接线前，Task 1 commit `ae345d46`）：
```
--- FAIL: TestPhase57_UserService_Delete_EnqueuesDisable        实际 0 行（期望 1）
--- FAIL: TestPhase57_UserService_BatchDelete_EnqueuesDisable   实际 0 行（期望 2）
    Error:  Received unexpected error: record not found
--- PASS: TestUserService_Delete_Success（环境① 补表未破坏既有测试）
--- PASS: TestUserHandler_Delete_Success（环境② 补表未破坏既有测试）
```

Wave 3 GREEN（Task 3 后）：
```
--- PASS: TestPhase57_Migration215_SQLiteDDL
--- PASS: TestPhase57_Migration215_PartialUniqueIndex
--- PASS: TestPhase57_Migration215_BareInsertWithoutID
--- PASS: TestPhase57_ADPushQueue_ModelRoundTrip
--- PASS: TestPhase57_HasADFieldChanged_Boundaries
--- PASS: TestPhase57_EnqueueUpdate_EnqueuesOnlyOnRealChange
--- PASS: TestPhase57_EnqueueUpdate_NoADConfig_SkipsEnqueue
--- PASS: TestPhase57_EnqueueUpdate_MergeOnSecondEdit
--- PASS: TestPhase57_EnqueueUpdate_SyncsWithTerminalRow
--- PASS: TestPhase57_EnqueueDisable_SingleAndBatch
--- PASS: TestPhase57_EnqueueDisable_DedupesDuplicateIDs
--- PASS: TestPhase57_EnqueueDisable_MergeOnRepeatCall
--- PASS: TestPhase57_EnqueueDisable_NoADConfig_SkipsEnqueue
--- PASS: TestPhase57_Enqueue_PropagatesInsertFailure
--- PASS: TestPhase57_UserService_Update_EnqueuesInSameTransaction
--- PASS: TestPhase57_UserService_Update_NoADFieldChange_NoEnqueue
--- PASS: TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails
--- PASS: TestPhase57_UserService_Delete_EnqueuesDisable
--- PASS: TestPhase57_UserService_BatchDelete_EnqueuesDisable
--- PASS: TestPhase57_UserService_Delete_NoADConfig_NoEnqueue
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/system  2.653s

--- PASS: TestUserHandler_Delete_Success / EmptyID / NotFound
--- PASS: TestUserHandler_BatchDelete_Success / Empty
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/system  3.235s

go test ./...   → exit 0（连续两轮，FAIL 行数 0）
go vet ./internal/services/system/ ./internal/services/addomain/ ./internal/models/ ./internal/core/db/...  → 无输出
```

## 源码断言结果（全部命中）

| 断言 | 期望 | 实测 |
|------|------|------|
| `Migrate215CreateADPushQueue(db)` in user_service_test.go | 1 | 1 |
| `CREATE TABLE IF NOT EXISTS sys_ad_config` in user_service_test.go | 1（含 deleted_at） | 1 |
| `Migrate215CreateADPushQueue(db)` in user_handler_test.go | 1 | 1 |
| `CREATE TABLE IF NOT EXISTS sys_ad_config` in user_handler_test.go | 1（含 deleted_at） | 1 |
| `oldUserSnapshot := user` in user_service.go | 1 | 1 |
| `EnqueueUpdateIfChanged(tx` in user_service.go | 1 | 1 |
| `EnqueueDisable(tx` in user_service.go | 2 | 2 |
| `go func` in user_service.go | 0 | 0 |
| `UpdateStatus` diff 行数 | 0 | 0 |
| `git diff --stat internal/api/v1/system/user_handler.go` | 空 | 空 |
| `func TestPhase57_UserService_` | ≥6 | 6 |

## Files Created/Modified

- `internal/services/system/user_service.go`（改）— 1 条 import + 1 行快照 + 3 处入队调用
- `internal/services/system/user_service_test.go`（改）— `setupUserServiceTestDB` 补 `sys_ad_config` + migration_215 + sys_user 补齐全列
- `internal/services/system/user_cache_impl_test.go`（改）— 第三套 sqlite 环境同构补表
- `internal/api/v1/system/user_handler_test.go`（改）— 第二套 sqlite 环境（handler 层）同构补表
- `internal/services/system/ad_push_queue_test.go`（改）— `seedADConfigEnabled` 去建表、追加 6 个 Wave 3 用例、加 `context` import（既有 14 例原样保留）

## Decisions Made

- **`oldUserSnapshot` 用值拷贝而非深拷贝**：`models.User` 无 slice/map 字段，指针字段由 GORM 重新赋值而非原地改写，浅拷贝足以隔离事务闭包内的覆盖；深拷贝只会徒增噪音
- **失败一律 `apperrors.DatabaseError(err)` 上抛**：禁 "捕获 + WARN + return nil" 降级（D-02 第二层 / T-57-07）；T-57-03 的直接证据是 `TestPhase57_UserService_Update_RollsBackWhenQueueInsertFails`
- **不扩展 `UpdateStatus`**（D-04 Deferred）：双向推送的状态变更排除在 v1.21 之外，避免 scope 膨胀
- **不动 handler 生产文件**（D-03）：`internal/api/v1/system/user_handler.go` 的 `adSyncInFlight` / `buildADSyncMap` / goroutine 三次重试保留，与 Phase 58 排水器同二进制删除

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `setupUserServiceTestDB` 的 sys_user DDL 补齐 models.User 全部列**
- **Found during:** Task 2
- **Issue:** `Update` 走 `tx.Save(&user)`，GORM 会写满 `models.User` 的全部列（`created_by` / `version` / `employee_no` / `ad_dn` …），而该 fixture 的 DDL 只有 15 列 → `SQL logic error: no such column: created_by (1)`。此前该 fixture 只能跑 `Update_NotFound`，从未跑通过成功路径。更隐蔽的后果：`RollsBackWhenQueueInsertFails` 会因这个无关错误而"假绿"
- **Fix:** DDL 补齐 15 个缺失列（纯增量，不影响既有 20 个 `TestUserService_*` 用例）
- **Files modified:** internal/services/system/user_service_test.go
- **Verification:** 三个 Update 接线用例全绿，回滚用例的 `SELECT nickname` 返回 NULL
- **Committed in:** `a69ecd9c`

**2. [Rule 3 - Blocking] 计划外的第三套 sqlite 环境 `setupUserCacheTestDB` 同步补表**
- **Found during:** Task 3
- **Issue:** 计划只识别出两套 sqlite 环境；实际 `internal/services/system/user_cache_impl_test.go` 的 `userCacheService` 内嵌 `userService`，复用同一套写路径，缺表时 `TestUserCache_Delete_TriggersInvalidate` / `TestUserCache_BatchDelete_TriggersInvalidate` 直接红（`no such table: sys_ad_config`）
- **Fix:** `setupUserCacheTestDB` 同构补 `sys_ad_config` + 直调 `Migrate215CreateADPushQueue`，加 Phase 57 注释标注来由
- **Files modified:** internal/services/system/user_cache_impl_test.go
- **Verification:** `go test ./...` exit 0
- **Committed in:** `b14e1fad`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** 均为测试 fixture 层的必需适配（HYGIENE-06 的真实覆盖面比计划估计的更广），零生产代码 scope 膨胀，`user_handler.go` 与 `UpdateStatus` 仍零改动。

## Issues Encountered

- 执行过程中一次 `go test ./...` 捕获到 3 行 `FAIL` 输出，但随后的两轮全量运行（以及针对 `internal/services/system`、`internal/api/v1/system`、`internal/services/addomain` 的连续 3 轮）均 exit 0 且无 FAIL，未能复现，判定为环境偶发（非本次改动引入：三套 fixture 均为各自独立的 `:memory:` 库）。全量套件最终以两轮 exit 0 作为验收证据。

## Known Stubs

无。三处接线点全部有实现且被行为断言直接锁定；无占位分支、无"待补"路径。

## Threat Flags

无。未新增网络端点、鉴权路径、文件访问或 schema 变更；`sys_ad_config` 只读、`ad_push_queue` 只写，均在既有事务边界内。T-57-01/02/03/07/10 按计划缓解，T-57-09 按计划 accept（operlog 仍由 handler 层既有 `operlog.Record` 承担，service 层未重复记录）。

## Next Phase Readiness

给 Phase 58（排水器）：

- **队列表字段集已定且不再变更**：`entity_type='user'` / `entity_id` / `action_type`（`update_user` | `disable`）/ `status`（pending|failed|pushed|dead）/ `attempts` / `next_retry_at` / `last_error` / `created_at` / `updated_at` / `pushed_at` / `failed_at`。无 payload 快照 —— 排水闭包必须重读 DB
- **合并语义**：同一 (entity_type, entity_id, action_type) 在 pending/failed 上恒一行（`uq_ad_push_queue_active` 夺锁）。合并时 `attempts`/`last_error`/`next_retry_at`/`failed_at` 已被清零、`created_at`/`updated_at` 刷新为最近一次编辑时间 → 排水器**不能**把 `created_at` 当首次入队时间排序推断业务时序
- **disable 行的执行原语**：`internal/services/addomain/ldap_client.go:338 DisableUser`（UAC 514 Replace，幂等可重放）。禁改为 delete AD 账号（D-04）
- **`ErrADTargetNotFound` → 直接转终态**，防"清 DN → 重推"死循环
- **delete 路径仍不直接触发 LDAP**：`Delete` / `BatchDelete` 现在只写队列行，Phase 58 排水器是唯一执行侧；在此之前删除用户不会碰 AD
- **可删的旧代码**：Phase 58 排水器落地后，`internal/api/v1/system/user_handler.go` 的 `adSyncInFlight` / `buildADSyncMap` / goroutine 三次重试路径可同二进制删除（本 plan 未动，红线守住）
- **`UpdateStatus` 仍不入队**（D-04 Deferred）：若 Phase 58 之后需要状态双向推送，另开 change

## Self-Check: PASSED

- 5 个文件全部存在且已提交（`git log --oneline -4`：`ae345d46` / `a69ecd9c` / `b14e1fad`）
- `go build ./...` exit 0；`go vet` 无输出；`go test ./...` 两轮 exit 0
- `TestPhase57_` 20 例全绿（既有 14 例零回归 + 新增 6 例）
- `git diff 8246173a..HEAD -- internal/api/v1/system/user_handler.go` 为空（D-03 红线）

---
*Phase: 57-ad-outbox-lite*
*Completed: 2026-10-16*
