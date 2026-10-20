---
phase: "61"
slug: "shadow-table-integrity"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-20"
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
| 61-01-01 | 01 | 1 | SHADOW-01 | T-61-01 / T-61-02 | 两道闸门拒绝超阈值删除 | unit | `go test ./internal/services/addomain/ -run TestShadowCleanup -count=1` | ❌ W0 | ⬜ pending |
| 61-01-02 | 01 | 1 | SHADOW-01 | — | gate1：该类 LDAP 0 条 → 不删 | unit | `go test ./internal/services/addomain/ -run TestShadowCleanup_RejectsEmptyLDAP -count=1` | ❌ W0 | ⬜ pending |
| 61-01-03 | 01 | 1 | SHADOW-01 | — | gate2：拟删 >50% 拒绝；恰 50% 放行 | unit | `go test ./internal/services/addomain/ -run TestShadowCleanup_Threshold -count=1` | ❌ W0 | ⬜ pending |
| 61-01-04 | 01 | 1 | SHADOW-01 | — | 复活：软删行同 DN 复现 → 下一轮恢复可见不撞约束 | unit | `go test ./internal/services/addomain/ -run TestSync.*Resurrect -count=1` | ❌ W0 | ⬜ pending |
| 61-0X-0X | — | — | SHADOW-02 | — | Create 失败 → Delete 回滚、旧成员原样 | unit | `go test ./internal/services/addomain/ -run TestSyncGroupMembers -count=1` | ✅ 既有需改造 + ❌ 失败注入 | ⬜ pending |
| 61-0X-0X | — | — | SHADOW-02 | — | 三个调用方路径不回归 | unit | 同上（SyncSingleGroup / syncGroupEntries 既有用例） | ✅ 既有 | ⬜ pending |
| 61-0X-0X | — | — | SHADOW-03 | — | 清理只在 full 分支执行 / 挂载位置正确 | unit | `go test ./internal/services/addomain/ -run TestSync78_SyncDataByID -count=1` + 新增挂载断言 | ✅/❌ | ⬜ pending |
| 全部 | 全部 | — | 全部 | — | 全套件不回归 | suite | `go test ./... -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Task ID 列的计划/波次编号为占位——planner 落定 plan 拆分后由 executor 按实际编号回填。*

---

## Wave 0 Requirements

- [ ] `internal/services/addomain/shadow_cleanup_test.go` — SHADOW-01 全部用例，复用 `setupSync78DB`（勿新建副本表）
- [ ] `sync_78_05_test.go` fixture 扩展：`sys_ad_group_member` 建表语句加 `CHECK (user_dn != 'FORCED_FAIL')`（或独立建表 helper），供 SHADOW-02 失败注入
- [ ] computer 清理用例建表：复用 `computer_78_06_test.go` 既有 helper（`setupSync78DB` 无 sys_ad_computer 表）
- [ ] 框架安装：无需（既有）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 首轮批量清理规模真实观察 | SHADOW-01 | 需生产 PG + 真实 AD 数据规模 | 部署后核对 sys_oper_log / SyncResult 中清理计数与 AD 端实际对象数 |
| 事务空窗真观察 | SHADOW-02 | 需并发外部查询会话 | 重放期间从第二会话查询 sys_ad_group_member，确认无空窗 |
| 闸门真触发 | SHADOW-01 | 需人为构造 AD 端大规模删除 | 测试域控删除 >50% 对象后跑全量同步，确认拒绝与日志 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
