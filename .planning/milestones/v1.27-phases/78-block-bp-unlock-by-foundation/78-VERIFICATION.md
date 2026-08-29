# Phase 78: 阻塞包攻破·基建解锁 — Plan Verification Report

**Phase Goal:** 用 Phase 75/76 修复与基建把 `internal/core` / `internal/device` / `internal/services/addomain` 全部推过 70%,3 结构阻塞包清零。
**Plans Verified:** 7 (78-01..78-07)
**Status:** **PASS-WITH-FIXES** — 所有 7 个 plan 结构上交付其切片;1 个 MED 问题在 78-01,已通过文档化 fallback 缓解;5 项必检维度(需求覆盖/任务完整性/依赖正确性/pitfall 覆盖/决策审计)齐备;race-safety、零生产代码改动、goal-backward SC 映射全部成立。

**整体: PASS-WITH-FIXES;HIGH: 0;MED: 1;LOW: 5**

---

## Goal-Backward SC Smoke Check

ROADMAP §Phase 78 SC#1-5 逐项回填(从 7 个 plan 的 must_haves 反向验证):

| SC | Truth Required | Advancing Plan(s) | Goal-Backward |
|----|----------------|-------------------|---------------|
| **SC#1** | internal/core ≥70% (40.2%→≥70%; Init 302 + captcha 98 + Close 60 stmts) | 78-01 + 78-02 | ✓ COVERED |
| **SC#2** | internal/device ≥70% (39.1%→≥70%; FileTransport scrapli + snmp UDP + task_scheduler) | 78-03 + 78-04 | ✓ COVERED |
| **SC#3** | internal/services/addomain ≥70% (21.8%→≥70%, 缺 ~1165; sqlite 段 ~535 + Iface stub 段 ldap_client 159 + failover) | 78-05 + 78-06 + 78-07 | ✓ COVERED |
| **SC#4** | check-coverage.sh P2_RATCHET_internal_core=39.00 + P2_RATCHET_internal_device=38.50 对应包实测 ≥70% | 78-02 显式对照 P2_RATCHET_internal_core;78-04 显式对照 P2_RATCHET_internal_device;78-07 确认 addomain 无独立豁免行 | ✓ COVERED |
| **SC#5** | 每个 plan 完成点 `go test ./...` 全绿,gate 不倒退 | 所有 plan 的 verification 块均含 `go build ./...` + `go test ./...` + `go test -race ./...` 验收 | ✓ COVERED |

**Goal-backward sanity:** 4 个独立 SC 全部至少有一个独立 plan 提供推进面;每个 SC 由 2-3 个 plan 分阶段推动(不依赖单一 plan 失败)。PLAN-set 健壮。

---

## Per-Plan Scorecard

| Plan | Goal | SC Testability | Task Atomicity | Deps | Pitfalls | Decision Audit | Race | Prod Scope | Verdict |
|------|------|----------------|----------------|------|----------|----------------|------|-----------|--------|
| **78-01** | ✓ BLOCK-03 partial | ✓ | ✓ 5 tasks | ✓ wave 1 | ✓ QUIRK-01, QUIRK-07 | ✓ D-78-01/08/10/11/01a/01b | ✓ RunT cleanup + -race gate | ✓ zero changes (D-78-10 excepted) | ⚠ **WITH-FIX** |
| **78-02** | ✓ BLOCK-03 main | ✓ | ✓ 5 tasks, probe-first | ✓ wave 2, dep 78-01 | ✓ R-7 MultiLevelCache, R1 Init hang, QUIRK-07 | ✓ D-78-01/02a-c/03/08/10/11 | ✓ hard-timeout + -race | ✓ zero changes (D-78-03 (a) excepted) | ✓ **PASS** |
| **78-03** | ✓ BLOCK-04 main | ✓ | ✓ 5 tasks | ✓ wave 1 | ✓ S-2/R2 FileTransport, R7 createConnection via D-78-05 | ✓ D-78-05/09/03a-d/08/10/11 | ✓ -race + cleanup | ✓ zero changes (D-78-05 forbids createConnection seam) | ✓ **PASS** |
| **78-04** | ✓ BLOCK-04 finish + SC#4 | ✓ | ✓ 4 tasks (probe-as-Task-1) | ✓ wave 2, dep 78-03 | ✓ R3 BER encoding via D-78-06, closeLocked nil-Conn via D-78-04c | ✓ D-78-06/02/04a-c/08/09/10/11 | ✓ fake goroutine cleanup + -race | ✓ zero changes (D-78-10 excepted) | ✓ **PASS** |
| **78-05** | ✓ BLOCK-05 segment A | ✓ | ✓ 5 tasks | ✓ wave 1 | ✓ G4 RESOLVED (seam), sqlite 缺表 family pattern | ✓ D-78-07/05a-c/08/10/11 | ✓ -race | ✓ zero changes (D-78-07 REJECTS seam) | ✓ **PASS** |
| **78-06** | ✓ BLOCK-05 segment B | ✓ | ✓ 5 tasks | ✓ wave 2, dep 78-05 | ✓ isUniqueConstraintError 方言 via D-78-06c | ✓ D-78-06a-e/08/10/11 | ✓ -race | ✓ zero changes | ✓ **PASS** |
| **78-07** | ✓ BLOCK-05 finish + SC#4 + package checkpoint | ✓ | ✓ 5 tasks | ✓ wave 3, dep [78-05, 78-06] | ✓ G4 RESOLVED explicit, R5 LDAP wire via D-78-04, cleanDomain via D-78-07c | ✓ D-78-04/07a-d/08/10/11 | ✓ 5-10s hard-timeout + -race | ✓ zero changes (D-78-10 excepted) | ✓ **PASS** |

