---
phase: 58-compare-and-move
plan: 06
subsystem: addomain / ad_push_queue / api-v1-system
tags: [go, sc6, d-01, f-07, f-10, phase-58, outbox-lite]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    plan: 05
    provides: "in-process cron 排水器已接生产（AddJob + SkipIfStillRunning）—— 删除直推的前提"
  - phase: 58-compare-and-move
    plan: 03
    provides: "ADPushDrainer.DrainOnce + drainer 行为面（入队后的唯一执行者）"
  - phase: 58-compare-and-move
    plan: 01
    provides: "F-G 已删 TestUserHandler_BuildADSyncMap（否则本 plan 删方法时编译红）"
  - phase: 57-ad-outbox-lite
    provides: "EnqueueDisable 批量模式 + DedupeEntityIDs + adPushOnConflictClause（EnqueueUpdateBatch 的模板）"
provides:
  - "addomain.(*ADPushEnqueuer).EnqueueUpdateBatch(tx, userIDs) error —— D-01 无 diff 批量入队变体"
  - "api/v1/system.(*UserHandler).enqueueImportUsersToAD(usernames) error —— 导入后同步批量入队入口"
  - "user_service.BatchDelete 的 F-07 存在性 Pluck（入队数 == 实际软删数 不变量可断言）"
  - "SC6 终态：两条直推路径零残留（5 条 grep 静态证伪）"
