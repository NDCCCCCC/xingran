---
phase: 58
slug: compare-and-move
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-10-17
---

# Phase 58 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 来源：58-RESEARCH.md §8 Validation Architecture（研究期实测基线）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `stretchr/testify` (`assert`/`require`) + `github.com/glebarez/sqlite`（in-memory） |
| **Config file** | 无 —— sqlite 测试 `gorm.Open(sqlite.Open(":memory:"))`（先例 `internal/services/system/user_service_test.go:49`；migrations 包用 `file:<TestName>?mode=memory&cache=shared`，先例 `migration_210_normalize_menu_parent_id_test.go:11`） |
| **新增依赖** | 无（robfig/cron、go-ldap、testify 均已在 go.mod） |
| **测试落地文件（建议）** | ① `internal/services/addomain/ad_push_queue_drain_test.go`（新建，排水核心）② `internal/services/addomain/ad_dn_config_match_test.go` 或并入 ①（D-02 纯函数 + NormalizeDN + OU 转义纯函数）③ `internal/services/system/ad_push_queue_test.go`（追加 F-07/F-10）④ `internal/api/v1/system/user_handler_test.go`（F-G 用例处置）⑤ `internal/core/db/migrations/migration_216_*.go` + 可选 `_test.go` |
| **Quick run command** | `go test ./internal/services/addomain/ -run TestPhase58 -count=1` |
| **Second env sentinel** | `go test ./internal/services/system/ -run TestPhase57 -count=1`（Phase 57 的 42 个既有用例必须保持绿 —— D-04 触碰 BatchDelete 的回归面） |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | quick ~3s（研究期实测 sentinel 2.788s） |

---

## Sampling Rate

