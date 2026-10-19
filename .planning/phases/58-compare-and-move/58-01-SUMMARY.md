---
phase: 58-compare-and-move
plan: 01
subsystem: testing
tags: [go, sqlite, gorm, ldap, ad-push-queue, fixture, test-harness, phase-57]

# Dependency graph
requires:
  - phase: 57-ad-outbox-lite
    provides: "ad_push_queue 模型 + Migrate215CreateADPushQueue 迁移（部分唯一索引 uq_ad_push_queue_active）+ mockLDAPClient + fakePool78 + ADPushQueueActivePredicate 单一事实源"
provides:
  - "internal/services/addomain 包的排水测试基座：setupDrainTestDB 五表 fixture + seedDrainADConfig/seedDrainUser/seedDrainDept/seedDrainQueueRow + newDrainTestLDAP/newDrainTestPool/drainTestAccount 装配 helper"
  - "mockLDAPClient 调用计数：moveUserCalls / moveUserTargets / disableUserCalls / updateUserAttrCalls"
  - "Phase 57 遗留缺口 F-F / F-G / F-15 / F-16 全部处置完毕"
affects: [58-02, 58-03, 58-04, 58-05, 58-06, 58-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "排水基座 fixture 复用既有包级 const（createUserTable / createDeptOUMappingTable / createDeptTable）—— 零副本，防两套 fixture 漂移"
    - "队列表一律直调 migrations.Migrate215CreateADPushQueue 建表，索引不手写（单一 DDL 事实源）"
    - "fixture 时间列一律 DATETIME（不写 timestamptz），否则 glebarez/go-sqlite 扫不回 time.Time"
    - "mock 默认值防御：newDrainTestLDAP 固定 dnExistsRes=true，防 code 32 用例假绿（T-58-W0-02）"

key-files:
  created:
    - internal/services/addomain/ad_push_queue_drain_test.go
  modified:
    - internal/services/addomain/ldap_client_mock_test.go
    - internal/services/system/user_service_test.go
    - internal/services/system/ad_push_queue_test.go
    - internal/services/system/user_cache_impl_test.go
    - internal/api/v1/system/user_handler_test.go

key-decisions:
  - "基座文件携带 !skip_db_tests build tag —— 因引用同包 fakePool78（定义于带该 tag 的 failover_client_78_07_test.go），不加 tag 会让 go vet -tags skip_db_tests 报 undefined"
  - "F-G 删除 TestUserHandler_BuildADSyncMap 时保留一条 tombstone 注释说明删除理由与覆盖损失为零的依据，但注释内不出现被删函数名（满足 grep -c buildADSyncMap == 0 的验收断言）"
  - "sys_dept fixture 复用 createDeptTable 的 10 列（仅含 dept_name 等），不扩 Leader/Phone/Email —— syncUserAttributes 读部门的唯一动作是 First(&dept) 后取 DeptName，GORM SELECT * 对缺失模型列只是零值不报错"

patterns-established:
  - "公开 seed helper 签名（后续 5 个 plan 直接消费）：seedDrainADConfig(t, db, id, configName, baseDN, syncEnabled, status) / seedDrainUser(t, db, id, username, adDn, adOuDn, deptID *string) *models.User / seedDrainDept(t, db, id, name) / seedDrainQueueRow(t, db, id, entityID, actionType, status, attempts, nextRetryAt *time.Time) models.ADPushQueue"
  - "本 phase 追加用例一律命名 TestPhase58_*，便于 go test ./internal/services/addomain/ -run TestPhase58 -count=1 一把捞起"

requirements-completed: [QUEUE-03, QUEUE-04, QUEUE-05]

# Metrics
duration: 10min
completed: 2026-10-19
---

# Phase 58 Plan 01: 排水测试基座 + Phase 57 fixture 缺口修补 Summary

**addomain 包五表排水测试基座（ad_push_queue 走迁移、其余三表复用既有 const 零副本）+ mockLDAPClient 调用计数 + F-F/F-G/F-15/F-16 四项 Phase 57 遗留缺口处置，为 Wave 1~6 的 RED 测试提供可判真的环境**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-10-19T01:02Z
- **Completed:** 2026-10-19T01:12Z
- **Tasks:** 3/3
- **Files modified:** 6（1 新建 + 5 增量修改）

## Accomplishments

- **排水测试基座落地**（`ad_push_queue_drain_test.go`，282 行）：`setupDrainTestDB(t)` 返回含 `ad_push_queue` / `sys_user` / `sys_ad_config` / `sys_dept` / `sys_dept_ou_mapping` 五张表的 `*gorm.DB`；队列表直调 `migrations.Migrate215CreateADPushQueue(db)`，三个索引（`uq_ad_push_queue_active` / `idx_ad_push_queue_status_retry` / `idx_ad_push_queue_entity_id`）由迁移建，不手写 DDL（T-58-W0-03）。
- **零副本 fixture 复用**：`sys_user` / `sys_dept_ou_mapping` / `sys_dept` 三张表复用 `user_ou_service_test.go` 的既有包级 const（`createUserTable` / `createDeptOUMappingTable` / `createDeptTable`），本文件不含任何 `createDrainDeptTable` 之类的副本。
- **F-F 补列生效**：新建 `createDrainADConfigTable` 含 `base_dn` + `config_name` 两列，并由基座自检用例的 `SELECT count(*) WHERE base_dn = ...` 与 `SELECT config_name` 往返断言锁死。
- **mock 调用计数（compare-and-move 幂等断言的前提）**：`mockLDAPClient` 新增 `moveUserCalls` / `moveUserTargets`（按序记录每次 `newOUDN`）/ `disableUserCalls` / `updateUserAttrCalls`；只新增字段与方法内自增，既有字段/签名零改动。
- **F-G 处置完成**：`TestUserHandler_BuildADSyncMap` 已删除，`grep -c buildADSyncMap internal/api/v1/system/user_handler_test.go` == 0 —— Phase 58 Plan 07 删除 handler 直推 goroutine 时不再有编译红。
- **基座自检把"环境坏"与"行为红"分离**：`TestPhase58_DrainTestBase_FixtureUsable` 逐条锁住 F-F 两列可读、队列表 nullable DATETIME 列无坑（`next_retry_at` 扫回 NULL）、mock 默认值（`dnExistsRes == true`）、`sys_dept.dept_name` 往返可用（防后续 wave 的 LDAP `department` 属性断言因基座缺列假红）。

## Task Commits

Each task was committed atomically:

1. **Task 1: mockLDAPClient 调用计数** - `31a34f90` (test)
2. **Task 2: 排水测试基座（五表 fixture + seed + 装配 helper）** - `5d63defb` (test)
3. **Task 3: Phase 57 fixture 缺口修补（F-F / F-G / F-15 / F-16）** - `6ee328e6` (test)

**Plan metadata:** (本 SUMMARY 的提交见下方 Final commit)

## Files Created/Modified

- `internal/services/addomain/ad_push_queue_drain_test.go`（**新建**）— 排水测试基座：`createDrainADConfigTable` const、`setupDrainTestDB`、seed helper ×4（含 `drainNow` / `drainBoolToInt` 两个私有工具）、装配 helper ×3、自检用例 ×1
- `internal/services/addomain/ldap_client_mock_test.go`（+13 行）— `mockLDAPClient` 新增 4 个调用计数字段；`MoveUser` / `DisableUser` / `UpdateUserAttribute` 方法体内自增
- `internal/services/system/user_service_test.go`（L117–123）— **F-F**：`sys_ad_config` 建表补 `config_name TEXT` / `base_dn TEXT`，注释块补一行说明
- `internal/api/v1/system/user_handler_test.go`（L163–165 F-15 注释；L570–575 F-G tombstone）— **F-15**：`sys_ad_config` fixture 上方补"刻意不 seed 配置行"注释；**F-G**：删除 `TestUserHandler_BuildADSyncMap` 全函数
- `internal/services/system/user_cache_impl_test.go`（L158–159）— **F-15**：同款"刻意不 seed"注释
- `internal/services/system/ad_push_queue_test.go`（L101）— **F-16**：`TestPhase57_Migration215_BareInsertWithoutID` 首行加 `t.Skip(...)`，注释块补"⚠ 已 skip"说明

## F-F / F-G / F-15 / F-16 落点行号

| 缺口 | 文件 | 落点 | 内容 |
|------|------|------|------|
| F-F | `internal/services/system/user_service_test.go` | L117（注释）/ L122–123（两列） | `config_name TEXT,` + `base_dn TEXT,` 插入 `id TEXT PRIMARY KEY,` 之后 |
| F-F（基座） | `internal/services/addomain/ad_push_queue_drain_test.go` | L48–67（const）/ L210+（自检断言） | `createDrainADConfigTable` 含同两列，自检锁 count 与 SELECT |
| F-G | `internal/api/v1/system/user_handler_test.go` | L568–586（原文）→ L570–575（tombstone 注释） | 函数整体删除，`buildADSyncMap` 字面量零残留 |
| F-15 | `internal/services/system/user_cache_impl_test.go` | L158–159 | "刻意不 seed sys_ad_config 行 —— 用例断言的正是'无启用 AD 配置时入队为 no-op'分支" |
| F-15 | `internal/api/v1/system/user_handler_test.go` | L163–165 | 同款注释（适配 TC9/TC12 语境，不与既有 Phase 57 注释重复） |
| F-16 | `internal/services/system/ad_push_queue_test.go` | L101 | `t.Skip("Phase 58 F-16: 裸 INSERT 无 ID 的 PG 契约断言（uuid 列 NOT NULL → 23502）需真实 PG 才能验证；sqlite 无法复现该约束，PG 契约断言留 v1.22…")` |

## 装配函数签名（后续 plan 直接消费，已定死）

```go
func setupDrainTestDB(t *testing.T) *gorm.DB
func seedDrainADConfig(t *testing.T, db *gorm.DB, id, configName, baseDN string, syncEnabled bool, status int)
func seedDrainUser(t *testing.T, db *gorm.DB, id, username, adDn, adOuDn string, deptID *string) *models.User
func seedDrainDept(t *testing.T, db *gorm.DB, id, name string)
func seedDrainQueueRow(t *testing.T, db *gorm.DB, id, entityID, actionType, status string, attempts int, nextRetryAt *time.Time) models.ADPushQueue
func newDrainTestLDAP() *mockLDAPClient                 // dnExistsRes 固定 true
func newDrainTestPool(accounts ...models.ADServiceAccount) *fakePool78  // 不传账号 → ErrAllAccountsUnavailable
func drainTestAccount(id, configID string) models.ADServiceAccount
```

## mock 新增计数字段清单

| 字段 | 用途 |
|------|------|
| `moveUserCalls int` | 断言"目标 OU 相同则不调 MoveUser"（compare-and-move 幂等）与终态路径不触达 MoveUser |
| `moveUserTargets []string` | 按调用顺序记录每次 `MoveUser(userDN, newOUDN)` 的 `newOUDN`（Plan 05 断言"移到了哪个 OU"） |
| `disableUserCalls int` | disable 动作（`ADPushActionDisable`）是否被调用 |
| `updateUserAttrCalls int` | 属性 Modify 路径是否被调用 |

## Decisions Made

1. **基座文件携带 `//go:build !skip_db_tests`** —— `fakePool78` 定义在带该 tag 的 `failover_client_78_07_test.go`。若新文件不带 tag，`go vet -tags skip_db_tests ./internal/services/addomain/` 会报 `undefined: fakePool78`。加 tag 后默认构建（无 tag）行为不变，`go test ./internal/services/addomain/ -run TestPhase58 -count=1` 正常捞起本文件。已实测 `go vet -tags skip_db_tests ./internal/services/addomain/` 无输出。
2. **F-G 用 tombstone 注释替位而非彻底抹掉上下文** —— 原 `// TC22: ...` 注释行与其函数体一并删除；替换为一条不出现被删函数名的说明注释（删除理由 + 覆盖损失为零的依据：四字段 diff 契约由 `TestPhase57_HasADFieldChanged_Boundaries` 锁定，对应 threat T-58-W0-05 的 accept 处置）。满足 `grep -c buildADSyncMap` == 0。
3. **`sys_dept` fixture 不扩列** —— 复用 `createDeptTable`（10 列，含 `dept_name`）。依据 Phase 58 Info 处置：`syncUserAttributes` 读部门的唯一动作是 `s.db.Where("id = ?", deptID).First(&dept)` 后取 `dept.DeptName`；GORM 生成 `SELECT *`，未在表中出现的模型列只是零值不报错（sqlite / PG 皆然）。若后续确需更多列，改 `user_ou_service_test.go` 的既有 const，不建第二份副本。

## Deviations from Plan

**1. [Rule 3 - Blocking] 新基座文件补 `!skip_db_tests` build tag**
- **Found during:** Task 2（排水基座落地）
- **Issue:** 计划要求新文件复用同包 `fakePool78`，但该 fake 定义于带 `//go:build !skip_db_tests` 的 `failover_client_78_07_test.go`；新文件不带同款 tag 时，`go test -tags skip_db_tests ./internal/services/addomain/` 与 `go vet -tags skip_db_tests ...` 会因 `undefined: fakePool78` 编译失败（属新增的编译面回归，对应 T-58-W0-04 同类风险）。
- **Fix:** 在新文件头部加 `//go:build !skip_db_tests` + `// +build !skip_db_tests`，并在文件头注释说明原因。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Verification:** `go vet -tags skip_db_tests ./internal/services/addomain/` 无输出；默认构建下 `go test ./internal/services/addomain/ -run TestPhase58 -count=1` → ok。
- **Committed in:** `5d63defb`（Task 2 提交）

**2. [Rule 3 - Blocking] gofmt 格式化新建文件**
- **Found during:** Task 2
- **Issue:** 新建文件手写对齐的注释缩进与行尾注释列宽不符合 gofmt（仓库内多数存量文件同样未 gofmt 化，属存量状态，不动）。
- **Fix:** 仅对**新建**文件执行 `gofmt -w`；存量文件（如 `ldap_client_mock_test.go` 的既有对齐）一律不碰，避免制造无关 diff。
- **Files modified:** `internal/services/addomain/ad_push_queue_drain_test.go`
- **Verification:** `gofmt -l` 对该文件无输出；三处验收 grep（`base_dn TEXT` / `config_name TEXT` / `db.Exec(createDeptTable)`）仍分别为 1/1/1。
- **Committed in:** `5d63defb`（Task 2 提交）

---

**Total deviations:** 2 auto-fixed（均为 Rule 3 blocking）
**Impact on plan:** 两处均为"不修则引入编译面/格式面回归"的必要处置，无功能范围扩张。计划指定的 fixture 复用策略、五表结构、函数签名、断言口径全部按原文落地。

## Issues Encountered

- **`gsd-sdk` CLI 在本次执行环境不可用**（`gsd-sdk: command not found`；`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>` 形式）。故本次未调用 `state.advance-plan` / `roadmap.update-plan-progress` 等 SDK 状态动词，改为按 STATE.md / ROADMAP.md 现有结构手工更新（见下方"状态更新"）。
- 工作区存在 689 项**与本 plan 无关**的未暂存 `.planning/phases/*` 删除（执行前既有状态）。全程只用 `git add <具体文件>` 逐文件暂存，未触碰这些改动。

## Known Stubs

无。本 plan 只新建测试基座与修补 fixture，未引入任何空值/占位/未接线数据源。

## Threat Flags

无新增安全相关表面。本 plan 全部改动落在 `_test.go` 文件；`files_modified` 中除新建的测试基座外均为测试文件（`verification` 节的人工核对项已确认：**Wave 0 未触碰任何生产代码文件**）。

## 状态更新

- **STATE.md**：Current Position 更新为 Phase 58 / Plan 01 完成；Decisions 追加本 plan 的 key-decisions；Performance Metrics 追加本次执行记录。
- **ROADMAP.md**：Phase 58 进度行 Plan 01 标记 complete（1/7 plans）。
- **VALIDATION.md**：Wave 0 Requirements 6 条复选框全部勾选（执行期回填）。
- **REQUIREMENTS.md**：QUEUE-03 / QUEUE-04 / QUEUE-05 在本 plan 完成的是**基座部分**，行为断言分别在 Wave 2（58-03）与 Wave 3（58-04）落地 —— 故本 plan **不**将这三项标记为 complete，留待对应 wave 收口。

## Next Phase Readiness

- **可立即开工**：Wave 1（58-02）的纯函数 RED 测试（`NormalizeDN` / `BuildOUDN` / `MatchADConfigByDN`）可直接消费本基座；`sys_ad_config.base_dn`（F-F）已就位，D-02 最长后缀匹配可走 GORM 读回而非裸 SQL。
- **Wave 2/3 的 sqlite 排水用例**已具备全部前置：队列表（迁移建的三个索引）、mock LDAP（含 `moveUserTargets`）、账号池 fake（`newDrainTestPool` 空列表即驱动 `ErrAllAccountsUnavailable`）、`ad_dn` / `ad_ou_dn` / `ad_synced_at` / `deleted_at` 四列齐全。
- **无阻塞**：Phase 57 的 42 个既有用例与 handler 层测试全绿（sentinel 已跑）；Plan 07 删除 `buildADSyncMap` 的编译障碍已清除。
- **须在 Wave 1 起遵守**：新追加用例命名一律 `TestPhase58_*`；若需向基座文件追加用例，注意该文件带 `!skip_db_tests` tag。

---

*Phase: 58-compare-and-move*
*Completed: 2026-10-19*