affects: [58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "D-01 批量入队变体：空切片 no-op → DedupeEntityIDs → IsADSyncEnabled 门控 → CreateInBatches(rows, pushQueueBatchSize) + adPushOnConflictClause"
    - "「不做 diff」的入队变体必须以 diff 变体作反向对照用例才可证伪（NoDiffFiltering 用例先跑 diff 变体断言不入队，再跑批量变体断言入队）"
    - "SC6 残留证据由 grep 静态证伪承担时，注释**不得**写出被删符号的字面名（否则注释自指使断言假阳）"
    - "F-07 存在性过滤必须排在软删语句**之前**（GORM 默认软删作用域会让软删之后的 Pluck 恒空）"

key-files:
  created: []
  modified:
    - internal/services/addomain/ad_push_queue_service.go
    - internal/services/addomain/ad_push_queue_drain_test.go
    - internal/services/addomain/user_ad_sync_service.go
    - internal/api/v1/system/user_handler.go
    - internal/api/v1/system/user_import_handler.go
    - internal/services/system/user_service.go
    - internal/services/system/ad_push_queue_test.go

key-decisions:
  - "EnqueueUpdateBatch 只产 update_user 行，绝不产 disable（T-58-06-04：disable 只由 Delete/BatchDelete/UpdateStatus 三条 service 事务路径产生）"
  - "不做 F-01 压制：压制只发生在 disable 打 update_user 上，update 之间无压制关系"
  - "enqueueImportUsersToAD 自开短事务（ExcelService 事务已提交）—— D-01 已接受的事务外例外，代码与 SUMMARY 双处写明，不重构 ExcelService 事务"
  - "Q5：SyncUserUpdateToAD / BatchSyncUsersToAD 保留不删，仅标注「Phase 58 SC6 后已无生产调用方」"
  - "F-07 的 Unscoped() 备选显式禁用：会把历史已软删、本轮未删的 id 算进 existing，破坏「入队数 == 软删数」的可断言性"

requirements-completed: ["QUEUE-03"]

# Metrics
duration: 55min
completed: 2026-10-19
---

# Phase 58 Plan 06: SC6 两条直推路径删除 + F-07/F-10 加固 Summary

**SC6 终态达成 —— 两条遗留直推路径（单用户编辑 goroutine 直推 + 导入后 `triggerADSyncAfterImport`）及其字段/方法零残留，5 条 grep 静态证伪全部 == 0；导入路径改为同步批量入队 `EnqueueUpdateBatch`（D-01，无 diff）；F-07 入队侧 Pluck 掐根 + F-10 两个删除回滚用例落地，Phase 57 sentinel 零回归。**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-10-19T11:40Z（近似）
- **Completed:** 2026-10-19T12:20Z（近似）
- **Tasks:** 3/3
- **Files modified:** 7（0 新建 + 7 增量）

## Task Commits

| # | Task | Commit | 类型 |
|---|------|--------|------|
| 1 | `EnqueueUpdateBatch` 批量入队变体（D-01） | `253461c2` | feat |
| 2 | SC6 删两条直推路径 + 导入换批量入队 + Q5 标注 | `ab8a7231` | feat |
| 3 | F-07 入队侧 Pluck + F-10 两个回滚用例 | `9c2dc88d` | feat |

## 交付契约

### 1. `EnqueueUpdateBatch`（`ad_push_queue_service.go:336-395`）

```go
func (e *ADPushEnqueuer) EnqueueUpdateBatch(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 { return nil }                     // 空切片 no-op
	deduped := DedupeEntityIDs(userIDs)
	if len(deduped) == 0 { return nil }                      // 全空串也 no-op
	enabled, err := IsADSyncEnabled(tx)
	if err != nil { return err }
	if !enabled { applogger.Warnf(...); return nil }          // D-02 第一层：WARN 放行
	now := time.Now().UTC()                                  // F-12 全链路 UTC
	rows := make([]models.ADPushQueue, 0, len(deduped))
	for _, id := range deduped { rows = append(rows, models.ADPushQueue{
		EntityType: models.ADPushEntityUser, EntityID: id,
		ActionType: models.ADPushActionUpdateUser, Status: models.ADPushStatusPending }) }
	if err := tx.Clauses(adPushOnConflictClause(now)).
		CreateInBatches(&rows, pushQueueBatchSize).Error; err != nil {
		applogger.Errorf(...); return err                    // 原样上抛，不吞
	}
	applogger.Infof("[ad_push_queue] 批量 update 入队成功: %d 行", len(rows))
	return nil
}
```

**与 `EnqueueUpdateIfChanged` 的语义差异（注释 + 用例双重锁定）：**

| 维度 | `EnqueueUpdateIfChanged` | `EnqueueUpdateBatch`（本 plan） |
|------|--------------------------|---------------------------------|
| diff | 四字段 diff（nickname/email/phone/deptId），无变更即不入队 | **不做 diff**，全量入队 |
| 入参 | `(tx, oldUser, req)` —— 需要"变更前旧值快照" | `(tx, userIDs)` —— 无快照可用 |
| 典型调用方 | 单用户编辑（service 事务内，能取到 oldUser） | 导入（ExcelService 事务已提交，拿不到旧值） |
| F-01 压制 | 无 | 无（update 之间无压制关系） |

**"不做 diff" 为什么正确**：排水器在闭包内重读 `sys_user` 推最新值，全量入队不产生冗余 LDAP 写 —— 冗余行在 `uq_ad_push_queue_active` 合并子句下会与后续编辑合并为同一行。

### 2. `enqueueImportUsersToAD`（`user_import_handler.go:128-146`）

```go
func (h *UserHandler) enqueueImportUsersToAD(usernames []string) error {
	return h.core.GetDB().Transaction(func(tx *gorm.DB) error {
		var userIDs []string
		if err := tx.WithContext(context.Background()).
			Model(&models.User{}).
			Where("username IN ?", usernames).
			Where("ad_dn IS NOT NULL AND ad_dn <> ''").
			Pluck("id", &userIDs).Error; err != nil { return err }
		if len(userIDs) == 0 {
			applogger.Infof("[AD-SYNC] 导入后无待同步用户（ad_dn 非空），跳过入队")
			return nil
		}
		return addomain.NewADPushEnqueuer().EnqueueUpdateBatch(tx, userIDs)
	})
}
```

- `Pluck` 逻辑从被删的 `triggerADSyncAfterImport` **原样平移**（含 `ad_dn` 非空预过滤 —— 入队侧过滤比排水侧判 dead 更省）。
- **事务外例外（D-01 已接受，非本 plan 可改）**：入队发生在 `ExcelService.ImportData` 的 `Transaction` 提交之后（`operations/excel_service.go:427`），这是 outbox-lite「同生共死」不变量的**一处例外**。进程崩溃窗口会丢推送，最坏后果是 AD 侧部门滞后，由 5 分钟全量同步或下次编辑修复。**未重构 ExcelService 事务去关闭该窗口**（CONTEXT 明文记录为用户批准的取舍）。
- 降级：入队失败仅 `applogger.Errorf`，不影响导入结果。

### 3. SC6 删除范围

| 位置 | 删除内容 | 保留 |
|------|----------|------|
| `user_handler.go:30-32` | `adSyncInFlight sync.Map` 字段 + 2 行注释 | — |
| `user_handler.go:189-238`（原） | `Update` 内整块 goroutine 直推（in-flight 去重 + 3 次重试 + 指数退避 + code 32 短路 break） | `operlog.Record` 位置与相对顺序不变 |
| `user_handler.go:244-260`（原） | 四字段 map 构造方法 | — |
| `user_import_handler.go:104-148`（原） | `triggerADSyncAfterImport`（goroutine + `BatchSyncUsersToAD` 调用） | — |
| imports | `user_handler.go` 的 `errors` / `time` / `sync` 变未使用已删 | `context`（`resolveResetPassword` 用）、`addomain`（字段类型用）逐个 `go build` 确认后保留 |

**`userADSyncService` 字段保留**（Q5 / 硬约束 7）：grep 确认 `SyncManagers` 仍在用 —— `user_import_handler.go:226/231`（`SyncManagersToAD`）。`NewUserHandler` 可变参构造与 `user_router.go:38` 注入不动。`user_import_handler_test.go` 的 `userADSyncService=nil` 语义断言全部保留、全部绿。

### 4. F-07（`user_service.go:657/662/667`）

```
657:  tx.Model(&models.User{}).Where("id IN ?", ids).Pluck("id", &existing)   ← Pluck（软删**之前**）
662:  tx.Where("id IN ?", ids).Delete(&models.User{})                        ← 软删
667:  addomain.NewADPushEnqueuer().EnqueueDisable(tx, existing)              ← 入队 existing（非 ids）
```

**不变量**：入队 disable 行数 == 本轮实际软删的用户数 == `len(existing)`。
**位置铁律**：`Pluck` 排在软删之后会被 GORM 默认软删作用域（`deleted_at IS NULL`）过滤成空 → 入队变 no-op → Phase 57 `TestPhase57_UserService_BatchDelete_EnqueuesDisable` 必红。
**`Unscoped()` 备选显式禁用**：会把历史已软删、本轮未删的 id 算进 `existing`，破坏可断言性。
`Delete`（单条）**不动** —— 事务外 `First` 已保证 id 存在，且 CONTEXT D-04 只锁 BatchDelete。

### 5. Q5 保留的两个方法

`SyncUserUpdateToAD` / `BatchSyncUsersToAD` 保留（删除会牵连 `manager_sync_test.go` 等既有用例），函数注释末尾各追加「Phase 58 SC6：本方法已无生产调用方（handler 直推已删），保留作显式管理面 / 未来复用」。`BatchSyncUsersToAD` 处额外说明：排水器的 OU 移动走 `compareAndMoveOU`（闭包内重读 DB 比较），而非依赖"导入前旧值快照"——「为什么不在这里移动 OU」的结论对排水器同样成立。真实删除留给 Phase 62 死代码清理。

## SC6 五条 grep 静态证伪（**实测输出**）

```bash
$ grep -c "buildADSyncMap"        internal/api/v1/system/user_handler.go        # 期望 0
0
$ grep -c "adSyncInFlight"        internal/api/v1/system/user_handler.go        # 期望 0
0
$ grep -c "SyncUserUpdateToAD"    internal/api/v1/system/user_handler.go        # 期望 0
0
$ grep -c "triggerADSyncAfterImport" internal/api/v1/system/user_import_handler.go  # 期望 0
0
$ grep -c "BatchSyncUsersToAD"    internal/api/v1/system/user_import_handler.go  # 期望 0
0
```

**保留断言**：`grep -c "userADSyncService \*addomain.UserADSyncService" user_handler.go` == **1**。
**顺序断言**（入队在 operlog 之前、operlog 在 response.Success 之前，未变）：

```bash
$ grep -n "enqueueImportUsersToAD\|operlog.Record\|response.Success" internal/api/v1/system/user_import_handler.go
91:		if err := h.enqueueImportUsersToAD(result.AffectedKeys); err != nil {
97:	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeImport,
101:	response.Success(c, gin.H{
117:// enqueueImportUsersToAD 导入后批量入队（D-01，替代已删除的导入后直推方法）。
125:func (h *UserHandler) enqueueImportUsersToAD(usernames []string) error {
182:	operlog.Record(... operlog.OperTypeDownload)
236:	operlog.Record(... operlog.OperTypeSync,
240:	response.Success(c, gin.H{
```

## 验收断言实测

### Task 1

| 断言 | 实测 |
|------|------|
| `go build ./...` | exit 0 |
| `TestPhase58_EnqueueUpdateBatch`（5 用例） | **5/5 PASS** |
| `grep -c "func (e \*ADPushEnqueuer) EnqueueUpdateBatch(tx \*gorm.DB, userIDs \[\]string) error"` | **1** |
| `grep -c "CreateInBatches(&rows, pushQueueBatchSize)"` | **2**（Disable + 新变体，共用同一常量） |
| `grep -A20 "func ...EnqueueUpdateBatch" \| grep -c "TargetWhere"` | **0**（复用 `adPushOnConflictClause`，未自造谓词） |
| `TestPhase58_EnqueueUpdateBatch_NoDiffFiltering` | PASS（反向对照：diff 变体同值不入队 → 批量变体同值入队） |
| `go test ./internal/services/addomain/ ./internal/services/system/ -run "TestPhase58\|TestPhase57"` | exit 0 |

### Task 2

| 断言 | 实测 |
|------|------|
| `go build ./...` | exit 0 |
| `go vet ./internal/api/v1/system/ ./internal/services/addomain/` | 无输出 |
| `go test ./internal/api/v1/system/ -count=1` | **ok 3.627s** |
| 5 条 SC6 grep | 全 **0**（见上） |
| 保留断言 `userADSyncService \*addomain.UserADSyncService` | **1** |
| `grep -c "func (h \*UserHandler) enqueueImportUsersToAD(usernames \[\]string) error"` | **1** |
| `grep -c "EnqueueUpdateBatch(tx, userIDs)"` | **1** |
| `grep -c "本方法已无生产调用方" user_ad_sync_service.go` | **2** |
| `go test ./internal/services/system/ -run TestPhase57 -count=1`（sentinel） | **ok** |

### Task 3

| 断言 | 实测 |
|------|------|
| `go build ./...` | exit 0 |
| `TestPhase58_UserService`（3 用例） | **3/3 PASS** |
| **`TestPhase57`（sentinel）** | **41 PASS / 0 FAIL**（见 Deviation #3：plan 写的「42」是文档口径差，非回归） |
| `grep -c "EnqueueDisable(tx, existing)"` | **1** |
| `grep -c "EnqueueDisable(tx, ids)"` | **0** |
| `grep -c 'Unscoped().Model(&models.User{}).Where("id IN ?" '` | **0** |
| `grep -c "EnqueueDisable(tx, \[\]string{id})"`（Delete 未动） | **1** |
| **Pluck 行号 < 软删行号** | `Pluck=657  Delete=662 → ORDER_OK` |
| 行为断言（不存在 id 不入队） | `count(disable) == 2`（不是 3），`entityIDs == [u1, u2]` |
| 行为断言（F-10 单条） | `deleted_at IS NULL` + `sys_user_role` 2 行不变 |
| 行为断言（F-10 批量） | 两个 `deleted_at IS NULL` + 关联行数不变 + 整体软删数 == 0 |
| `go test ./internal/services/system/ ./internal/services/addomain/ ./internal/api/v1/system/ -count=1` | 三包全 ok |
| `go vet` 三包 | 无输出 |

## Test Results

```
go build ./...                                                    → exit 0
go test ./internal/api/v1/system/ -count=1                        → ok 3.627s
go test ./internal/services/system/ -run "TestPhase58_UserService" -count=1  → ok 2.600s
go test ./internal/services/system/ -run TestPhase57 -count=1    → ok 3.165s（41 PASS / 0 FAIL，sentinel）
go test ./internal/services/addomain/ -run "TestPhase58_EnqueueUpdateBatch" -count=1 → ok 2.054s（5/5 PASS）
go test ./internal/services/addomain/ -count=1                   → ok 7.4s
go test ./internal/services/system/ -count=1                      → ok 7.7s
go vet ./internal/api/v1/system/ ./internal/services/system/ ./internal/services/addomain/ → 无输出
go test ./...                                                     → 全仓绿（过滤 ok/no-test-files 后零输出）
```

## Decisions Made

1. **`EnqueueUpdateBatch` 只产 `update_user` 行**（T-58-06-04）—— disable 只由 `Delete`/`BatchDelete`/`UpdateStatus` 三条 service 事务路径产生。
2. **不做 F-01 压制** —— 压制只发生在 disable 打 update_user 上，update 之间无压制关系。
3. **事务外例外照实写进代码注释**（硬约束 4）—— 未重构 `ExcelService` 事务去关闭 crash 窗口。
4. **Q5 保留 + 标注** —— 两个 service 方法不删。
5. **F-07 的 `Unscoped()` 备选禁用** —— 会破坏「入队数 == 软删数」的可断言性。
6. **`TestPhase58_EnqueueUpdateBatch_NoDiffFiltering` 以 diff 变体作反向对照** —— 否则"无 diff"断言不可证伪（无条件入队的实现也能变绿）。
7. **本 plan 标记 QUEUE-03 为 complete**（沿用 58-05 分档口径：QUEUE-04/05 的端到端证据由 58-07 回填）。

## Deviations from Plan

**1. [Rule 2 — 注释自指会让 plan 自己的 grep 证伪假阳] 删除注释里的被删符号字面名**

- **Found during:** Task 2 验收 grep
- **Issue:** plan 给的注释模板（「原此处有 goroutine 直推 AD（`adSyncInFlight` 去重 …）+ `buildADSyncMap` 四字段构造，已整块删除）」）把被删符号的字面名写进了注释，使 SC6 的 5 条 grep 证伪（期望值 0）命中 3 条：`buildADSyncMap`=1、`adSyncInFlight`=1、`triggerADSyncAfterImport`=2。**这是 plan 内部的断言自指矛盾**，与 58-05 Deviation #1/#2、58-02~58-04 同性质。
- **Fix:** 改写为不含字面名的表述（「按用户 ID 的 in-flight 去重 + 3 次重试 + code 32 短路 break」「四字段 map 构造方法」「已删除的导入后直推方法」），并在注释里写明**为什么**不写字面名（让后来者不会"好心"补回去）。行为约束零变化。
- **Verification:** 复跑 5 条 grep → 全 **0**。
- **Files modified:** `internal/api/v1/system/user_handler.go`, `internal/api/v1/system/user_import_handler.go`
- **Committed in:** `ab8a7231`

**2. [Rule 3 — 阻塞] `enqueueBatchSeedTwoUsers` 与「无启用配置」用例前提互斥**

- **Found during:** Task 1 写测试时
- **Issue:** plan 的 `TestPhase58_EnqueueUpdateBatch_NoADConfigSkips` 描述同时要求 `seedDrainADConfig(..., syncEnabled=false, status=0)` 与"铺两个用户"，但它参照的 `setupDrainTestDB + seedDrainUser` 组合若复用已有的"铺一条启用配置"helper，就会让 `IsADSyncEnabled` 命中 → 用例前提自相矛盾、恒绿变恒红。
- **Fix:** 拆出 `enqueueBatchSeedTwoUsersNoConfig`（只铺用户，不铺任何 AD 配置行），并在 `seedDrainADConfig(syncEnabled=false)` 处加注释说明"故意 seed 未启用的配置而不是完全不 seed"—— 完全无行时「无启用配置」是因为表空，验的不是"有配置但未启用"这一档。**断言未弱化，反而更精确**。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Committed in:** `253461c2`

**3. [Rule 1 — 文档数字与实测不符] Phase 57 sentinel 实为 41 个顶层用例，不是 plan 写的 42**

- **Found during:** Task 3 sentinel 计数
- **Issue:** plan 与 CONTEXT D-04 均写「42 个既有用例」。实测 `go test ./internal/services/system/ -run TestPhase57 -count=1 -v | grep -c "^--- PASS"` == **41**。为排除是本 plan 删掉了一个用例，把 `internal/` 回退到 pre-plan 的 `d41e16cb` 重跑 → 同样 **41**。
- **Fix:** 不改代码、不改断言；按实测口径记录。42 很可能是把某个子用例也算进了计数。
- **Files modified:** 无
- **Committed in:** 无（仅文档口径修正）

**4. [Rule 2 — 追加反向对照] `NoDiffFiltering` 用例增加 diff 变体对照组**

- **Found during:** Task 1 用例设计
- **Issue:** plan 描述的该用例（"用户 nickname 改成完全相同的值 → 仍入队 1 行"）**单独不可证伪** —— 一个"无条件对所有 id 入队"的错误实现同样能让它变绿。
- **Fix:** 用例改为先对同一组"同值"参数调用 `EnqueueUpdateIfChanged` 并断言**不入队**，再调用 `EnqueueUpdateBatch` 并断言**入队 2 行**。断言强度**提高**，未弱化。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Committed in:** `253461c2`

**5. [Rule 2 — 断言可证伪化] `MergeOnSecondCall` 增加 failed 行合并判别**

- **Found during:** Task 1 用例设计
- **Issue:** plan 描述的该用例（"连续两次同 ids 入队 → count 恒为 2"）同样**不可证伪** —— 一个"若已存在则跳过"的幂等实现也能让 count 恒为 2。
- **Fix:** 追加第三步：把两行手工打回 `failed`（attempts=3）后再入队，断言行数仍为 2 **且** `status` 被合并子句打回 `pending`、`attempts` 归零。这才是"合并"而非"幂等跳过"的判别式。断言强度提高。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Committed in:** `253461c2`

---

**Total deviations:** 5 auto-fixed（2 × Rule 2 注释自指、1 × Rule 3 测试前提互斥、1 × Rule 1 文档计数、2 × Rule 2 断言可证伪化）
**Impact on plan:** 无一降低验收强度；两处**提高**（#4/#5 把不可证伪的用例变成可证伪）。**未弱化任何断言，未引入特性开关，未触碰四个命名文件之外的生产代码。**

## Issues Encountered

- **`gsd-sdk` CLI 在本执行环境不可用**（沿用 58-01~58-05 结论，`NO_GSD_SDK`）。STATE.md / ROADMAP.md / VALIDATION.md / REQUIREMENTS.md 的回填按现有结构手工 Edit 完成，未调用 `state.advance-plan` 等 SDK 动词。
- **工作区存在 688 个与本 plan 无关的未暂存 `.planning/phases/*` 删除**（执行前既有状态，58-05 SUMMARY 已记录）。中途为对比 Phase 57 基线误用 `git stash push -u` 暂存了这批删除，已 `git stash pop` **完整还原**（688 个删除全部回到工作区，用户原有的 `pre-merge-2` stash 保持在 `stash@{0}` 未受影响）。全程只用 `git add <具体文件>` 逐文件暂存，从未 `git add .`。
- **F-10 两个回滚用例的失败点落在 F-01 压制 UPDATE 而非队列表 INSERT** —— `EnqueueDisable` 的第一步是对 `ad_push_queue` 的 `UPDATE ... status='dead'`（F-01 压制），队列表不存在时该语句先炸并原样返回 error。回滚语义完全一致（`deleted_at` 保持 NULL、关联行不变），断言有效，只是错误来源比 plan 预期的 INSERT 早一步。

## Known Stubs

无。本 plan 交付的全部符号均被生产代码或测试真实引用：

| 符号 | 引用点 |
|------|--------|
| `EnqueueUpdateBatch` | `user_import_handler.go:141`（生产）+ 5 个测试用例 |
| `enqueueImportUsersToAD` | `user_import_handler.go:91`（生产调用点） |
| `userADSyncService` 字段 | `user_import_handler.go:226/231`（`SyncManagers`）+ `NewUserHandler` 可变参 |
| `SyncUserUpdateToAD` / `BatchSyncUsersToAD` | **无生产调用方**（Q5 已知项，注释已标注；仍被 `manager_sync_test.go` 等既有用例覆盖） |

⚠ **预期内但须下游知悉**：`SyncUserUpdateToAD` / `BatchSyncUsersToAD` 现为测试专用符号。58-07 写 SC6 端到端证据时**不得**把它们当作推送路径的证据（推送路径的唯一执行者是 `ADPushDrainer.DrainOnce`）。

## Threat Flags

本 plan 的威胁登记项（T-58-06-01 ~ T-58-06-08）全部以代码 + 可证伪断言落实：

| Threat | 落实位置 |
|--------|----------|
| T-58-06-01 Tampering（直推与排水器并存乱序） | SC6：两条直推全删，5 条 grep 静态证伪全 0；**未加任何互斥开关/特性开关**（ROADMAP 明文拒绝的路线） |
| T-58-06-02 Repudiation（入队失败无痕迹） | `applogger.Errorf("[AD-SYNC] 导入后批量入队失败（不影响导入结果）: %v")` + `applogger.Infof("批量 update 入队成功: %d 行")` |
| T-58-06-03 Tampering（username 越权入队） | `Pluck` 只取 `ad_dn` 非空的 `id`；`username IN ?` 参数化；行载荷是系统内 UUID |
| T-58-06-04 EoP（导入路径触发 disable） | `EnqueueUpdateBatch` 只产 `action_type='update_user'` |
| T-58-06-05 Repudiation（事务外入队打折） | **accept** —— 代码注释 + SUMMARY 双处写明；无 ad_dn 的行入队前过滤，不会产生必然 dead 的行 |
| T-58-06-06 DoS（重复/无效 id 刷孤儿行） | F-07：`Pluck` 前置 + `EnqueueDisable(tx, existing)`；`TestPhase58_UserService_BatchDelete_EnqueuesOnlyExisting`（含重复 id 子用例） |
| T-58-06-07 Tampering（队列表不可写仍半成功） | F-10 两个回滚用例（`deleted_at IS NULL` + `sys_user_role` 行数不变） |
| T-58-06-08 Repudiation（死代码残留） | Q5 保留 + 两处显式标注（`grep -c "本方法已无生产调用方"` == 2） |

## Next Phase Readiness

- **Wave 6（58-07）可立即开** —— SC6 终态已落地，AD 推送的唯一通路是 `ad_push_queue` → `ADPushDrainer.DrainOnce`。
- **58-07 必须遵守**：
  1. SC6 的端到端证据必须走 `DrainOnce`，**不得**引用 `SyncUserUpdateToAD` / `BatchSyncUsersToAD`（Q5 保留的死代码，不是推送路径）。
  2. SC6 的 5 条 grep 断言沿用本 SUMMARY 的**实测输出表格**口径，且新增注释时**不得**写出被删符号字面名（否则 Deviation #1 复发）。
  3. `TestPhase57` sentinel 的实测口径是 **41 个顶层用例**（不是 plan 写的 42），58-07 若按 42 核对会误判。
  4. F-07 的 `Pluck` 位置铁律（`user_service.go:657 < :662`）在 58-07 的 source-assert 中应直接复用行号比较，且 `Unscoped()` 断言继续用 plan 的**代码形态** pattern（注释里的字面 `Unscoped()` 命中属预期，不构成违规）。
- **无阻塞**：全仓 `go test ./...` 全绿；三包 `go vet` 无输出；Phase 57 sentinel 41 PASS / 0 FAIL。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*

## Self-Check: PASSED