---

## Dimension-by-Dimension Findings

**D1 (Requirement Coverage):** BLOCK-03 → 78-01+78-02 ✓; BLOCK-04 → 78-03+78-04 ✓; BLOCK-05 → 78-05+78-06+78-07 ✓. Unmapped: 0.

**D2 (Task Completeness):** All 34 tasks have explicit `<read_first>` / `<action>` / `<verify>` (with `<automated>`) / `<done>` / `<acceptance_criteria>`. Edge case: 78-02 Task 1 acceptance criterion permits "探针结论 B (Init returns error)" — by design per ROADMAP L211 + D-78-02b. Not a defect.

**D3 (Dependency Correctness):** Clean three-chain dependency graph:
- `78-01 → 78-02` (core 同包,wave 1→2)
- `78-03 → 78-04` (device 同包,wave 1→2)
- `78-05 → 78-06 → 78-07` (addomain 同包,wave 1→2→3)

Cross-package 并行:wave 1: 78-01 ∥ 78-03 ∥ 78-05。Wave 2: 78-02 ∥ 78-04 ∥ 78-06。Wave 3: 78-07 独立。无环,无前向引用。

**D4 (Key Links Planned):** 每 plan 的 `key_links` 段引用显式 `from → to via pattern` 链。所有链均验证 source → target 接线(e.g., 78-03's `pool.connections[deviceID]` pre-seed with `mu = p.getDeviceLock(deviceID)` lock-consistency note 是 D-78-05 第三路径,超出 RESEARCH 的 DQ5 (a)/(b))。

**D5 (Scope Sanity):** 无 plan 触及 5-task 阻塞阈值。78-03/78-06/78-07 处于预算上沿,但有 D-78-NN 决策与 76-02 先例支撑,合理。

**D6 (Verification Derivation):** 所有 `must_haves.truths` 都是用户可观察的(非"library installed"而是"GenerateCaptcha 走 :257 原子分支可直测 via mr.TTL assertion" / "GetConnection returns same `*PooledConnection` pointer" / "fake.RequestCount() == 1")。所有 truths 映射可验证 artifacts(min_lines + `contains` patterns)。所有 key_links 链 source → target 经显式 pattern。

**D7 (Context Compliance):** N/A — Phase 78 未运行 `/gsd:discuss-phase`;planner 吸收所有 8 个判断点到每 plan `decision_audit` 表(D-78-01..11)。无用户决策可矛盾。

**D7b (Scope Reduction Detection):** 扫描所有 task actions 的 scope-reduction 用语。**0 例**用 "v1"/"简化"/"stub"/"占位"/"未来增强" 作为用户强制缩减的正当性。文档化 fallback 路径(78-04 D-78-02 ≥40%, 78-07 D-78-04 ≥45%)是**研究驱动不确定性**的预算 hedge,非用户强制缩减。ROADMAP L195 "补 ~1165 stmts" 是覆盖率缺口,非"每 stmt 必须覆盖"。

**D7c (Architectural Tier Compliance):** SKIPPED — Phase 78 RESEARCH.md 无 `## Architectural Responsibility Map` 段。

**D8 (Nyquist Compliance):** SKIPPED with explicit gate check (8e) — Phase 78 无 `*-VALIDATION.md`,`workflow.nyquist_validation` 不在 config.json。

**D9 (Cross-Plan Data Contracts):** 共享实体(`setupSync78DB`、`entry78`、`seedPool78`、`mockLDAPClient`、`factoryFixturePath`)显式不跨 plan 重复,同包内复用规则一致。D-78-06e 显式禁止 78-06 重新定义 78-05 的 helper。78-04 经 `depends_on: [78-03]` 引用 78-03's `seedPool78`。无冲突。

**D10 (CLAUDE.md Compliance):** Status Value Convention (0=enabled, 1=disabled) — 78-01/78-05/78-06 显式要求 `models.*` 具名常量禁裸 0/1 ✓. 测试约定(no t.Parallel for shared sqlite, t.Cleanup for goroutines, no real device/凭据入仓) — 每 plan 强制 ✓. D-78-08 文件名后缀 `_78_NN_test.go` — 7 个 plan 全用 ✓.

**D11 (Research Resolution):** 78-RESEARCH.md `## 5 Decision Queue` 列 DQ1-DQ5 — 全 5 项在 plans 中已解决(DQ1→78-02 D-78-01;DQ2→78-04 D-78-02;DQ3→78-02 D-78-03;DQ4→78-07 D-78-04;DQ5→78-03 D-78-05 with enhancement)。G4 (FailoverClient 具体类型) 自我修正被携入 78-07。

**D12 (Pattern Compliance):** SKIPPED — Phase 78 无 PATTERNS.md。已交叉引用 76-02/76-03/76-04/76-05 先例。

---

## Issue List

### HIGH (blocks execution)

**None.**

### MED (should fix; falls back gracefully)

**M-1: 78-01 Task 1 `TestCap78_GenerateCaptcha_FailClose` 设计脆性**
- **Plan:** 78-01
- **Task:** Task 1, `TestCap78_GenerateCaptcha_FailClose`
- **描述:** Test uses `mr.Close()` mid-test then RunT's t.Cleanup 尝试二次 Close;根据 miniredis v2.38+ 行为可能 panic。Plan 已确认并提供 fallback("若 panic 则改用 `miniredis.Run()` + 手动 defer,SUMMARY 记录"),但主路径脆。
- **Severity:** MED — fallback 文档化;实际可能工作(miniredis v2.38 对 Close 幂等);executor 可能需早期切换。
- **Suggested fix (executor-side, not planner-side):** 先试主路径;若 CI 中 panic,切换到 `miniredis.Run()` + 手动 `defer pc.Close()` + `t.Cleanup(func() { _ = pc.Close() })` (对已关闭 `*miniredis.Miniredis` 二次 Close 返回 error 不 panic)。记录切换到 SUMMARY。
- **Why not HIGH:** Plan 显式提供 fallback 路径,失败模式是"cleanup 时 panic"非"测试结果错误"。

### LOW (informational)

- **L-1:** 78-02 Task 1 probe 可能发现未预期 hang;D-78-03 ladder 正确;若 (a) 触发则 Phase 79/80 可能 revisit。By design。
- **L-2:** 78-03 Task 1 fixture 推演 (SendConfig/SendConfigs huawei_vrp privilege elevation) 可能与 scrapligo 字节流不一致。D-78-03d 降级到 error-path coverage。By design。
- **L-3:** 78-04 closeLocked nil-Conn 是 D-78-10 判断点。By design。
- **L-4:** 78-06 isUniqueConstraintError 可能揭示 sqlite 方言 gap。D-78-06c 要求真实 driver error + D-78-10 fallback。By design。
- **L-5:** 78-07 cleanDomain 可疑语义是 D-78-07c 判断点。By design。

---

## Pitfall Coverage Matrix

| Pitfall | Plan | Status |
|---------|------|--------|
| MultiLevelCache goroutine leak (R-7) | 78-02 D-78-01 (Simple + explicit WithWriter test) | ✓ ADDRESSED |
| QUIRK-01 IncrementBy nil-deref | 78-01 Task 1 truth #2 (GenerateCaptcha 真实链路 双分支) | ✓ ADDRESSED |
| QUIRK-07 MetricsCacheService.Stop idempotency | 78-01 Task 5 TestMx78_Stop_Idempotent + 78-02 Task 4 Close_Idempotent | ✓ ADDRESSED(双覆盖) |
| SSH fake prompt after pty-req+shell (S-2) | 78-03 D-78-03a (FileTransport 不调 Close) + fixture design | ✓ ADDRESSED |
| SSH fake auth callback (nil, false) for rejection (S-3) | N/A — plan 用 FileTransport (76-02 模式),无 fake SSH server auth path 触发 | ⚠ NOT APPLICABLE(by plan design 决定) |
| createConnection SSH coupling (R7) | 78-03 D-78-05 (pre-seed pool.connections[deviceID]) — third path beyond DQ5 (a)/(b) | ✓ ADDRESSED with enhancement |
| FailoverClient interface already in place (G4 RESOLVED) | 78-07 G4 RESOLVED explicit | ✓ ADDRESSED |
| snmp UDP fake scope decision | 78-04 D-78-06 (real UDP listener + gosnmp codec) + D-78-02 lightweight/fallback | ✓ ADDRESSED |
| LDAP responder as probe + fallback | 78-07 D-78-04 (asn1-ber, zero new module) | ✓ ADDRESSED |

**8/9 ADDRESSED, 1/9 NOT APPLICABLE(by plan design)。可接受。**

---

## Production Code Change Audit

| Plan | Permitted Production Change | Trigger |
|------|---------------------------|---------|
| 78-01 | D-78-10 quirk only | Executor source-vs-docs mismatch |
| 78-02 | D-78-03 (a) Close order / D-78-10 quirk | Init probe 揭示 hang |
| 78-03 | **ZERO (D-78-05 forbids createConnection seam)** | N/A — zero drift |
| 78-04 | D-78-10 quirk only (closeLocked nil guard likely) | Executor judgment |
| 78-05 | **ZERO (D-78-07 REJECTS seam)** | N/A — zero drift |
| 78-06 | D-78-10 quirk only | Executor source-vs-docs mismatch |
| 78-07 | D-78-10 quirk only (cleanDomain fix likely) | Executor judgment |

**Zero-drift plans:** 78-03 and 78-05。Overall: ✓ ACCEPTABLE — 匹配 Phase 77 纪律("production .go changes = 0 default with D-78-10 escape hatch")。

---

## Dependency Graph

```
         ┌─────────────┐
         │  78-01 core │  wave 1
         │  captcha+   │
         │  background │
         │  metrics    │
         └──────┬──────┘
                │ depends_on
                ▼
         ┌─────────────┐
         │  78-02 core │  wave 2
         │  Init/Close │
         │  装配链     │
         └─────────────┘

         ┌─────────────┐
         │ 78-03 device│  wave 1
         │ scrapli三大 │
         └──────┬──────┘
                ▼
         ┌─────────────┐
         │ 78-04 device│  wave 2
         │ snmp+tasks  │
         └─────────────┘

         ┌─────────────┐
         │ 78-05 addom │  wave 1
         └──────┬──────┘
                ▼
         ┌─────────────┐
         │ 78-06 addom │  wave 2
         └──────┬──────┘
                ▼
         ┌─────────────┐
         │ 78-07 addom │  wave 3
         └─────────────┘
```

Cross-package parallelism: wave 1: 78-01 ∥ 78-03 ∥ 78-05。Wave 2: 78-02 ∥ 78-04 ∥ 78-06。Wave 3: 78-07 独立。

---

## Suggested Iteration Plan

**No planner invocation needed.** 7 个 plan 结构上交付其切片。单一 MED issue (M-1: 78-01 TestCap78_GenerateCaptcha_FailClose) 是文档化 fallback 路径供 executor inline 处理,非 planner-side defect。

若 orchestrator 阈值不允许 PASS-WITH-FIXES,唯一 planner-side 改动将是:
- (A) 优化 78-01 Task 1 action,使用 `miniredis.Run()` + 手动 defer 作主路径(非 fallback) — 实现细节,非 plan defect
- (B) 在 78-01 acceptance_criteria 加一行注:"fallback to miniredis.Run() + 手动 defer 可接受 if mr.Close() during RunT 触发 double-close panic"

任一改均小修(<30 秒)。

**Recommendation:** 准备路由到 `/gsd:execute-phase 78`。MED issue 由 executor 经文档化 fallback 路径处理。

---

## File Locations

- Plans under verification: `D:\CODE\ClaudeCode\guoguo\.planning\workstreams\milestone\phases\78-block-bp-unlock-by-foundation\78-0{1..7}-PLAN.md`
- Research artifact: `D:\CODE\ClaudeCode\guoguo\.planning\workstreams\milestone\phases\78-block-bp-unlock-by-foundation\78-RESEARCH.md`
- Roadmap (SC#1-5 source): `D:\CODE\ClaudeCode\guoguo\.planning\workstreams\milestone\ROADMAP.md` L183-216
- Requirements (BLOCK-03/04/05 source): `D:\CODE\ClaudeCode\guoguo\.planning\workstreams\milestone\REQUIREMENTS.md` L28-34
- Verification exemplar (format benchmark): `D:\CODE\ClaudeCode\guoguo\.planning\workstreams\milestone\phases\76-test-doubles\76-VERIFICATION.md`

---

**VERIFICATION overall: PASS-WITH-FIXES;HIGH: 0;MED: 1;LOW: 5**