- **After every task commit:** `go build ./...` + `go test ./internal/services/addomain/ -run TestPhase58 -count=1`
- **After every plan wave:** `go test ./...` + `go vet ./internal/services/addomain/ ./internal/scheduler/ ./internal/services/system/`
- **触碰 Phase 57 代码的 wave（D-01 删除直推 / F-07 BatchDelete Pluck）:** 额外跑 sentinel `go test ./internal/services/system/ -run TestPhase57 -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green + 静态断言（grep）通过
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

> 本表已按 **task 粒度** 回填（2026-10-17 规划期修订轮）：Task ID / Plan / Wave 与 7 份 `58-0N-PLAN.md` 的实际任务一一对应。测试名可与 planner 提案略差异，但**不得删除断言**。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 58-01/T1 | 01 | 0 | Wave 0 fixture（mock 调用计数） | T-58-W0-04 | mock 新增字段不破坏既有用例 | unit fixture | `go test ./internal/services/addomain/ -count=1` | ✅ 文件既有（改） | ✅ green `31a34f90` |
| 58-01/T2 | 01 | 0 | F-F / F-H（排水基座 5 表 + seed + 装配 helper） | T-58-W0-01/02/03 | `base_dn`/`config_name` 补列；复用既有 `createUserTable`/`createDeptOUMappingTable`/`createDeptTable` 零副本 | unit fixture | `go test ./internal/services/addomain/ -run TestPhase58_DrainTestBase -count=1` | ✅ 新建（原 ❌ W0） | ✅ green `5d63defb` |
| 58-01/T3 | 01 | 0 | F-F / F-G / F-15 / F-16 | T-58-W0-05 | 删 `TestUserHandler_BuildADSyncMap` 不丢四字段 diff 契约 | unit + 回归 | `go build ./... && go test ./internal/services/system/ -run TestPhase57 -count=1 && go test ./internal/api/v1/system/ -count=1` | ✅（改既有） | ✅ green `6ee328e6` |
| 58-02/T1 | 02 | 1 | HYGIENE-02 + QUEUE-04（`NormalizeDN` / `BuildOUDN`） | T-58-02-02/04 | 特殊字符部门名 → 合法 DN；**attr 名与值两侧统一 ToLower** | unit（纯函数） | `go test ./internal/services/addomain/ -run "TestPhase58_NormalizeDN|TestPhase58_BuildOUDN" -count=1` | ✅ 新建（原 ❌ W0） | ✅ green `b2909660` |
| 58-02/T2 | 02 | 1 | D-02（最长后缀唯一匹配契约） | T-58-02-01/05/06 | 无匹配 / 歧义 → error，不误路由 | unit（纯函数 + GORM 读回） | `go test ./internal/services/addomain/ -run "TestPhase58_MatchADConfigByDN|TestPhase58_FirstEnabledADConfigID|TestPhase58_FirstSyncEnabledADConfigID" -count=1` | ✅ 新建（原 ❌ W0） | ✅ green `e1fca53a` |
| 58-02/T3 | 02 | 1 | HYGIENE-01（三处调用点收敛） | T-58-02-03 | 双份 `getDefaultADConfigID` 消灭；F-D 两入口语义不合并 | source-assert + 回归 | `go test ./internal/scheduler/ ./internal/api/v1/system/ -count=1` + `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` 每行 0 | ✅（改既有） | ✅ green `fec65e7d` |
| 58-03/T1 | 03 | 2 | QUEUE-03（拾取谓词 / 退避 / 配置分组）+ 测试注入点 + `markDead` 契约 | T-58-03-05/10 | 只读 `ADPushQueueActivePredicate` 常量；`clientFactoryOverride` 生产恒 nil | unit（sqlite） | `go test ./internal/services/addomain/ -run "TestPhase58_BackoffDuration|TestPhase58_PickRows|TestPhase58_Drain_NoConfigDefer" -count=1` | ❌ W0 | ✅ green `04dfa698` |
| 58-03/T2 | 03 | 2 | QUEUE-03 / QUEUE-05 / F-11 / F-12 / F-14 | T-58-03-01~09 | code 32 → dead + 清 `ad_dn` 不烧 attempts；F-11 禁裸 attempts 判死；`markDead` 显式带 attempts | unit + mock LDAP | `go test ./internal/services/addomain/ -run "TestPhase58_Drain_" -count=1`（sentinel：`go test ./internal/services/system/ -run TestPhase57 -count=1`） | ❌ W0 | ✅ green `34294f67` |
| 58-04/T1 | 04 | 3 | F-B（配置作用域 OU 查询） | T-58-04-01 | 目标 OU 按 `ad_config_id` 收敛，多配置不跨域 | unit（sqlite） | `go test ./internal/services/addomain/ -run "TestPhase58_FindOUDNByDeptIDForConfig" -count=1` | ✅（改既有） | ✅ green `44b4b958`（3/3 PASS） |
| 58-04/T2 | 04 | 3 | QUEUE-04 / QUEUE-05（compare-and-move + 终态分流） | T-58-04-02~09 | `MoveUser` 后 `ad_dn`+`ad_ou_dn` 双写；**终态路径禁被 `markPushed` 覆盖** | unit + mock LDAP | `go test ./internal/services/addomain/ -run "TestPhase58_Drain_" -count=1` | ❌ W0 | ✅ green `f00f41b1`（23/23 PASS = Plan 03 的 13 + 本 plan 10） |
| 58-04/T3 | 04 | 3 | F-A（`user_ad_sync_service.go` 两处修复） | T-58-04-02 | 两处载体与 drainer 语义一致 | unit + mock LDAP | `go test ./internal/services/addomain/ -run "TestPhase58_Move" -count=1` | ✅（改既有） | ✅ green `c89b4835`（2/2 PASS） |
| 58-05/T1 | 05 | 4 | D-05（sys_config 单旋钮 + migration_216 双方言） | T-58-05-06/07 | seed 幂等不覆盖管理员改动；非阻断档 | unit（sqlite）+ migration | `go test ./internal/core/db/migrations/ -run "TestPhase58_Migration216" -count=1` + `go test ./internal/services/addomain/ -run "TestPhase58_SecondsToCronSpec|TestPhase58_LoadDrainIntervalSeconds" -count=1` | ❌ W0 | ✅ green `721d2890` |
| 58-05/T2 | 05 | 4 | D-05 / SC1（cron 接线 + 禁 sys_job + 账号池单例） | T-58-05-01/02/04/08 | 未注册 `sys_job`；池经 `s.getPool()` 字面取单例；**`NewAccountPool` 计数 == 3（`:106` 熔断恢复任务局部池 + `:125` 全局单例 + `getPool()` 惰性兜底；低于 3 即为误删）且 `addomain.NewAccountPool(s.db, nil)` == 1** | source-assert + 回归 | `go test ./internal/scheduler/ -count=1` + `grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go | grep -i "sys_job" | wc -l` == 0 | ✅（改既有） | ✅ green `1bcd3106` |
| 58-06/T1 | 06 | 5 | D-01（`EnqueueUpdateBatch` 批量入队变体） | T-58-06-03/04 | 只产 `update_user` 行；不做 diff；错误上抛不吞 | unit（sqlite） | `go test ./internal/services/addomain/ -run "TestPhase58_EnqueueUpdateBatch" -count=1` | ❌ W0 | ✅ green `253461c2`（5/5 PASS） |
| 58-06/T2 | 06 | 5 | SC6（D-01/D-03 两条直推路径删除）+ Q5 | T-58-06-01/02/05/08 | 5 条 grep 静态证伪；operlog 顺序不变；`userADSyncService` 字段保留 | source-assert + 回归 | `go build ./... && go test ./internal/api/v1/system/ -count=1` + 5 条 grep == 0 | ✅（改既有） | ✅ green `ab8a7231`（5 grep 全 0；未加任何互斥开关） |
| 58-06/T3 | 06 | 5 | F-07（入队侧 Pluck）/ F-10（回滚用例） | T-58-06-06/07 | **Pluck 排在软删之前**（软删后 → 作用域过滤 → existing 恒空 → 入队 no-op）；入队数 == 软删数 | unit + Phase 57 sentinel | `go test ./internal/services/system/ -run "TestPhase58_UserService" -count=1`（sentinel：`go test ./internal/services/system/ -run TestPhase57 -count=1`） | ✅（改既有） | ✅ green `9c2dc88d`（3/3 PASS；Pluck:657 < Delete:662；sentinel 41 PASS/0 FAIL） |
| 58-07/T1 | 07 | 6 | SC1 / SC2 / SC5（端到端收尾用例） | T-58-07-03 | 入队 → 排水 → compare-and-move → 终态 走生产符号；禁为变绿弱化断言 | unit + mock LDAP | `go test ./internal/services/addomain/ -run "TestPhase58_EndToEnd" -count=1`（**全量 `go test ./...` + 五包 `go vet` 移至 plan 级 `<verification>`**） | ❌ W0 | ⬜ pending |
| 58-07/T2 | 07 | 6 | SC1~SC6 逐条证据 + F-13 归档 | T-58-07-01/04/06 | 「已验证」必须附命令 + 实际输出；F-13 只核对不改 | source-assert | `go test ./internal/services/addomain/ -run "TestPhase58_MatchADConfigByDN|TestPhase58_BuildOUDN|TestPhase58_PickRows_TerminalRowsNotPicked" -count=1` + 各 SC 的 grep | ✅（文档） | ⬜ pending |
| 58-07/T3 | 07 | 6 | 5 项人工 UAT + REQUIREMENTS Traceability 回填 | T-58-07-02/05 | 人工项显式留档不可默认「已验证」；Phase 57 的 Done 项不被误伤 | doc-assert | `grep -c "UAT-58-" .planning/phases/58-compare-and-move/58-HUMAN-UAT.md` >= 5 + REQUIREMENTS 5 个 `- [x]` | ✅（文档） | ⬜ pending |

> **Task ID 图例**：`58-0N/TM` = 计划 `58-0N-PLAN.md` 的第 M 个 task。共 **19 个 task / 7 个 plan / 7 个 wave**（wave 0 = 58-01，wave 1..6 = 58-02..58-07 顺序执行）。
> **`File Exists` 列语义**：❌ W0 = 该测试文件由 wave 0（58-01）新建，当前尚不存在；✅ = 文件既有，本 task 只做增量。
> **`wave_0_complete: true` 的含义**：Wave 0 的 6 条 Requirements **已被 58-01 的三个 task 完整覆盖并排入 wave 0**，不是指文件已存在（执行尚未开始）。
> 本表按 task 粒度锁定：Plan 07 只跑 addomain 目标用例，全量回归在该 plan 的 `<verification>` 层执行 —— 单 task 不背 3 分钟反馈延迟。

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> 本节条目的实现负责已完整分配到 **58-01 的三个 task**（T1 mock 计数 / T2 排水基座 / T3 旧 fixture 缺口），故 frontmatter `wave_0_complete: true`。下面的复选框在**执行期**（wave 0 执完后）勾选，不是规划期未完成。

- [x] `internal/services/addomain/ad_push_queue_drain_test.go` —— 新建（RED）。自带 sqlite fixture：`ad_push_queue`（直调 `migrations.Migrate215CreateADPushQueue`）、`sys_user`（含 `ad_dn`/`ad_ou_dn`/`ad_synced_at`/`deleted_at` 全列）、`sys_ad_config`（**必须含 `base_dn` + `config_name`**，F-F）、`sys_dept`、`sys_dept_ou_mapping`。`internal/services/addomain` 包目前没有这种 fixture，需新建。 → **58-01 Task 2 完成**（`setupDrainTestDB`；`sys_user`/`sys_dept`/`sys_dept_ou_mapping` 复用 `user_ou_service_test.go` 既有 const，零副本）
- [x] mock LDAP：`ldap_client_mock_test.go` 已有 `mockLDAPClient`，确认已实现 `MoveUser` / `DisableUser` / `DNExists` 且**可注入错误**驱动 code 32 / 失败路径；不可注入则写可配置 fake。 → **58-01 Task 1 完成**（`moveUserErr`/`disableUserErr`/`updateUserAttrErr` 既有可注入；新增 `moveUserCalls`/`moveUserTargets`/`disableUserCalls`/`updateUserAttrCalls` 计数）
- [x] mock AccountPool：`account_pool_*_test.go` 已有先例（`ListAvailable` / `MarkFailure` / `MarkSuccess`），确认可返回空列表驱动 `ErrAllAccountsUnavailable`。 → **58-01 Task 2 完成**（复用同包 `fakePool78`；`newDrainTestPool()` 空变参 → `listAvailableRes == nil` → `ErrAllAccountsUnavailable`）
- [x] `internal/services/system/user_service_test.go` —— 补 `sys_ad_config.base_dn` 列（F-F）。 → **58-01 Task 3 完成**（L122–123 补 `config_name` / `base_dn`）
- [x] `internal/api/v1/system/user_handler_test.go` —— 处置 `TestUserHandler_BuildADSyncMap`（F-G），否则 D-01 删直推时编译红。 → **58-01 Task 3 完成**（函数整体删除，`grep -c buildADSyncMap` == 0）
- [x] Framework install：无（依赖已具备）。

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实 AD 上的 compare-and-move（MoveDN 真生效 + 二次排水不重复移动） | QUEUE-04 | 需真域控 | 编辑用户部门 → 观察排水后 AD 端 OU 变化；再手动触发排水确认无二次移动 |
| 账号池熔断 30min 内排水器退避节奏 | QUEUE-03 | 需人为触发熔断 | 观察 sys_ad_service_accounts 熔断期间 attempts 增长符合 60s→…→30min 曲线 |
| `sys_config` 改 `drain_interval_seconds` 重启生效 | D-05 | 需重启后端 | 改值 → 重启 → 日志确认新间隔 |
| 多 AD 配置（父子域）真实环境 longest-suffix 命中 | D-02 | 需第二个域 | 两域环境编辑分属用户 → 各自命中正确域控 |
| code 32 真实触发（AD 端删对象 → 本地编辑 → 排水 → dead + ad_dn 清空） | QUEUE-05 | 需真域控 | AD 端删 OU/用户 → 编辑用户 → 排水 → 查队列行 dead 且 sys_user.ad_dn 为空 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies（19/19 task 行均有 Automated Command；唯一例外 58-07/T1 的全量回归按 Nyquist 反馈延迟预算上移到 plan 级 `<verification>`，targeted `-run TestPhase58_EndToEnd` 仍在 task 级）
- [x] Sampling continuity: no 3 consecutive tasks without automated verify（逐 task 连续，19 行全填）
- [x] Wave 0 covers all MISSING references（6 条 Wave 0 Requirements → 58-01/T1~T3；由 58-01 Task 2 的基座自检用例兜底）
- [x] No watch-mode flags（全表无 `-watch` / `--watch`）
- [x] Feedback latency < 30s（task 级只跑单包 `-run` 目标用例；`go test ./...` 仅出现在 plan 级）
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved（2026-10-17，规划期回填；执行期若某 task 偏离本表需在对应 SUMMARY 记录原因）
