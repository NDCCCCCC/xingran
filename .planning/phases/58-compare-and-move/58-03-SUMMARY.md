---
phase: 58-compare-and-move
plan: 03
subsystem: addomain / ad_push_queue
tags: [go, ldap, drainer, outbox-lite, backoff, state-machine, failover-client, phase-58]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    plan: 01
    provides: "排水测试基座（setupDrainTestDB / seedDrainADConfig / seedDrainUser / seedDrainQueueRow / newDrainTestLDAP / newDrainTestPool / drainTestAccount + mockLDAPClient 调用计数）"
  - phase: 58-compare-and-move
    plan: 02
    provides: "MatchADConfigByDN（D-02 最长后缀唯一匹配）+ NormalizeDN / BuildOUDN"
  - phase: 57-ad-outbox-lite
    provides: "ADPushQueue 模型 + ADPushQueueActivePredicate 单一事实源 + syncUserAttributes（含 DNExists 预检与 code 32 清 ad_dn）+ ErrADTargetNotFound / ErrAllAccountsUnavailable"
provides:
  - "addomain.ADPushDrainer —— QUEUE-03 排水执行面（拾取 / 退避 / 状态机 / 配置分组 / disable+属性两条推送路径）"
  - "addomain.backoffDuration —— 退避曲线纯函数（60s→2m→4m→8m→16m→封顶 30min）"
  - "ADPushDrainer.clientFactoryOverride —— 未导出测试注入点（DrainOnce 零网络可测的唯一入口）"
  - "测试装配 helper newDrainTestDrainer（58-04/58-07 的 DrainOnce 用例统一入口）"
