---
phase: 103
slug: cache-closure-base-authority
status: draft
nyquist_compliant: false
wave_0_complete: true
created: 2026-09-07
---

# Phase 103 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 本相为**行为等价重构**——validation 的核心是"迁前迁后语义等价"，不是新功能验收。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `testify` |
| **Config file** | none — 标准 `go test` |
| **Quick run command** | `go test ./internal/services/... -count=1` |
| **Full suite command** | `go build ./... && go test ./internal/services/...` |
| **Estimated runtime** | ~120 秒 |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/... -count=1`
- **After every plan wave:** Run `go build ./... && go test ./internal/services/...`
- **Before `/gsd:verify-work`:** Full suite must be green + 七 gate 不倒退（后端 coverage ≥78.33%）
- **Max feedback latency:** 120 秒

---

## Per-Task Verification Map

> Task IDs 由 planner 分配后回填；下表按 requirement × 验证维度锁定断言面。

| Req | Domain | Wave | Verification Dimension | Test Type | Automated Command | File Exists | Status |
|-----|--------|------|------------------------|-----------|-------------------|-------------|--------|
| CONV-01 | mac_history query :309/:434/:837 | 1 | 严格语义（D-103-6）：cache 写失败返回 error | unit | `go test ./internal/services/ -run "MACHistory" -count=1` | ✅ | ⬜ pending |
| CONV-01 | mac_history query :309/:434/:837 | 1 | TTL 等价：`s.perfCacheTTL()` 路径不变 | unit | `go test ./internal/services/ -run "MACHistory" -count=1` | ✅ | ⬜ pending |
| CONV-01 | mac_history query :309/:434/:837 | 1 | 命中/回源/回填时序等价 | unit | `go test ./internal/services/ -run "MACHistory" -count=1` | ✅ | ⬜ pending |
| CONV-01 | mac_history vendor :259-281 | 1 | "Unknown Vendor" 占位保留（D-103-5） | unit | `go test ./internal/services/ -run "Vendor" -count=1` | ✅ | ⬜ pending |
| CONV-01 | mac_history vendor :259-281 | 1 | TTL `24*time.Hour` 字面量等价 | unit | `go test ./internal/services/ -run "Vendor" -count=1` | ✅ | ⬜ pending |
| CONV-01 | mac_history heatmap :118 | 1 | fallback 直查语义（D-103-9）：cache 失败降级 `queryHeatmapFromMV` + warn | unit | `go test ./internal/services/ -run "Heatmap" -count=1` | ✅ | ⬜ pending |
| CONV-02 | asset reconciliation :799-820 | 1 | warn-on-set 不阻断（D-103-7）：set 失败仍返回 cached | unit | `go test ./internal/services/asset/ -run "Reconciliation" -count=1` | ✅ | ⬜ pending |
| CONV-02 | asset reconciliation :799-820 | 1 | `Workstation.ID != ""` 脏缓存防御保留 | unit | `go test ./internal/services/asset/ -run "Reconciliation" -count=1` | ✅ | ⬜ pending |
| CONV-02 | asset reconciliation :799-820 | 1 | TTL `constants.reconciliationHealthCacheTTL` 等价 | unit | `go test ./internal/services/asset/ -run "Reconciliation" -count=1` | ✅ | ⬜ pending |
| CONV-03 | rpa selector :169-174/:226 | 1 | best-effort 静默（D-103-8）：错误返回 nil 视为 miss | unit | `go test ./internal/services/rpa/ -run "Selector" -count=1` | ✅ | ⬜ pending |
| CONV-03 | rpa selector :169-174/:226 | 1 | TTL `30*time.Minute` 字面量等价（D-103-12） | unit | `go test ./internal/services/rpa/ -run "Selector" -count=1` | ✅ | ⬜ pending |
| CONV-04 | invariants 扩口 | 2 | 4 文件 interface{} 闭包 GetOrSet 计数 == 0（硬失败档 D-103-17） | AST scan | `go test ./internal/services/system/ -run "GetOrSetResidue" -count=1 -v` | ✅ | ⬜ pending |
| CONV-04 | invariants 扩口 | 2 | 手写 cache-aside 模式计数 == 0（D-103-18） | AST scan | `go test ./internal/services/system/ -run "CacheAside" -count=1 -v` | ❌ W0 | ⬜ pending |
| CONV-01..03 | caller audit（D-103-20） | 1 | constructor 签名变更全调用点编译通过 | build | `go build ./...` | ✅ | ⬜ pending |
| CONV-01..03 | cache key 等价（D-103-21） | 1 | 迁后 key 字符串与迁前逐处一致 | unit | `go test ./internal/services/... -count=1` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

现有测试基础设施覆盖三域既有面，无需安装框架：

- `internal/services/mac_history_query_service_79_05_test.go` — mac_history query service
- `internal/services/mac_history_tail_79_05_test.go` — heatmap service
- `internal/services/asset/reconciliation_sqlite_runtime_test.go` — reconciliation
- `internal/services/asset/asset_gapfill_test.go` — asset gapfill + reconciliation 集成
- `internal/services/rpa/ai_selector_excel_test.go` — selector_learner
- `internal/services/system/cache_invariants_92_test.go` — AST 扫描宿主（CONV-04 扩口目标）

**唯一 Wave 0 缺口（由本相自身交付，非前置安装）：**

- [ ] 手写 cache-aside AST 检测断言（`cache.Get` → `json.Unmarshal` → `cache.Set` + `json.Marshal` 序列模式）— D-103-18 新增，无现存实现
- [ ] fake/stub `base.CacheProvider`（可注入 Get/Set 失败）— 4 种错误语义等价测试的共享 fixture；若三域现有测试已有等价 mock 则复用，否则本相新增

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 七 gate 不倒退（后端 coverage ≥78.33%） | v1.31 D-02 | 需跨 gate 汇总实测落盘，非单测断言 | 依次跑 `go build ./...` / `go test ./internal/services/...` / `go test -cover ./...` 对照 78.33% 基线 / 前端 45 dirs / lint / type-check / diff coverage，结果落盘 |
| caller audit 清单落盘 | D-103-20 | 清单完整性需人工核对 grep 结果 vs 实际改动 | grep 四构造函数全调用点，逐点确认已切 `base.CacheProvider`，清单写入 SUMMARY |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references（手写 cache-aside AST 断言 + fake CacheProvider）
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
