---
phase: 58
slug: compare-and-move
status: draft
nyquist_compliant: false
wave_0_complete: false
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

> Task ID / Plan / Wave 在 planner 产出后回填；本表按 Req 粒度锁定（建议测试名 planner 可改名但不得删除断言）。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | 01 | 0 | F-F/F-G/W0 fixture | — | N/A | unit fixture | `go test ./internal/services/addomain/ -run TestPhase58 -count=1` | ❌ W0 | ⬜ pending |
| TBD | — | 1 | D-02 / HYGIENE-01 | — | 无匹配/多匹配不误推 | unit（纯函数） | `go test ./internal/services/addomain/ -run TestPhase58_MatchADConfigByDN -v` | ❌ W0 | ⬜ pending |
| TBD | — | 1 | HYGIENE-02 | — | 特殊字符部门名 → 合法 DN | unit（纯函数） | 同上 `-run TestPhase58_BuildOUDN` | ❌ W0 | ⬜ pending |
| TBD | — | 1 | QUEUE-04 | — | NormalizeDN 归一化 | unit | 同上 `-run TestPhase58_NormalizeDN` | ❌ W0 | ⬜ pending |
| TBD | — | 2 | QUEUE-03 | — | 拾取谓词 NULL=可拾取/上限100/created_at 序 | unit（sqlite） | 同上 `-run TestPhase58_DrainPick` | ❌ W0 | ⬜ pending |
| TBD | — | 2 | QUEUE-03 | — | 退避曲线 + 封顶 | unit | 同上 `-run TestPhase58_Backoff` | ❌ W0 | ⬜ pending |
| TBD | — | 2 | QUEUE-03 | — | 成功→pushed / 失败→failed+backoff / 5 次→dead / 池耗尽可重试 | unit + mock LDAP | 同上 `-run TestPhase58_DrainSuccess` | ❌ W0 | ⬜ pending |
| TBD | — | 2 | QUEUE-03 F-11 | — | 编辑清零 attempts 后不得裸判 dead | unit | `go test ./internal/services/system/ -run TestPhase58 -v` | ❌ W0 | ⬜ pending |
| TBD | — | 3 | QUEUE-04 | — | compare-and-move：相同跳过 / 不同 Move+回写 ad_dn+ad_ou_dn（F-A）/ 重读最新值 / config 收敛（F-B） | unit + mock LDAP | 同上 `-run TestPhase58_CompareAndMove` | ❌ W0 | ⬜ pending |
| TBD | — | 3 | QUEUE-05 | — | code 32 → dead + 清 ad_dn + 不烧 attempts + 不再重试；F-07 孤儿 disable → dead | unit + mock LDAP | 同上 `-run TestPhase58_Terminal` | ❌ W0 | ⬜ pending |
| TBD | — | 3 | D-01 | — | 批量 EnqueueUpdate：预过滤+门控+去重+合并 | unit | `go test ./internal/services/system/ -run TestPhase58 -v` | ❌ W0 | ⬜ pending |
| TBD | — | 3 | F-10 / F-07 入队侧 | — | Delete/BatchDelete 回滚 + 只对存在行入队 | unit | 同上 | ✅（模板） | ⬜ pending |
| TBD | — | 4 | D-03 | — | `getDefaultADConfigID` 双份消灭 | source-assert | `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` == 0 | ✅ | ⬜ pending |
| TBD | — | 4 | D-05 | — | SecondsToCronSpec 合法/非法回落；migration_216 seed 幂等 | unit | `go test ./internal/scheduler/ -run TestPhase58 -v` | ❌ W0 | ⬜ pending |
| TBD | — | 4 | D-05 / SC1 | — | 排水器未注册 sys_job | source-assert | grep `sys_job` 表 seed 无 `ad_push_queue` 命中 | ✅ | ⬜ pending |
| TBD | — | 4 | F-12 | — | 时间戳统一 UTC | unit | `go test ./internal/services/addomain/ -run TestPhase58 -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/addomain/ad_push_queue_drain_test.go` —— 新建（RED）。自带 sqlite fixture：`ad_push_queue`（直调 `migrations.Migrate215CreateADPushQueue`）、`sys_user`（含 `ad_dn`/`ad_ou_dn`/`ad_synced_at`/`deleted_at` 全列）、`sys_ad_config`（**必须含 `base_dn` + `config_name`**，F-F）、`sys_dept`、`sys_dept_ou_mapping`。`internal/services/addomain` 包目前没有这种 fixture，需新建。
- [ ] mock LDAP：`ldap_client_mock_test.go` 已有 `mockLDAPClient`，确认已实现 `MoveUser` / `DisableUser` / `DNExists` 且**可注入错误**驱动 code 32 / 失败路径；不可注入则写可配置 fake。
- [ ] mock AccountPool：`account_pool_*_test.go` 已有先例（`ListAvailable` / `MarkFailure` / `MarkSuccess`），确认可返回空列表驱动 `ErrAllAccountsUnavailable`。
- [ ] `internal/services/system/user_service_test.go` —— 补 `sys_ad_config.base_dn` 列（F-F）。
- [ ] `internal/api/v1/system/user_handler_test.go` —— 处置 `TestUserHandler_BuildADSyncMap`（F-G），否则 D-01 删直推时编译红。
- [ ] Framework install：无（依赖已具备）。

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

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