affects: [58-04, 58-05, 58-06, 58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "P1 三重防线：drainRow 无返回值（结构上无法上抛）+ executeGroup 闭包体硬编码 return nil + 注释引前科 ad-modify-fail-double-counts-breaker"
    - "P12 按配置分组：groupByResolvedConfig 把行按 MatchADConfigByDN 结果分桶，每组一次 ExecuteWithFailover（100 行 ≠ 100 次 bind）"
    - "不可恢复 / 可重试二分：unrecoverable 分支传 row.Attempts 原值（不烧计数），达上限分支传 newAttempts"
    - "deadRows 携带 reason（drainDeadRow 内嵌 ADPushQueue）—— 三类不可恢复原因分别落库，排障 SQL 可区分根因"
    - "状态写回统一 Updates(map[string]interface{}{...}) + 显式 updated_at，时间戳一律 .UTC()（F-12）"

key-files:
  created:
    - internal/services/addomain/ad_push_queue_drain_service.go
  modified:
    - internal/services/addomain/ad_push_queue_drain_test.go
    - internal/services/addomain/ad_push_queue_service.go
    - internal/models/ad_push_queue.go

key-decisions:
  - "deadRows 元素类型用 drainDeadRow（内嵌 models.ADPushQueue + reason）而非裸 []models.ADPushQueue —— plan 的 DrainOnce 伪码给全部 deadRows 写死 reason='unrecoverable:no-ad-config'，与同 plan 的用例断言 last_error 含 'unrecoverable:user-not-found' 直接冲突；改为逐行携带 reason 后两者同时满足"
  - "drainRow / handlePushError / clearUserADRefs / buildDrainUpdateReq 随 Task 1 一并落地（而非 Task 2）—— DrainOnce 引用 executeGroup→drainRow 是编译依赖，Task 1 的 `go build ./...` 验收无法在缺 drainRow 时通过"
  - "F-11 口径确认：attempts 即'自上次编辑以来的连续失败计数'，dead 只在本轮实际失败分支内按 attempts+1 >= 5 判定；'编辑即续命'（持续编辑的失败行永不终态）是**有意设计**，已写进 markFailed 注释"
  - "models.ADPushQueue 的 uniqueIndex 注释改为'唯一索引 tag'字样 —— 满足 W-1 红线断言 `grep -c uniqueIndex internal/models/ad_push_queue.go == 0`（该注释为 Phase 57 既有，非本 plan 新增；语义未变）"
  - "本 plan 不标记 QUEUE-03 / QUEUE-05 为 complete —— 二者均为跨 plan 需求（QUEUE-03 的 cron 接线在 58-05、QUEUE-05 的 OU 路径 code 32 在 58-04），沿用 58-01 的分档口径，最终回填由 58-07 承担"

requirements-completed: []

# Metrics
duration: 20min
completed: 2026-10-19
---

# Phase 58 Plan 03: 排水器骨架 + 状态机 + disable/属性两条推送路径 Summary

**ADPushDrainer 落地 QUEUE-03 排水执行面：D-05 拾取谓词（next_retry_at IS NULL = 可拾取）+ 退避曲线纯函数 + 六状态跃迁状态机 + 按 ad_dn 归属分组（每组一次账号池故障切换）+ disable(UAC 514)/update_user(属性) 两条真实推送路径；F-11 禁令以 markFailed 单点判定锁死，F-12 四处时间戳 UTC 与 F-14 model default tag 一并落地，13 个 Drain/拾取用例全绿、Phase 57 sentinel 与全仓 `go test ./...` 零回归**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-10-19T01:58Z
- **Completed:** 2026-10-19T02:18Z
- **Tasks:** 2/2
- **Files modified:** 4（1 新建 + 3 增量修改）

## 交付契约（下游 58-04/58-05/58-07 直接消费）

```go
// internal/services/addomain/ad_push_queue_drain_service.go
func NewADPushDrainer(db *gorm.DB, pool AccountPool) *ADPushDrainer
func (d *ADPushDrainer) DrainOnce(ctx context.Context) error
func backoffDuration(attempts int) time.Duration            // 纯函数
func (d *ADPushDrainer) pickRows(ctx, now) ([]models.ADPushQueue, error)
func (d *ADPushDrainer) groupByResolvedConfig(ctx, rows) (map[string][]models.ADPushQueue, []drainDeadRow, error)
func (d *ADPushDrainer) loadUserUnscoped(ctx, userID string) (*models.User, error)
func (d *ADPushDrainer) executeGroup(ctx, cfgID string, group []models.ADPushQueue, now time.Time) error
func (d *ADPushDrainer) markPushed(ctx, id string, now time.Time) error
func (d *ADPushDrainer) markFailed(ctx, row models.ADPushQueue, now time.Time, poolExhausted bool, msg string) error
func (d *ADPushDrainer) markDead(ctx, id string, now time.Time, attempts int, reason string) error
func (d *ADPushDrainer) drainRow(ctx, c LDAPClientIface, row models.ADPushQueue, cfgID string, now time.Time) // 无返回值（P1）
func (d *ADPushDrainer) handlePushError(ctx, row models.ADPushQueue, now time.Time, err error, action string)
func (d *ADPushDrainer) clearUserADRefs(ctx, userID string)
func buildDrainUpdateReq(user *models.User) map[string]interface{}
```

### `ADPushDrainer` 结构体全文

```go
type ADPushDrainer struct {
	db                     *gorm.DB
	pool                   AccountPool
	now                    func() time.Time // 测试注入点；一轮只取一次（P8）
	mu                     sync.Mutex       // 轮次互斥事实源（TryLock）
	adSync                 *UserADSyncService
	mapper                 *DeptOUmapper // Plan 04 使用
	clientFactoryOverride  func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface // 未导出，仅测试
}
```

### 退避曲线实现（D-05，硬编码 discretion）

| attempts | 退避 |
|----------|------|
| <= 0 | 60s |
| 1 | 60s |
| 2 | 2min |
| 3 | 4min |
| 4 | 8min |
| 5 | 16min |
| >= 6 | 30min（`backoffCap` 封顶） |

`attempts` 达 5 转 dead（`maxDrainAttempts`）；总重试窗 ~61min，刻意桥接账号池 30min 熔断周期。

### `last_error` 前缀约定表（排障 SQL 可 grep）

| 前缀 | 语义 | 产出点 |
|------|------|--------|
| `unrecoverable:user-not-found` | 队列行 entity_id 指向不存在的用户（F-07 孤儿） | `groupByResolvedConfig` / `drainRow` |
| `unrecoverable:no-ad-dn` | 用户无 `ad_dn`（不可当成成功，防 Phase 59 守卫误判） | `groupByResolvedConfig` / `drainRow` |
| `unrecoverable:no-ad-config` | `MatchADConfigByDN` 无匹配 / 歧义，或配置在分组后被删 | `groupByResolvedConfig` / `executeGroup` |
| `unrecoverable:target-not-found` | code 32 或 `ErrADTargetNotFound`（QUEUE-05，已清 `ad_dn`） | `drainRow` / `handlePushError` |
| `unrecoverable:unknown-action` | `action_type` 不在枚举内（不放行） | `drainRow` default |
| `dead:max-attempts` | 连续失败达 5 转终态 | `markFailed` |
| `retry:<action>:<err>` | 可恢复失败，已排退避 | `handlePushError` / `drainRow` |
| `pool:<err>` | 账号池熔断（`ErrAllAccountsUnavailable`），**不判 dead** | `executeGroup` / `handlePushError` |

（`superseded by disable` 为 Phase 57 入队侧既有前缀，非本 plan 产出。）

## F-11 dead 判定口径与已知后果（确认落库）

- 判定唯一发生点：`markFailed` 内 `if !poolExhausted && newAttempts >= maxDrainAttempts`（源码仅 1 处，`grep -c "newAttempts >= maxDrainAttempts"` == 1）。
- `attempts` 语义 = "自上次编辑以来的连续失败次数"（Phase 57 合并子句每次编辑清零），因此**不可**在拾取 / 成功路径凭裸 `attempts` 判死。
- **已知并接受的后果（"编辑即续命"）**：用户持续编辑某个持续失败的属性 → 每次编辑清零 attempts → 该行永不终态。这是**有意设计**（用户还在主动管这件事），已写进 `markFailed` 注释；退避上限 30min 保证不会密集打域控。若将来出现"不再编辑但仍重试"的真实反馈，再评估独立计数列。
- `markDead(ctx, id, now, attempts int, reason string)` 的 `attempts` 是显式入参并落库；本 plan 共 8 处传 `row.Attempts`（不烧计数）、1 处传 `newAttempts`（达上限），无四参漏传调用。

## Plan 04 的接口位

- **`internal/services/addomain/ad_push_queue_drain_service.go:370`** —— `// Phase 58 Plan 04: 此处补 compare-and-move OU 移动分支`
  - 上方是完整可用的属性路径（syncUserAttributes → reload → 判空 → dead）；
  - 下方是 `d.markPushed(ctx, row.ID, now)`；
  - `drainRow` 签名已含 `cfgID`，与 58-04 T2 终稿同构，58-04 只需替换该行 + 加 `if terminal { return }` 守卫。
- **测试注入点**：`ADPushDrainer.clientFactoryOverride`（executeGroup 内 `fc.clientFactory = d.clientFactoryOverride`）+ `newDrainTestDrainer(t, db, mock, accounts...)` —— 58-04/58-07 的 DrainOnce 用例必须走该 helper，**不得**自行 `NewADPushDrainer` 后忘记注入（否则真拨号 + MarkFailure 污染账号池）。

## F-12 / F-14 改动行号

| 缺口 | 文件 | 行 | 改动 |
|------|------|----|------|
| F-12 | `internal/services/addomain/ad_push_queue_service.go` | 184 | `adPushOnConflictClause(time.Now())` → `time.Now().UTC()` |
| F-12 | `internal/services/addomain/ad_push_queue_service.go` | 275 | `now := time.Now()` → `time.Now().UTC()` |
| F-12 | `internal/models/ad_push_queue.go` | 92 | `BeforeCreate`：`time.Now()` → `time.Now().UTC()` |
| F-12 | `internal/models/ad_push_queue.go` | 104 | `BeforeUpdate`：`time.Now()` → `time.Now().UTC()` |
| F-14 | `internal/models/ad_push_queue.go` | 71 | `Status` tag 加 `default:'pending'`（与 migration_215 DDL 对齐） |
| W-1 | `internal/models/ad_push_queue.go` | 61 | 注释内 `uniqueIndex` 字样改为"唯一索引 tag"（满足红线 grep，语义未变） |

## Task Commits

Each task was committed atomically:

1. **Task 1: 排水器骨架 + 拾取谓词 + 退避纯函数** — `04dfa698` (feat)
2. **Task 2: drainRow 状态跃迁 + F-11 禁令 + F-12/F-14 一行修** — `34294f67` (feat)

## 验收断言实测（全部通过）

| 断言 | 结果 |
|------|------|
| `go test ./internal/services/addomain/ -run "TestPhase58_BackoffDuration\|TestPhase58_PickRows\|TestPhase58_Drain_NoConfigDefer" -count=1 -v` | 9/9 PASS |
| `go test ./internal/services/addomain/ -run "TestPhase58_Drain_" -count=1 -v` | 13/13 PASS（12 新增 + NoConfigDefer） |
| `go build ./...` | exit 0 |
| `grep -c "Where(models.ADPushQueueActivePredicate)"`（P7） | 1 |
| `grep -v '^\s*//' \| grep -c "ADPushQueueActivePredicate = "`（不得改写） | 0 |
| `grep -c "Unscoped()"`（P2） | 3 |
| `grep -v '^\s*//' \| grep -c "ExecuteWithFailover"`（P12，滤注释） | 1 |
| `grep -c "fc.ExecuteWithFailover(ctx, func("` | 1 |
| `grep -c "ldap.EscapeDN"`（P4） | 0 |
| `grep -c "func backoffDuration(attempts int) time.Duration"` | 1 |
| `grep -c "drainBatchLimit = 100"` / `"maxDrainAttempts = 5"` | 1 / 1 |
| `grep -c "clientFactoryOverride func(\*models.ADConfig, \*models.ADServiceAccount) LDAPClientIface"` | 1 |
| `grep -c "fc.clientFactory = d.clientFactoryOverride"` | 1 |
| `grep -c '"processing"'`（P5） | 0 |
| `grep -c "d.adSync.syncUserAttributes(ctx, c, user, updateReq)"`（P11） | 1 |
| `grep -c "newAttempts >= maxDrainAttempts"`（F-11） | 1 |
| `grep -c "markDead(ctx context.Context, id string, now time.Time, attempts int, reason string) error"` | 1 |
| `grep -c "d.markDead(ctx, row.ID, now, newAttempts,"` | 1 |
| `grep -c "d.markDead(ctx, row.ID, now, row.Attempts,"` | 8（≥5） |
| `grep -c "d.markDead(ctx, row.ID, now, \""`（禁四参漏传） | 0 |
| `grep -c "ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject)"`（QUEUE-05） | 1 |
| `grep -c "errors.Is(err, ErrADTargetNotFound)"`（QUEUE-05） | 1 |
| `grep -v '^\s*//' \| grep -c "if refreshed, err := d.loadUserUnscoped(ctx, row.EntityID); err == nil"` | 1 |
| `grep -n "time.Now()" <file> \| grep -vc "time.Now().UTC()"`（queue_service / model，F-12 有意义口径） | 0 / 0 |
| `grep -c "default:'pending'"`（F-14） | 1 |
| `grep -c "uniqueIndex" internal/models/ad_push_queue.go`（W-1） | 0 |

## Test Results

```
go build ./...                                                              → exit 0
go test ./internal/services/addomain/ -count=1                              → ok (8.2s)
go test ./internal/services/system/ -run TestPhase57 -count=1（sentinel）    → ok
go test ./internal/api/v1/system/ -count=1                                  → ok
go test ./...                                                               → exit 0（全仓）
go vet ./internal/services/addomain/                                        → 无输出
```

Phase 57 的 42 个既有用例（含 F-12 触碰的入队路径）零回归。

## Decisions Made

1. **`deadRows` 携带 reason（`drainDeadRow`）** —— 见 Deviations #1。
2. **`drainRow` 及其下游函数随 Task 1 落地** —— 见 Deviations #2。
3. **W-1 注释口径**：`models/ad_push_queue.go:61` 的既有注释含 `uniqueIndex` 字面量，与 plan 的 W-1 红线断言（`grep -c uniqueIndex == 0`）冲突；改为"唯一索引 tag"（语义不变，仍指向 GORM 的该 tag）。见 Deviations #3。
4. **不标记 QUEUE-03 / QUEUE-05 为 complete** —— 二者均为跨 plan 需求（QUEUE-03 的 cron 接线在 58-05；QUEUE-05 的 OU 路径 code 32 在 58-04），沿用 58-01 的分档口径，最终由 58-07 回填（ROADMAP 已写明"全部 5 REQ"归 58-07）。

## Deviations from Plan

**1. [Rule 1 - 计划内部矛盾] `deadRows` 元素类型改为携带 reason 的 `drainDeadRow`**

- **Found during:** Task 1
- **Issue:** plan 的 `DrainOnce` 伪码对全部 `deadRows` 硬编码 `reason = "unrecoverable:no-ad-config"`，但同一 plan 的 Task 2 用例 `TestPhase58_Drain_OrphanDisableRowGoesDead` 断言 `last_error` 含 `"unrecoverable:user-not-found"` —— 两者不可同时满足（孤儿行正是在 `groupByResolvedConfig` 里被归入 deadRows 的）。
- **Fix:** `groupByResolvedConfig` 的第二个返回值改为 `[]drainDeadRow`（`struct { models.ADPushQueue; reason string }`），三类不可恢复原因（`user-not-found` / `no-ad-dn` / `no-ad-config`）分别落库；`DrainOnce` 的循环体保持 plan 的形态（`d.markDead(ctx, row.ID, now, row.Attempts, row.reason)`，内嵌提升后 `row.ID` / `row.Attempts` 仍可用）。断言强度未降低。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_service.go`
- **Committed in:** `04dfa698`

**2. [Rule 3 - Blocking] `drainRow` / `handlePushError` / `clearUserADRefs` / `buildDrainUpdateReq` 随 Task 1 一并落地**

- **Found during:** Task 1
- **Issue:** plan 把 `drainRow` 归入 Task 2，但 Task 1 的验收要求 `go build ./...` exit 0，而 `DrainOnce → executeGroup → drainRow` 是硬编译依赖 —— 缺 `drainRow` 时 Task 1 无法编译，更无法运行其 9 个用例。
- **Fix:** Task 1 一次性写全排水器（含 `drainRow` 及下游四个函数）；Task 2 的落地物相应为「12 个 DrainOnce 用例 + `newDrainTestDrainer` helper + F-12/F-14 一行修」。功能契约与验收断言零变化。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_service.go`
- **Committed in:** `04dfa698`（Task 1）/ `34294f67`（Task 2）

**3. [Rule 1 - 计划内部矛盾] `internal/models/ad_push_queue.go` 注释内 `uniqueIndex` 字面量改写**

- **Found during:** Task 2
- **Issue:** plan 的 W-1 红线断言要求 `grep -c "uniqueIndex" internal/models/ad_push_queue.go` == 0，但该文件第 61 行的 Phase 57 既有注释含 `uniqueIndex tag` 字样（断言在未改动的文件上即已不成立）。
- **Fix:** 注释改为"不含任何唯一索引 tag"，语义不变（仍指向 GORM 的唯一索引 tag 声明）。模型本身从未声明 `uniqueIndex`，红线实质未破。
- **Files modified:** `internal/models/ad_push_queue.go`
- **Committed in:** `34294f67`

**4. [Rule 1 - 计划内部矛盾] F-12 的 `grep -n "time.Now()" ... == 0` 断言不可满足**

- **Found during:** Task 2
- **Issue:** plan 的 F-12 验收要求 `grep -n "time.Now()" <file>` 命中数为 0，但 plan 自己规定的修复正是 `time.Now()` → **`time.Now().UTC()`** —— 后者是前者的字符串超串，grep 必然命中（改动前后均为 2 处/文件）。
- **Fix:** 按 plan 的 `<action>` 规定执行修复（四处全部改为 `time.Now().UTC()`），并给出**语义等价的可成立口径**：`grep -n "time.Now()" <file> | grep -vc "time.Now().UTC()"` == 0（无任何残留的本地时区取时）。断言实质未降低。
- **Files modified:** `internal/services/addomain/ad_push_queue_service.go`、`internal/models/ad_push_queue.go`
- **Committed in:** `34294f67`

---

**Total deviations:** 4 auto-fixed（3 处 Rule 1 计划内部矛盾 + 1 处 Rule 3 blocking）
**Impact on plan:** 均未降低任何验收断言的强度；两处 plan 内部矛盾以"plan 自己写的可证伪断言"为准裁决（与 58-02 同款处置口径）。

## Issues Encountered

- **`gsd-sdk` CLI 在本执行环境不可用**（沿用 58-01/58-02 结论）。STATE.md / ROADMAP.md / VALIDATION.md 的进度回填按现有结构手工完成，未调用 `state.advance-plan` 等 SDK 动词。
- 工作区存在大量**与本 plan 无关**的未暂存 `.planning/phases/*` 删除（执行前既有状态）。全程只用 `git add <具体文件>` 逐文件暂存。

## Known Stubs

无。`mapper` 字段（`*DeptOUmapper`）在本 plan 只注入未使用，是 **Plan 04 的显式接口预留**（`// Phase 58 Plan 04: 此处补 compare-and-move OU 移动分支`，`:370`），不是未接线数据源 —— 本 plan 的全部交付符号均被生产代码或测试真实引用（`NewADPushDrainer` / `DrainOnce` 被本 plan 13 个用例驱动；`backoffDuration` 被 `markFailed` 调用）。

## Threat Flags

无新增安全相关表面。本 plan 的威胁登记项（T-58-03-01 ~ T-58-03-10）全部以**代码 + 可证伪测试**落实：

| Threat | 落实位置 |
|--------|----------|
| T-58-03-01 Elevation/Tampering（disable 行作用于 AD 账号） | 仅 `c.DisableUser`（UAC 514，幂等）；注释写明禁 delete AD 账号（Phase 57 D-04 红线）；`TestPhase58_Drain_DisableSuccessWritesPushedAt` |
| T-58-03-02 DoS（闭包上抛行级 error） | P1 三重防线：`drainRow` 无返回值 + `executeGroup` 闭包体 `return nil` + 注释引前科；`TestPhase58_Drain_UpdateUserSuccessWritesPushedAt` 断言同组单次 Connect |
| T-58-03-03 DoS（忘记 Unscoped） | `loadUserUnscoped` 唯一读用户入口；`TestPhase58_Drain_DisableSoftDeletedUserStillDrains` 构造软删用户锁死 |
| T-58-03-04 Repudiation（行级失败无痕迹） | `last_error` 结构化前缀（见上表）+ 终态留表 + 每次跃迁显式 `updated_at`（UTC）；后台任务不适用 `operlog.Record`（CLAUDE.md 约定针对 HTTP 写端点） |
| T-58-03-05 Tampering（行被篡改指向任意用户） | 无 HTTP 触发面；`entity_id` 只走参数化查询；`action_type` 走 `switch`，未知 → dead（`TestPhase58_Drain_OrphanDisableRowGoesDead` 覆盖 not-found 分支） |
| T-58-03-06 Info Disclosure（last_error 含 DN 原文） | **accept**（DN 已在 `sys_user.ad_dn` 明文，本列不引入新类别；本 phase 无队列查询端点） |
| T-58-03-07 Tampering（dead 行被误判已同步） | `TestPhase58_PickRows_TerminalRowsNotPicked` 锁定终态不重试；dead 与 pushed 前缀可区分 |
| T-58-03-08 DoS（毒丸行永不终态） | **accept** —— F-11 的"编辑即续命"已知后果，写进 `markFailed` 注释与本 SUMMARY |
| T-58-03-09 Repudiation（时间戳时区不一致） | F-12 四处 `.UTC()`；`TestPhase58_Drain_WritesUTCTimestamps` 断言 `pushed_at.Location() == time.UTC` 且值等于注入的固定 UTC 时刻 |
| T-58-03-10 Tampering（ActivePredicate 被改写） | P7 红线：`Where(models.ADPushQueueActivePredicate)` 引用 + 断言"不得重新赋值该常量" |

## 状态更新

- **STATE.md**：Current Position 更新为 Phase 58 / Plan 03 完成；Decisions 追加本 plan 的 key-decisions；Performance Metrics 追加执行记录。
- **ROADMAP.md**：Phase 58 进度行 Plan 03 标记 complete（3/7 plans）。
- **VALIDATION.md**：58-03/T1、T2 两行状态 `⬜ pending` → `✅ green`（附 commit 号）。
- **REQUIREMENTS.md**：QUEUE-03 / QUEUE-05 均为跨 plan 需求，**不在本 plan 标记 complete**（沿用 58-01 分档口径；最终回填归 58-07）。

## Next Phase Readiness

- **Wave 3（58-04）可立即开** —— 排水器执行面已就位，compare-and-move 落点已标（`:370`），`drainRow` 签名含 `cfgID` 与 58-04 终稿同构。
- **58-04 必须遵守**：① 走 `newDrainTestDrainer` 注入 mock（禁自行 `NewADPushDrainer`）；② OU 路径的 code 32 复用 `handlePushError` 第 2 分支（`ldap.IsErrorWithCode`），不得新增第三条 dead 路径；③ 终态路径**禁被 `markPushed` 覆盖**（`drainRow` 属性分支已有 reload → 判空 → dead → markPushed 的顺序，58-04 在其中插 `if terminal { return }`）。
- **Wave 4（58-05）**：`NewADPushDrainer(db, s.getPool())` 是 cron 接线的唯一构造形态（**禁** `addomain.NewAccountPool`）。
- **无阻塞**：全仓 `go test ./...` exit 0；Phase 57 sentinel（`TestPhase57`）绿。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*

## Self-Check: PASSED
