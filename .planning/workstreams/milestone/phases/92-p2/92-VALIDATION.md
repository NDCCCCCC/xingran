---
phase: 92
slug: p2
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-05
---

# Phase 92 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `92-RESEARCH.md` § Validation Architecture (2026-09-05, HIGH confidence).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test（标准）+ testify + miniredis/v2.38.0 + glebarez/sqlite |
| **Config file** | none（go.mod 驱动；无独立 test config） |
| **Quick run command** | `go build ./... && go test ./internal/services/base/... ./internal/services/system/... ./internal/services/operations/...` |
| **Full suite command** | `go test ./internal/services/...`（SC-4）/ 全量 `go test ./...` |
| **Estimated runtime** | ~60-120 seconds（三包 quick）/ 全量 ~5-8 minutes |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...` + quick 命令（base + system + operations 三包）
- **After every plan wave:** Run `go test ./internal/services/...`（SC-4 口径）
- **Before `/gsd:verify-work`:** Full suite must be green（全量 `go test ./...` + operlog/status AST 锁值防线保持绿）
- **Max feedback latency:** ~120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 92-01 T1 | 01 | 1 | CACHE-UNIFY-01 | T-92-01（缓存键注入，本 task 键构造零触碰） | 抽象层 compile gate（行为锁在 92-01 T3） | compile gate（go build + go vet） | `go build ./internal/services/base/ && go vet ./internal/services/base/` | ❌ W0（T1 创建三生产文件） | ⬜ pending |
| 92-01 T3 | 01 | 1 | CACHE-UNIFY-05 | T-92-02（失效遗漏） | pattern+key 失效 + NoOp 透传 | integration（miniredis） | `go test ./internal/services/base/ -run TestBase92 -v` | ❌ W0（T3 创建 cache_service_base_test.go） | ⬜ pending |
| 92-02 T1-T3 | 02 | 2 | CACHE-UNIFY-02 | T-92-01/T-92-03 | 行为不变（29 处迁移） | regression | `go test ./internal/services/system/ -run "Cache" -v && go test ./internal/services/system/` | ✅ | ⬜ pending |
| 92-03 T1 | 03 | 3 | CACHE-UNIFY-03 | T-92-01/T-92-02 | 分发语义不变 | regression + unit | `go test ./internal/services/operations/ -run "Cache\|Floor" -v` | ✅ | ⬜ pending |
| 92-03 T3 | 03 | 3 | CACHE-UNIFY-04 | T-92-02 | GetExpiration 委托行为不变 | regression | `go test ./internal/services/ -run "Dcs7901\|Ccs7901" -v` | ✅ | ⬜ pending |
| 92-04 T2 | 04 | 4 | D-10② | T-92-01 | 无 interface{} 闭包式 GetOrSet 残留 | AST/regex 扫描（warning 不 fail） | `go test ./internal/services/system/ -run TestNoInterfaceGetOrSetResidue -v` | ❌ W0（T2 创建 cache_invariants_92_test.go） | ⬜ pending |
| 92-04 T4 | (全) | 4 | SC-4 | — | 1688+ 测试 0 回归 | full regression | `go test ./internal/services/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*2026-09-05 revision 回填：Task ID 列已绑定 92-01..04 实际 task（checker warning 3 收口）；Wave-0 两文件分别由 92-01 T3 与 92-04 T2 承接，行内 File Exists 标注 ❌ W0 及承接 task。*

---

## Wave 0 Requirements

- [ ] `internal/services/base/cache_service_base_test.go` — 覆盖 CACHE-UNIFY-01/05：泛型函数族 hit/miss/TTL(FastForward)/失效/NoOp 透传/GetStats + nil-config 分支（REQUIREMENTS 原文命名的文件）（由 92-01 T3 承接）
- [ ] （可选并入上文件）base 包 alias 翻转后 `var _` 编译期断言（由 92-01 T2 alias 块内建：system/cache_provider.go 断言行随 Task 2 落地）
- [ ] invariants 扫描测试（system 或 pkg 侧，warning-only，Phase 89/90 模式）（由 92-04 T2 承接）
- [ ] Framework install: 无需（全部既有）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (none) | — | D-09 决策：SC-5 端到端验证以 miniredis 自动化落地，无强制手动项 | — |

*All phase behaviors have automated verification (per D-09).*

---

## Phase Gate（verify 前置）

- 全量 `go test ./...` 绿
- `internal/utils/operlog/regression_test.go` + `internal/models/status_constants_test.go` AST 锁值防线保持绿
- LOC numstat 双口径报告（样板毛减口径 + repo 全口径，沿 OVR-91-01 先例）
- invariants 扫描输出：interface{} 闭包式 GetOrSet 残留 = 0

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references（Wave-0 文件待 92-01 T3 / 92-04 T2 执行落地后勾选）
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
