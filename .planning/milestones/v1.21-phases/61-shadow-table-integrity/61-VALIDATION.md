---
phase: "61"
slug: "shadow-table-integrity"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-10-20"
validated: "2026-10-21"
---

# Phase 61 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test`（testify require/assert + glebarez/sqlite 内存库） |
| **Config file** | none（build tag `!skip_db_tests` 约定，见 `sync_78_05_test.go:1`） |
| **Quick run command** | `go test ./internal/services/addomain/ -count=1`（基线实测 ok 8.165s） |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds (addomain 包) / ~2 min (full suite) |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + `go test ./internal/services/addomain/ -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 61-01-01 | 01 | 1 | SHADOW-01 | T-61-01 / T-61-02 | user 类别清理 + 两道闸门（0 条拒绝 / >50% 拒绝、恰 50% 放行）+ full 分支单点挂载 | unit | `go test ./internal/services/addomain/ -run TestShadowCleanup -count=1` | ✅（61-01 落地） | ✅ green（12/12 PASS） |
| 61-01-02 | 01 | 1 | SHADOW-01 / SHADOW-03 | — | OU + computer 类别扩展（per-category 独立闸门）+ SyncResult 三计数 | unit | `go test ./internal/services/addomain/ -run TestShadowCleanup -count=1` | ✅（61-01 落地） | ✅ green（全包 ok） |
| 61-02-01 | 02 | 2 | SHADOW-01 | T-61-05 | sync.go 三类 upsert 复活化（existing Unscoped + OnConflict 补 deleted_at + 命中行 DeletedAt 清零）+ D-78-05c 两用例翻转 | unit | `go test ./internal/services/addomain/ -run "TestSync_Resurrect" -count=1` | ✅（61-02 落地） | ✅ green（5/5 PASS，先 RED 后 GREEN） |
| 61-02-02 | 02 | 2 | SHADOW-01 | — | computer.go + group_sync_service.go 复活化（增量路径 + SyncSingleGroup 冲突更新） | unit | `go test ./internal/services/addomain/ -run "TestSync_ResurrectComputer\|TestSync_ResurrectGroup_IncrementalPath" -count=1` | ✅（61-02 落地） | ✅ green（3/3 PASS） |
| 61-03-01 | 03 | 3 | SHADOW-02 | T-61-06 | Create 失败 → Delete 回滚、旧成员原样（CHECK 注入）；重放幂等；Unscoped 硬删语义保留 | unit | `go test ./internal/services/addomain/ -run "TestSyncGroupMembers" -count=1` | ✅（61-03 落地：fixture CHECK + 2 新用例） | ✅ green（2/2 PASS，回滚用例先 RED 后 GREEN） |
| 61-03-02 | 03 | 3 | SHADOW-02 | T-61-07 | 三个调用方（sync.go syncGroups / group_sync_service.go SyncSingleGroup / syncGroupEntries）零改动不回归 | unit | `go test ./internal/services/addomain/ -count=1` + `go test ./... -count=1` | ✅ 既有 | ✅ green（全包 ok 8.0s；全套件 78 ok） |
| 全部 | 全部 | — | 全部 | — | 全套件不回归 | suite | `go test ./... -count=1` | ✅ | ✅ green（78 ok / 0 FAIL，2026-10-21） |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*回填注记（2026-10-21，61-03 executor）：*
- *占位 61-01-01~04 行按实际 plan 拆分收敛为 61-01-01/61-01-02（planner 期预设 61-01 含 61-02 的复活用例，实际复活归 61-02）；61-02 复活路径行补 61-02-01/02。*
- *SHADOW-03 挂载断言由 61-01-01 行承担（TestShadowCleanup_FullSyncOnly + grep 挂载点计数 == 1，见 61-01-SUMMARY D5），不再单列。*
- *SHADOW-02「三个调用方路径」覆盖口径：SyncSingleGroup 无直接单测（既有缺口，非本 phase 引入；该路径仅 ad_domain_handler.go:782 调用），其事务化受益由源断言零改动 + 全包/全套件零回归 + syncGroupEntries 增量路径用例（TestSync_ResurrectGroup_IncrementalPath / TestShadowCleanup_CategoriesIndependent）承担。*
- *全套件基线：已知既有 flake `TestSNMPClient_WaitForReady_Timeout`（internal/device，与 addomain 无依赖，58-07 口径）本轮未触发；如复现按 58-07 披露口径处理，不影响本表判定。*

---

## Wave 0 Requirements

- [x] `internal/services/addomain/shadow_cleanup_test.go` — SHADOW-01 全部用例，复用 `setupSync78DB`（勿新建副本表）（61-01 落地）
- [x] `sync_78_05_test.go` fixture 扩展：`sys_ad_group_member` 建表语句加 `CHECK (user_dn != 'FORCED_FAIL')`（或独立建表 helper），供 SHADOW-02 失败注入（61-03 落地，表级 CHECK 追加在既有 DDL，共享 fixture 零副作用）
- [x] computer 清理用例建表：复用 `computer_78_06_test.go` 既有 helper（`setupSync78DB` 无 sys_ad_computer 表）（61-01 落地）
- [x] 框架安装：无需（既有）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 首轮批量清理规模真实观察 | SHADOW-01 | 需生产 PG + 真实 AD 数据规模 | 部署后核对 sys_oper_log / SyncResult 中清理计数与 AD 端实际对象数 |
| 事务空窗真观察 | SHADOW-02 | 需并发外部查询会话 | 重放期间从第二会话查询 sys_ad_group_member，确认无空窗 |
| 闸门真触发 | SHADOW-01 | 需人为构造 AD 端大规模删除 | 测试域控删除 >50% 对象后跑全量同步，确认拒绝与日志 |

**执行状态（2026-10-21 盘点）：** 三项均待真环境窗口（生产 PG + 真实 AD 或测试域控），
代码侧无遗留。逐项执行前提：①首轮清理发生在部署后首次全量同步，需在同步前后各取一次
AD 对象计数；②需第二会话（psql 直连即可）在重放期间轮询 sys_ad_group_member 计数；
③需测试域控（**严禁**在生产域控人为删除对象），建议搭最小域或用快照回滚。完成后在此回填结果。

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated（2026-10-21，61-03 executor 回填；自动化项全绿，Manual-Only 三项见上表待真环境执行）
